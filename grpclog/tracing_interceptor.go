package grpclog

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/smallstep/logging"
	"github.com/smallstep/logging/tracing"
)

// TracingClientContext gets the tracing id from the context or generates a
// new one and appends it to the outgoing context
func TracingClientContext(ctx context.Context, traceHeader string) context.Context {
	tp, ok := logging.GetTraceparent(ctx)
	if !ok {
		tp, _ = tracing.New()
	}
	// On errors, return context as it is.
	if tp == nil {
		return ctx
	}
	// Add traceparent to the outgoing context
	return metadata.AppendToOutgoingContext(ctx, traceHeader, tp.String())
}

// TracingUnaryClientInterceptor appends the tracing header to the metadata.
func TracingUnaryClientInterceptor(traceHeader string) grpc.UnaryClientInterceptor {
	traceHeader = strings.ToLower(traceHeader)
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Append tracing header
		ctx = TracingClientContext(ctx, traceHeader)

		// Invoke method.
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// TracingStreamClientInterceptor appends the tracing header to the metadata.
func TracingStreamClientInterceptor(traceHeader string) grpc.StreamClientInterceptor {
	traceHeader = strings.ToLower(traceHeader)
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Append tracing header
		ctx = TracingClientContext(ctx, traceHeader)

		// Invoke method
		return streamer(ctx, desc, cc, method, opts...)
	}
}
