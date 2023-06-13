package logging

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type options struct {
	Format       string        `json:"format"`
	Level        zapcore.Level `json:"level"`
	TraceHeader  string        `json:"traceHeader"`
	LogRequests  bool          `json:"logRequests"`
	LogResponses bool          `json:"logResponses"`
	TimeFormat   string        `json:"timeFormat"`
	CallerSkip   int           `json:"callerSkip"`
}

func defaultOptions() *options {
	return &options{
		Format:      formatFromEnv(),
		Level:       levelFromEnv(),
		TraceHeader: DefaultTraceHeader,
		CallerSkip:  1,
	}
}

func formatFromEnv() (format string) {
	if format = os.Getenv("LOG_FORMAT"); format == "" {
		format = "json"
	}
	return
}

func levelFromEnv() (level zapcore.Level) {
	v := os.Getenv("LOG_LEVEL")
	if err := level.UnmarshalText([]byte(v)); err != nil {
		level = zapcore.InfoLevel
	}

	return
}

func (o *options) apply(opts []Option) (err error) {
	for _, fn := range opts {
		if err = fn(o); err != nil {
			return
		}
	}
	return
}

// Option is the type used to modify logger options.
type Option func(o *options) error

// WithConfig uses a JSON to configure the logger.
func WithConfig(raw json.RawMessage) Option {
	return func(o *options) error {
		if err := json.Unmarshal(raw, o); err != nil {
			return errors.Wrap(err, "error unmarshalling logging attributes")
		}
		return nil
	}
}

// WithFormatText configures the format of the logs as text. Defaults to text.
func WithFormatText() Option {
	return func(o *options) error {
		o.Format = "text"
		return nil
	}
}

// WithFormatJSON configures the format of the logs as JSON. Defaults to text.
func WithFormatJSON() Option {
	return func(o *options) error {
		o.Format = "json"
		return nil
	}
}

// WithTimeFormat sets a specific format for the time fields.
func WithTimeFormat(format string) Option {
	return func(o *options) error {
		o.TimeFormat = format
		return nil
	}
}

// WithTraceHeader defines the name of the header used for tracing. Defaults to
// 'Traceparent'.
func WithTraceHeader(name string) Option {
	return func(o *options) error {
		o.TraceHeader = name
		return nil
	}
}

// WithLogRequests enables the log of the requests.
func WithLogRequests() Option {
	return func(o *options) error {
		o.LogRequests = true
		return nil
	}
}

// WithLogResponses enables the log of responses
func WithLogResponses() Option {
	return func(o *options) error {
		o.LogResponses = true
		return nil
	}
}

// WithCallerSkip increases the number of callers skipped by caller annotation.
func WithCallerSkip(skip int) Option {
	return func(o *options) error {
		o.CallerSkip = skip
		return nil
	}
}

// WithLogLevel sets the verbosity of the logger.
func WithLogLevel(level zapcore.Level) Option {
	return func(o *options) error {
		o.Level = level
		return nil
	}
}
