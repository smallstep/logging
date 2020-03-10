package grpclog

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/golang/protobuf/proto"
	"github.com/smallstep/logging"
	"github.com/smallstep/logging/tracing"
)

type interceptorType int

const (
	unaryType interceptorType = iota
	streamType
)

// TracingContext gets the tracing id from the context metadata or generates a
// new one and appends it to the context.
func TracingContext(ctx context.Context, traceHeader string) context.Context {
	var err error
	var tp *tracing.Traceparent

	// Parse traceparent if available. Ignore errors.
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if v, ok := md[traceHeader]; ok {
			tp, _ = tracing.Parse(v[0])
		}
	} else {
		md = metadata.MD{}
	}

	// If no traceparent or bad, generate a new one.
	// Do not fail if we can't generate the tracing id.
	if tp == nil {
		tp, err = logging.NewTraceparent()
		if err != nil {
			return ctx
		}
	}

	// Generate new metadata and context with the tracing id.
	newMD := metadata.Join(md, metadata.Pairs(traceHeader, tp.String()))
	ctx = metadata.NewIncomingContext(ctx, newMD)
	return logging.WithTraceparent(ctx, tp)
}

// UnaryServerInterceptor returns a new unary server interceptors for logging
// unary requests.
func UnaryServerInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	l := newServerLogger(logger, unaryType)
	traceHeader := strings.ToLower(logger.TraceHeader())

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		t1 := time.Now()

		// Get or set the traceparent
		ctx = TracingContext(ctx, traceHeader)

		fields := []zap.Field{}
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, zap.String("grpc.request.deadline", d.Format(time.RFC3339)))
		}

		// Call handler
		resp, err := handler(ctx, req)
		duration := time.Since(t1)

		if logger.LogRequests() {
			if p, ok := req.(proto.Message); ok {
				fields = append(fields, zap.Object("grpc.request.content", &jsonpbObjectMarshaler{pb: p}))
			}
		}
		if err == nil && logger.LogResponses() {
			if p, ok := resp.(proto.Message); ok {
				fields = append(fields, zap.Object("grpc.response.content", &jsonpbObjectMarshaler{pb: p}))
			}
		}

		// Write log
		l.Log(ctx, info.FullMethod, t1, duration, fields, err)

		return resp, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor for
// logging stream requests.
func StreamServerInterceptor(logger *logging.Logger) grpc.StreamServerInterceptor {
	l := newServerLogger(logger, streamType)
	traceHeader := strings.ToLower(logger.TraceHeader())

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		t1 := time.Now()

		// Get or set the traceparent
		ctx := TracingContext(stream.Context(), traceHeader)

		fields := []zap.Field{}
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, zap.String("grpc.request.deadline", d.Format(time.RFC3339)))
		}

		// Create stream logger and wrap stream with the new context
		wrapped := newServerStream(ctx, info.FullMethod, stream, l)

		// Call handler
		err := handler(srv, wrapped)
		duration := time.Since(t1)

		// Write log
		l.Log(ctx, info.FullMethod, t1, duration, fields, err)

		return err
	}
}
