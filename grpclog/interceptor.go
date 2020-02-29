package grpclog

import (
	"context"
	"crypto/tls"
	"path"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/smallstep/logging"
)

type interceptorType int

const (
	unaryType interceptorType = iota
	streamType
)

// UnaryServerInterceptor returns a new unary server interceptors for logging
// unary requests.
func UnaryServerInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	traceHeader := strings.ToLower(logger.TraceHeader())
	timeFormat := logger.TimeFormat()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var requestID string
		t1 := time.Now()

		// Get or set request id
		ctx, requestID = getRequestID(ctx, traceHeader)

		fields := []zap.Field{}
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, zap.String("grpc.request.deadline", d.Format(time.RFC3339)))
		}

		// Call handler
		resp, err := handler(ctx, req)
		duration := time.Since(t1)
		startTime := t1.Format(timeFormat)

		// Write log
		writeLog(ctx, logger, unaryType, requestID, info.FullMethod, startTime, duration, fields, err)

		return resp, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor for
// logging stream requests.
func StreamServerInterceptor(logger *logging.Logger) grpc.StreamServerInterceptor {
	traceHeader := strings.ToLower(logger.TraceHeader())
	timeFormat := logger.TimeFormat()

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		t1 := time.Now()

		// Get or set request id
		ctx, requestID := getRequestID(stream.Context(), traceHeader)

		fields := []zap.Field{}
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, zap.String("grpc.request.deadline", d.Format(time.RFC3339)))
		}

		// Wrap stream with the new context
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = ctx

		// Call handler
		err := handler(srv, wrapped)
		duration := time.Since(t1)
		startTime := t1.Format(timeFormat)

		// Write log
		writeLog(ctx, logger, streamType, requestID, info.FullMethod, startTime, duration, fields, err)

		return err
	}
}

// getRequestID get the requestID from the context metadata or generates a new
// one and appends it to the context.
func getRequestID(ctx context.Context, traceHeader string) (context.Context, string) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v, ok := md[traceHeader]; ok {
			return logging.WithRequestID(ctx, v[0]), v[0]
		}
		requestID := logging.NewRequestID()
		newMD := metadata.Join(md, metadata.Pairs(traceHeader, requestID))
		return logging.WithRequestID(metadata.NewIncomingContext(ctx, newMD), requestID), requestID
	}

	requestID := logging.NewRequestID()
	newMD := metadata.Pairs(traceHeader, requestID)
	return logging.WithRequestID(metadata.NewIncomingContext(ctx, newMD), requestID), requestID
}

func writeLog(ctx context.Context, logger *logging.Logger, typ interceptorType, requestID, fullMethod string, startTime string, duration time.Duration, extra []zap.Field, grpcErr error) {
	var pkg string
	code := grpc.Code(grpcErr)
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)
	parts := strings.Split(service, ".")
	if l := len(parts); l > 1 {
		pkg = strings.Join(parts[:l-1], ".")
		service = parts[l-1]
	}

	fields := []zap.Field{
		zap.String("system", "grpc"),
		zap.String("span.kind", "server"),
		zap.String("grpc.package", pkg),
		zap.String("grpc.service", service),
		zap.String("grpc.method", method),
		zap.String("grpc.code", code.String()),
		zap.String("grpc.request.id", requestID),
		zap.String("grpc.start_time", startTime),
		zap.Duration("grpc.durations", duration),
		zap.Int64("grpc.duration-ns", duration.Nanoseconds()),
	}

	if len(extra) > 0 {
		fields = append(fields, extra...)
	}

	if peer, ok := peer.FromContext(ctx); ok {
		fields = append(fields, zap.String("peer.address", peer.Addr.String()))
		if s, ok := getPeerIdentity(peer); ok {
			fields = append(fields, zap.String("peer.identity", s))
		}
	}

	if grpcErr != nil {
		fields = append(fields, zap.Error(grpcErr))
	}

	var msg string
	switch typ {
	case unaryType:
		msg = "finished unary call with code " + code.String()
	case streamType:
		msg = "finished streaming call with code " + code.String()
	}

	switch codeToLevel(code) {
	case zapcore.InfoLevel:
		logger.Info(msg, fields...)
	case zapcore.WarnLevel:
		logger.Warn(msg, fields...)
	default:
		logger.Error(msg, fields...)
	}
}

// codeToLevel returns the log level to use for a given gRPC return code.
func codeToLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zapcore.InfoLevel
	case codes.Canceled:
		return zapcore.InfoLevel
	case codes.Unknown:
		return zapcore.ErrorLevel
	case codes.InvalidArgument:
		return zapcore.InfoLevel
	case codes.DeadlineExceeded:
		return zapcore.WarnLevel
	case codes.NotFound:
		return zapcore.InfoLevel
	case codes.AlreadyExists:
		return zapcore.InfoLevel
	case codes.PermissionDenied:
		return zapcore.WarnLevel
	case codes.Unauthenticated:
		return zapcore.InfoLevel
	case codes.ResourceExhausted:
		return zapcore.WarnLevel
	case codes.FailedPrecondition:
		return zapcore.WarnLevel
	case codes.Aborted:
		return zapcore.WarnLevel
	case codes.OutOfRange:
		return zapcore.WarnLevel
	case codes.Unimplemented:
		return zapcore.ErrorLevel
	case codes.Internal:
		return zapcore.ErrorLevel
	case codes.Unavailable:
		return zapcore.WarnLevel
	case codes.DataLoss:
		return zapcore.ErrorLevel
	default:
		return zapcore.ErrorLevel
	}
}

func getPeerIdentity(p *peer.Peer) (string, bool) {
	if p.AuthInfo == nil {
		return "", false
	}
	switch p.AuthInfo.AuthType() {
	case "tls":
		if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			return getCommonName(tlsInfo.State)
		}
	}

	return "", false
}

func getCommonName(cs tls.ConnectionState) (string, bool) {
	if len(cs.PeerCertificates) == 0 || cs.PeerCertificates[0] == nil {
		return "", false
	}
	return cs.PeerCertificates[0].Subject.CommonName, true
}
