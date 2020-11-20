package httplog

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/smallstep/logging"
	"go.uber.org/zap"
)

type options struct {
	Redactors []RedactorFunc
}

// Option is the type used to modify logger options.
type Option func(o *options)

func (o *options) apply(opts []Option) *options {
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// RedactorFunc is a function that redacts the HTTP request or response
// logged.
type RedactorFunc func(w http.ResponseWriter, r *http.Request)

// WithRedactor is an option that adds a new redactor to a httplog.
func WithRedactor(fn RedactorFunc) Option {
	return func(o *options) {
		o.Redactors = append(o.Redactors, fn)
	}
}

// LoggerHandler creates a logger handler
type LoggerHandler struct {
	*logging.Logger
	name         string
	next         http.Handler
	logRequests  bool
	logResponses bool
	timeFormat   string
	options      *options
}

// Middleware returns the given http.Handler with the logger integrated.
func Middleware(logger *logging.Logger, next http.Handler, opts ...Option) http.Handler {
	o := new(options).apply(opts)
	h := logging.Tracing(logger.TraceHeader())
	return h(&LoggerHandler{
		Logger:       logger,
		name:         logger.Name(),
		next:         next,
		logRequests:  logger.LogRequests(),
		logResponses: logger.LogResponses(),
		timeFormat:   logger.TimeFormat(),
		options:      o,
	})
}

// ServeHTTP implements the http.Handler and call to the handler to log with a
// custom http.ResponseWriter that records the response code and the number of
// bytes sent.
func (l *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rw ResponseLogger
	t := time.Now()
	if l.logResponses {
		rw = NewRawResponseLogger(w)
	} else {
		rw = NewResponseLogger(w)
	}
	if l.logRequests {
		if rf, err := NewRequest(r); err == nil {
			rw.WithField(RequestKey, rf)
		}
	}
	// Serve next http handler.
	l.next.ServeHTTP(rw, r)
	d := time.Since(t)
	// Redact request and response if configured.
	for _, redactor := range l.options.Redactors {
		redactor(rw, r)
	}
	// Write logs.
	l.writeEntry(rw, r, t, d)
}

// writeEntry writes to the Logger writer the request information in the logger.
func (l *LoggerHandler) writeEntry(w ResponseLogger, r *http.Request, t time.Time, d time.Duration) {
	ctx := r.Context()
	var requestID, tracingID string
	if tp, ok := logging.GetTraceparent(ctx); ok {
		requestID = tp.TraceID()
		tracingID = tp.String()
	}

	// Remote hostname
	addr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		addr = r.RemoteAddr
	}

	// From https://github.com/gorilla/handlers
	uri := r.RequestURI
	// Requests using the CONNECT method over HTTP/2.0 must use
	// the authority field (aka r.Host) to identify the target.
	// Refer: https://httpwg.github.io/specs/rfc7540.html#CONNECT
	if r.ProtoMajor == 2 && r.Method == "CONNECT" {
		uri = r.Host
	}
	if uri == "" {
		uri = r.URL.RequestURI()
	}

	status := w.StatusCode()

	fields := []zap.Field{
		zap.String("name", l.name),
		zap.String("system", "http"),
		zap.String("request-id", requestID),
		zap.String("tracing-id", tracingID),
		zap.String("remote-address", addr),
		zap.String("time", t.Format(l.timeFormat)),
		zap.Duration("duration", d),
		zap.Int64("duration-ns", d.Nanoseconds()),
		zap.String("method", r.Method),
		zap.String("path", uri),
		zap.String("protocol", r.Proto),
		zap.Int("status", status),
		zap.Int("size", w.Size()),
		zap.String("referer", r.Referer()),
		zap.String("user-agent", r.UserAgent()),
	}

	// Add request added in the middleware.
	if r, ok := w.Request(); ok {
		fields = append(fields, zap.Object(RequestKey, r))
	}

	// Add request from response logger.
	if rw, ok := w.(RawResponseLogger); ok {
		fields = append(fields, zap.Object(ResponseKey, &Response{
			Headers: Headers(rw.Header()),
			Body:    rw.Response(),
		}))
	}

	// Add error if present.
	if v, ok := w.Field(ErrorKey); ok {
		if err, ok := v.(error); ok {
			fields = append(fields, zap.Error(err))
		} else {
			fields = append(fields, zap.Any(ErrorKey, err))
		}
	}

	var message string
	if v, ok := w.Field(MessageKey); ok {
		if message, ok = v.(string); !ok {
			message = fmt.Sprint(v)
		}
	}

	// Add custom fields. Disabled for the moment.
	// for k, v := range w.Fields() {
	// 	fields = append(fields, zap.Any(k, v))
	// }

	switch {
	case status < http.StatusBadRequest:
		l.Info(message, fields...)
	case status < http.StatusInternalServerError:
		l.Warn(message, fields...)
	default:
		l.Error(message, fields...)
	}
}
