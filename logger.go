package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/smallstep/logging/encoder"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level indicates the log level.
type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// DefaultTraceHeader is the default header used as a trace id.
const DefaultTraceHeader = "Traceparent"

// Logger is a request logger that uses zap.Logger as core.
type Logger struct {
	*zap.Logger
	name    string
	options *options
}

type writer struct {
	*zap.Logger
	Name  zap.Field
	Level Level
}

func (w *writer) Write(b []byte) (int, error) {
	switch w.Level {
	case DebugLevel:
		w.Debug(string(b), w.Name)
	case InfoLevel:
		w.Info(string(b), w.Name)
	case WarnLevel:
		w.Warn(string(b), w.Name)
	case ErrorLevel:
		w.Error(string(b), w.Name)
	case FatalLevel:
		w.Fatal(string(b), w.Name)
	default:
		w.Info(string(b), w.Name)
	}
	return len(b), nil
}

// New initializes the logger with the given options.
func New(name string, opts ...Option) (*Logger, error) {
	o := defaultOptions()
	if err := o.apply(opts); err != nil {
		return nil, err
	}

	config := zap.NewProductionEncoderConfig()

	var outEncoder, errEncoder zapcore.Encoder
	switch strings.ToLower(o.Format) {
	case "", "text":
		outEncoder = encoder.NewTextEncoder(config)
		errEncoder = encoder.NewTextEncoder(config)
	case "json":
		outEncoder = zapcore.NewJSONEncoder(config)
		errEncoder = zapcore.NewJSONEncoder(config)
	case "common":
		outEncoder = encoder.NewCLFEncoder(config)
		errEncoder = encoder.NewCLFEncoder(config)
	default:
		return nil, errors.Errorf("unsupported logger.format '%s'", o.Format)
	}

	// Logs info and debug to stdout
	outWriter := zapcore.Lock(os.Stdout)
	outLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel
	})

	// Logs warning and errors to stderr
	errWriter := zapcore.Lock(os.Stderr)
	errLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	})

	// Create zap.Logger
	logger := zap.New(zapcore.NewTee(
		zapcore.NewCore(outEncoder, outWriter, outLevel),
		zapcore.NewCore(errEncoder, errWriter, errLevel),
	)).WithOptions(zap.AddCallerSkip(o.CallerSkip))

	return &Logger{
		Logger:  logger,
		name:    name,
		options: o,
	}, nil
}

// Clone creates a new copy of the logger with the given options.
func (l *Logger) Clone(opts ...zap.Option) *Logger {
	return &Logger{
		Logger:  l.Logger.WithOptions(opts...),
		name:    l.name,
		options: l.options,
	}
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Name returns the logging name.
func (l *Logger) Name() string {
	return l.name
}

// TraceHeader returns the trace header configured.
func (l *Logger) TraceHeader() string {
	if l.options.TraceHeader == "" {
		return DefaultTraceHeader
	}
	return l.options.TraceHeader
}

// LogRequests returns if the logging of requests is enabled.
func (l *Logger) LogRequests() bool {
	return l.options.LogRequests
}

// LogResponses returns if the logging of responses is enabled.
func (l *Logger) LogResponses() bool {
	return l.options.LogResponses
}

// TimeFormat returns the configured time format.
func (l *Logger) TimeFormat() string {
	if l.options.TimeFormat == "" {
		return time.RFC3339
	}
	return l.options.TimeFormat
}

// Writer returns a io.Writer for the given level. It is generally combined with
// the Logger.StdLogger method.
func (l *Logger) Writer(level Level) io.Writer {
	return &writer{
		Logger: l.Logger,
		Name:   zap.String("name", l.name),
		Level:  level,
	}
}

// StdLogger returns a *log.Logger with the specified writer. The writer can be
// created with the Logger.Writer method.
//  srv := &http.Server{
//      Addr:     ":8080",
//      Handler:  httplog.Middleware(logger, http.DefaultServeMux),
//      ErrorLog: logger.StdLogger(logging.ErrorLevel),
//  }
func (l *Logger) StdLogger(level Level) *log.Logger {
	return log.New(l.Writer(level), "", 0)
}

// Debug logs a message at debug level.
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Info logs a message at info level.
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Warn logs a message at warn level.
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Error logs a message at error level.
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Fatal logs a message at fatal level and then calls to os.Exit(1).
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// Debugf formats and logs a message at debug level.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

// Infof formats and logs a message at info level.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

// Warnf formats and logs a message at warn level.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, args...))
}

// Errorf formats and logs a message at error level.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

// Fatalf formats and logs a message at fatal level and then calls to
// os.Exit(1).
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatal(fmt.Sprintf(format, args...))
}
