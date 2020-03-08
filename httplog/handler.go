package httplog

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/smallstep/logging"
	"go.uber.org/zap"
)

// LoggerHandler creates a logger handler
type LoggerHandler struct {
	*logging.Logger
	name         string
	next         http.Handler
	logRequests  bool
	logResponses bool
	timeFormat   string
}

// NewLoggerHandler returns the given http.Handler with the logger integrated.
func Middleware(logger *logging.Logger, next http.Handler) http.Handler {
	h := logging.Tracing(logger.TraceHeader())
	return h(&LoggerHandler{
		Logger:       logger,
		name:         logger.Name(),
		next:         next,
		logRequests:  logger.LogRequests(),
		logResponses: logger.LogResponses(),
		timeFormat:   logger.TimeFormat(),
	})
}

// ServeHTTP implements the http.Handler and call to the handler to log with a
// custom http.ResponseWriter that records the response code and the number of
// bytes sent.
func (l *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rw ResponseLogger
	var f []zap.Field
	t := time.Now()
	if l.logRequests {
		if rf, err := NewRequest(r); err == nil {
			f = append(f, zap.Object("request", rf))
		}
	}
	if l.logResponses {
		rw = NewRawResponseLogger(w)
	} else {
		rw = NewResponseLogger(w)
	}
	l.next.ServeHTTP(rw, r)
	d := time.Since(t)
	l.writeEntry(rw, r, t, d, f)
}

// writeEntry writes to the Logger writer the request information in the logger.
func (l *LoggerHandler) writeEntry(w ResponseLogger, r *http.Request, t time.Time, d time.Duration, extraFields []zap.Field) {
	ctx := r.Context()
	reqID, _ := logging.GetTraceparent(ctx)

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
		zap.String("request-id", reqID),
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

	fields = append(fields, extraFields...)

	if rw, ok := w.(RawResponseLogger); ok {
		fields = append(fields, zap.Object("response", &Response{
			Headers: Headers(rw.Header()),
			Body:    rw.Response(),
		}))
	}

	var message string
	v, ok := w.Field("message")
	if ok {
		if message, ok = v.(string); !ok {
			message = fmt.Sprint(v)
		}
	}

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
