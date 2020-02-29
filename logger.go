package logging

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// defaultTraceIdHeader is the default header used as a trace id.
const defaultTraceIDHeader = "X-Trace-ID"

// Logger is a request logger that uses zap.Logger as core.
type Logger struct {
	*zap.Logger
	name    string
	options *options
}

// New initializes the logger with the given options.
func New(name string, opts ...Option) (*Logger, error) {
	o := defaultOptions()
	if err := o.apply(opts); err != nil {
		return nil, err
	}

	config := zap.NewProductionConfig()

	switch strings.ToLower(o.Format) {
	// case "", "text":
	// TODO
	case "json":
		config.Encoding = "json"
	// case "common":
	// TODO
	default:
		return nil, errors.Errorf("unsupported logger.format '%s'", o.Format)
	}

	base, err := config.Build()
	if err != nil {
		return nil, errors.Wrap(err, "error creating logger")
	}

	return &Logger{
		Logger:  base,
		name:    name,
		options: o,
	}, nil
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// GetTraceHeader returns the trace header configured
func (l *Logger) GetTraceHeader() string {
	if l.options.TraceHeader == "" {
		return defaultTraceIDHeader
	}
	return l.options.TraceHeader
}

// Middleware returns the logger middleware that will trace the request of the
// given handler.
func (l *Logger) Middleware(next http.Handler) http.Handler {
	return NewLoggerHandler(l.name, l, next)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}
