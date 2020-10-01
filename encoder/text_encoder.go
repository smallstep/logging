package encoder

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	red      = "\x1b[31m"
	yellow   = "\x1b[33m"
	blue     = "\x1b[36m"
	gray     = "\x1b[37m"
	colorEnd = "\x1b[0m"
)

// NewTextEncoder returns a new text encoder that logs messages similar to
// logrus text encoder.
func NewTextEncoder(config zapcore.EncoderConfig) zapcore.Encoder {
	return &textEncoder{
		genericEncoder: newGenericEncoder(config),
	}
}

type textEncoder struct {
	*genericEncoder
	colorStart string
}

// Clone copies the encoder, ensuring that adding fields to the copy doesn't
// affect the original.
func (e *textEncoder) Clone() zapcore.Encoder {
	enc := e.clone()
	enc.buf.Write(e.buf.Bytes())
	return enc
}

func (e *textEncoder) clone() *textEncoder {
	return &textEncoder{
		genericEncoder: e.CloneWithBuffer(pool.Get()),
		colorStart:     e.colorStart,
	}
}

func (e *textEncoder) addKey(key string) {
	if e.colorStart != "" {
		e.AppendString(e.colorStart + key + colorEnd)
	} else {
		e.AppendString(key)
	}
	e.AppendString("=")
}

func (e *textEncoder) addMessage(level, message string) {
	if e.colorStart != "" {
		e.AppendString(e.colorStart + level + colorEnd)
	} else {
		e.AppendString(level)
	}
	e.AppendString(fmt.Sprintf(" %-44s ", message))
}

// EncodeEntry encodes an entry and fields, along with any accumulated context,
// into a byte buffer and returns it. Any fields that are empty, including
// fields on the `Entry` type, should be omitted.
func (e *textEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := e.clone()

	switch entry.Level {
	case zapcore.WarnLevel:
		final.colorStart = yellow
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		final.colorStart = red
	default:
		final.colorStart = blue
	}

	final.addMessage(strings.ToUpper(entry.Level.String()), entry.Message)

	for i := range fields {
		fields[i].AddTo(final)
		final.buf.AppendString(" ")
	}
	final.buf.AppendString("\n")
	return final.buf, nil
}

// Implementation of the zapcore.ObjectEncoder interface.
func (e *textEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	e.addKey(key)
	return e.AppendArray(marshaler)
}
func (e *textEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	e.addKey(key)
	return e.AppendObject(marshaler)
}

func (e *textEncoder) AddBinary(key string, value []byte) { // for arbitrary bytes
	e.AddString(key, `"`+base64.StdEncoding.EncodeToString(value)+`"`)
}

func (e *textEncoder) AddByteString(key string, value []byte) { // for UTF-8 encoded bytes
	e.AddString(key, string(value))
}

func (e *textEncoder) AddString(key, value string) {
	e.addKey(key)
	e.AppendString(value)
}

func (e *textEncoder) AddBool(key string, value bool) {
	e.addKey(key)
	e.AppendBool(value)
}

func (e *textEncoder) AddComplex128(key string, value complex128) {
	e.addKey(key)
	e.AppendComplex128(value)
}

func (e *textEncoder) AddComplex64(key string, value complex64) {
	e.addKey(key)
	e.AppendComplex64(value)
}

func (e *textEncoder) AddDuration(key string, value time.Duration) {
	e.addKey(key)
	e.AppendDuration(value)
}

func (e *textEncoder) AddFloat64(key string, value float64) {
	e.addKey(key)
	switch {
	case math.IsNaN(value):
		e.AppendString("NaN")
	case math.IsInf(value, 1):
		e.AppendString("+Inf")
	case math.IsInf(value, -1):
		e.AppendString("-Inf")
	default:
		e.AppendFloat64(value)
	}
}

func (e *textEncoder) AddInt64(key string, value int64) {
	e.addKey(key)
	e.AppendInt64(value)
}

func (e *textEncoder) AddUint64(key string, value uint64) {
	e.addKey(key)
	e.AppendUint64(value)
}

func (e *textEncoder) AddTime(key string, value time.Time) {
	e.addKey(key)
	e.AppendTime(value)
}

func (e *textEncoder) AddFloat32(key string, value float32) { e.AddFloat64(key, float64(value)) }
func (e *textEncoder) AddInt(key string, value int)         { e.AddInt64(key, int64(value)) }
func (e *textEncoder) AddInt32(key string, value int32)     { e.AddInt64(key, int64(value)) }
func (e *textEncoder) AddInt16(key string, value int16)     { e.AddInt64(key, int64(value)) }
func (e *textEncoder) AddInt8(key string, value int8)       { e.AddInt64(key, int64(value)) }
func (e *textEncoder) AddUint(key string, value uint)       { e.AddUint64(key, uint64(value)) }
func (e *textEncoder) AddUint32(key string, value uint32)   { e.AddUint64(key, uint64(value)) }
func (e *textEncoder) AddUint16(key string, value uint16)   { e.AddUint64(key, uint64(value)) }
func (e *textEncoder) AddUint8(key string, value uint8)     { e.AddUint64(key, uint64(value)) }
func (e *textEncoder) AddUintptr(key string, value uintptr) { e.AddUint64(key, uint64(value)) }

// AddReflected uses reflection to serialize arbitrary objects, so it can be
// slow and allocation-heavy.
func (e *textEncoder) AddReflected(key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	e.AppendByteString(b)
	return nil
}

// OpenNamespace opens an isolated namespace where all subsequent fields will be
// added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (e *textEncoder) OpenNamespace(key string) {}
