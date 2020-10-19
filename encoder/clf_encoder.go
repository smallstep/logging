package encoder

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var clfFields = [...]string{
	"request-id", "remote-address", "name", "user-id", "time", "duration", "method", "path", "protocol", "status", "size",
}

var clfFieldsEmpty []string
var clfFieldsMap map[string]int

func init() {
	clfFieldsEmpty = make([]string, len(clfFields))
	clfFieldsMap = make(map[string]int, len(clfFields))
	for i, s := range clfFields {
		clfFieldsEmpty[i] = "-"
		clfFieldsMap[s] = i
	}
}

// NewCLFEncoder returns a new encoder that logs messages with the Common Log
// Format. Each logged line will follow the format:
// 	<request-id> <remote-address> <name> <user-id> <time> <duration> "<method> <path> <protocol>" <status> <size>
func NewCLFEncoder(config zapcore.EncoderConfig) zapcore.Encoder {
	return &clfEncoder{
		EncoderConfig: &config,
		data:          make([]string, len(clfFieldsMap)),
	}
}

type clfEncoder struct {
	*zapcore.EncoderConfig
	data []string
}

// Clone copies the encoder, ensuring that adding fields to the copy doesn't
// affect the original.
func (e *clfEncoder) Clone() zapcore.Encoder {
	data := make([]string, len(clfFieldsMap))
	copy(data, e.data)
	return &clfEncoder{
		EncoderConfig: e.EncoderConfig,
		data:          data,
	}
}

func (e *clfEncoder) clone() *clfEncoder {
	data := make([]string, len(clfFieldsMap))
	copy(data, clfFieldsEmpty)
	return &clfEncoder{
		EncoderConfig: e.EncoderConfig,
		data:          data,
	}
}

// EncodeEntry encodes an entry and fields, along with any accumulated context,
// into a byte buffer and returns it. Any fields that are empty, including
// fields on the `Entry` type, should be omitted.
func (e *clfEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := e.clone()

	buf := pool.Get()
	if entry.Message != "" {
		buf.AppendString(entry.Message + "\n")
	}
	if len(fields) == 0 {
		return buf, nil
	}

	for i := range fields {
		fields[i].AddTo(final)
	}

	buf.AppendString(final.data[0])
	buf.AppendByte(' ')
	buf.AppendString(final.data[1])
	buf.AppendByte(' ')
	buf.AppendString(final.data[2])
	buf.AppendByte(' ')
	buf.AppendString(final.data[3])
	buf.AppendByte(' ')
	buf.AppendString(final.data[4])
	buf.AppendByte(' ')
	buf.AppendString(final.data[5])
	buf.AppendString(" \"")
	buf.AppendString(final.data[6])
	buf.AppendByte(' ')
	buf.AppendString(final.data[7])
	buf.AppendByte(' ')
	buf.AppendString(final.data[8])
	buf.AppendString("\" ")
	buf.AppendString(final.data[9])
	buf.AppendByte(' ')
	buf.AppendString(final.data[10])
	buf.AppendByte('\n')
	return buf, nil
}

// Implementation of the zapcore.ObjectEncoder interface.
func (e *clfEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	return nil
}
func (e *clfEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	return nil
}

func (e *clfEncoder) AddBinary(key string, value []byte) { // for arbitrary bytes
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = base64.StdEncoding.EncodeToString(value)
	}
}

func (e *clfEncoder) AddByteString(key string, value []byte) { // for UTF-8 encoded bytes
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = string(value)
	}
}

func (e *clfEncoder) AddString(key, value string) {
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = value
	}
}

func (e *clfEncoder) AddBool(key string, value bool) {
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = strconv.FormatBool(value)
	}
}

func (e *clfEncoder) AddComplex128(key string, value complex128) {
	if i, ok := clfFieldsMap[key]; ok {
		r, img := float64(real(value)), float64(imag(value))
		e.data[i] = fmt.Sprintf(`"%s+%si"`, strconv.FormatFloat(r, 'f', -1, 64), strconv.FormatFloat(img, 'f', -1, 64))
	}
}

func (e *clfEncoder) AddComplex64(key string, value complex64) {
	e.AddComplex128(key, complex128(value))
}

func (e *clfEncoder) AddDuration(key string, value time.Duration) {
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = strconv.FormatInt(value.Milliseconds(), 10)
	}
}

func (e *clfEncoder) AddFloat64(key string, value float64) {
	if i, ok := clfFieldsMap[key]; ok {
		switch {
		case math.IsNaN(value):
			e.data[i] = "NaN"
		case math.IsInf(value, 1):
			e.data[i] = "+Inf"
		case math.IsInf(value, -1):
			e.data[i] = "-Inf"
		default:
			e.data[i] = strconv.FormatFloat(value, 'f', -1, 64)
		}
	}
}

func (e *clfEncoder) AddInt64(key string, value int64) {
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = strconv.FormatInt(value, 10)
	}
}

func (e *clfEncoder) AddUint64(key string, value uint64) {
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = strconv.FormatUint(value, 10)
	}
}

func (e *clfEncoder) AddTime(key string, value time.Time) {
	if i, ok := clfFieldsMap[key]; ok {
		e.data[i] = value.Format(time.RFC3339)
	}
}

func (e *clfEncoder) AddFloat32(key string, value float32) { e.AddFloat64(key, float64(value)) }
func (e *clfEncoder) AddInt(key string, value int)         { e.AddInt64(key, int64(value)) }
func (e *clfEncoder) AddInt32(key string, value int32)     { e.AddInt64(key, int64(value)) }
func (e *clfEncoder) AddInt16(key string, value int16)     { e.AddInt64(key, int64(value)) }
func (e *clfEncoder) AddInt8(key string, value int8)       { e.AddInt64(key, int64(value)) }
func (e *clfEncoder) AddUint(key string, value uint)       { e.AddUint64(key, uint64(value)) }
func (e *clfEncoder) AddUint32(key string, value uint32)   { e.AddUint64(key, uint64(value)) }
func (e *clfEncoder) AddUint16(key string, value uint16)   { e.AddUint64(key, uint64(value)) }
func (e *clfEncoder) AddUint8(key string, value uint8)     { e.AddUint64(key, uint64(value)) }
func (e *clfEncoder) AddUintptr(key string, value uintptr) { e.AddUint64(key, uint64(value)) }

// AddReflected uses reflection to serialize arbitrary objects, so it can be
// slow and allocation-heavy.
func (e *clfEncoder) AddReflected(key string, value interface{}) error {
	return nil
}

// OpenNamespace opens an isolated namespace where all subsequent fields will be
// added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (e *clfEncoder) OpenNamespace(key string) {}
