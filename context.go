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
			tracingID := req.Header.Get(headerName)
			if tracingID == "" {
				traceparent, err := tracing.New()
				if err != nil {
					// do not fail if we can't generate the tracing id
					return
				}
				tracingID = traceparent.String()
				req.Header.Set(headerName, tracingID)
			}
			ctx := WithTraceparent(req.Context(), tracingID)
			next.ServeHTTP(w, req.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// WithTraceparent returns a new context with the given tracing id added to the
// context.
func WithTraceparent(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceheaderKey, id)
}

// GetTracing returns the tracing id from the context if it exists.
func GetTraceparent(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(traceheaderKey).(string)
	return v, ok
}
