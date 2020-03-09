package logging

import (
	"context"
	"net/http"

	"github.com/smallstep/logging/tracing"
)

type key int

// traceheaderKey is the context key that should store the request identifier.
const traceheaderKey key = iota

// Tracing returns a new middleware that gets the given header and sets it in
// the context so it can be written in the logger. If the header does not exists
// or it's the empty string, it uses github.com/smallstep/tracing to create a
// new one.
func Tracing(headerName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			var err error
			var tp *tracing.Traceparent
			// Parse traceparent if available. Ignore errors.
			if s := req.Header.Get(headerName); s != "" {
				tp, _ = tracing.Parse(s)
			}
			// If no traceparent or bad, generate a new one.
			// Do not fail if we can't generate the tracing id.
			if tp == nil {
				if tp, err = tracing.New(); err != nil {
					return
				}
				req.Header.Set(headerName, tp.String())
			}
			ctx := WithTraceparent(req.Context(), tp)
			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// NewTraceparent generates a new traceparent.
func NewTraceparent() (*tracing.Traceparent, error) {
	return tracing.New()
}

// WithTraceparent returns a new context with the given tracing id added to the
// context.
func WithTraceparent(ctx context.Context, tp *tracing.Traceparent) context.Context {
	return context.WithValue(ctx, traceheaderKey, id)
}

// GetTracing returns the tracing id from the context if it exists.
func GetTraceparent(ctx context.Context) (*tracing.Traceparent, bool) {
	v, ok := ctx.Value(traceheaderKey).(*tracing.Traceparent)
	return v, ok
}
