package encoder

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// For JSON-escaping; see jsonEncoder.safeAddString below.
const _hex = "0123456789abcdef"

type objectEncoder struct {
	*zapcore.EncoderConfig
	buf        *buffer.Buffer
	formatTime string
}

// Implementation of zapcore.ObjectEncoder
func (e *objectEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	e.addKey(key)
	return e.AppendArray(marshaler)
}
func (e *objectEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	e.addKey(key)
	return e.AppendObject(marshaler)
}

// Built-in types.
func (e *objectEncoder) AddBinary(key string, value []byte) { // for arbitrary bytes
	e.AddString(key, base64.StdEncoding.EncodeToString(value))
}

func (e *objectEncoder) AddByteString(key string, value []byte) { // for UTF-8 encoded bytes
	e.AddString(key, string(value))
}

func (e *objectEncoder) AddString(key, value string) {
	e.addKey(key)
	e.AppendString(value)
}

func (e *objectEncoder) AddBool(key string, value bool) {
	e.addKey(key)
	e.AppendBool(value)
}

func (e *objectEncoder) AddComplex128(key string, value complex128) {
	e.addKey(key)
	e.AppendComplex128(value)
}

func (e *objectEncoder) AddComplex64(key string, value complex64) {
	e.addKey(key)
	e.AppendComplex64(value)
}

func (e *objectEncoder) AddDuration(key string, value time.Duration) {
	e.addKey(key)
	if encoder := e.EncodeDuration; encoder != nil {
		encoder(value, e)
	} else {
		e.AppendString(value.String())
	}
}

func (e *objectEncoder) AddFloat64(key string, value float64) {
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

func (e *objectEncoder) AddInt64(key string, value int64) {
	e.addKey(key)
	e.AppendInt64(value)
}

func (e *objectEncoder) AddUint64(key string, value uint64) {
	e.addKey(key)
	e.AppendUint64(value)
}

func (e *objectEncoder) AddTime(key string, value time.Time) {
	e.addKey(key)
	if encoder := e.EncodeTime; encoder != nil {
		encoder(value, e)
	} else {
		e.AppendTime(value)
	}
}

// AddReflected uses reflection to serialize arbitrary objects, so it can be
// slow and allocation-heavy.
func (e *objectEncoder) AddReflected(key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	e.addKey(key)
	e.buf.AppendString(string(b))
	return nil
}

// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (e *objectEncoder) OpenNamespace(key string) {}

func (e *objectEncoder) AddFloat32(key string, value float32) { e.AddFloat64(key, float64(value)) }
func (e *objectEncoder) AddInt(key string, value int)         { e.AddInt64(key, int64(value)) }
func (e *objectEncoder) AddInt32(key string, value int32)     { e.AddInt64(key, int64(value)) }
func (e *objectEncoder) AddInt16(key string, value int16)     { e.AddInt64(key, int64(value)) }
func (e *objectEncoder) AddInt8(key string, value int8)       { e.AddInt64(key, int64(value)) }
func (e *objectEncoder) AddUint(key string, value uint)       { e.AddUint64(key, uint64(value)) }
func (e *objectEncoder) AddUint32(key string, value uint32)   { e.AddUint64(key, uint64(value)) }
func (e *objectEncoder) AddUint16(key string, value uint16)   { e.AddUint64(key, uint64(value)) }
func (e *objectEncoder) AddUint8(key string, value uint8)     { e.AddUint64(key, uint64(value)) }
func (e *objectEncoder) AddUintptr(key string, value uintptr) { e.AddUint64(key, uint64(value)) }

// Implement PrimiviteArrayEncoder interface.
func (e *objectEncoder) AppendString(v string) {
	e.buf.AppendByte('"')
	e.safeAppendString(v)
	e.buf.AppendByte('"')
}

func (e *objectEncoder) AppendByteString(v []byte) {
	e.addElementSeparator()
	e.buf.AppendByte('"')
	e.safeAddByteString(v)
	e.buf.AppendByte('"')
}

func (e *objectEncoder) AppendComplex128(v complex128) {
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(v)), float64(imag(v))
	e.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	e.buf.AppendFloat(r, 64)
	e.buf.AppendByte('+')
	e.buf.AppendFloat(i, 64)
	e.buf.AppendByte('i')
	e.buf.AppendByte('"')
}

func (e *objectEncoder) AppendBool(v bool)           { e.buf.AppendBool(v) }
func (e *objectEncoder) AppendFloat64(v float64)     { e.buf.AppendFloat(v, 64) }
func (e *objectEncoder) AppendFloat32(v float32)     { e.buf.AppendFloat(float64(v), 32) }
func (e *objectEncoder) AppendInt(v int)             { e.buf.AppendInt(int64(v)) }
func (e *objectEncoder) AppendInt64(v int64)         { e.buf.AppendInt(v) }
func (e *objectEncoder) AppendInt32(v int32)         { e.buf.AppendInt(int64(v)) }
func (e *objectEncoder) AppendInt16(v int16)         { e.buf.AppendInt(int64(v)) }
func (e *objectEncoder) AppendInt8(v int8)           { e.buf.AppendInt(int64(v)) }
func (e *objectEncoder) AppendUint(v uint)           { e.buf.AppendUint(uint64(v)) }
func (e *objectEncoder) AppendUint64(v uint64)       { e.buf.AppendUint(v) }
func (e *objectEncoder) AppendUint32(v uint32)       { e.buf.AppendUint(uint64(v)) }
func (e *objectEncoder) AppendUint16(v uint16)       { e.buf.AppendUint(uint64(v)) }
func (e *objectEncoder) AppendUint8(v uint8)         { e.buf.AppendUint(uint64(v)) }
func (e *objectEncoder) AppendUintptr(v uintptr)     { e.buf.AppendUint(uint64(v)) }
func (e *objectEncoder) AppendComplex64(v complex64) { e.AppendComplex128(complex128(v)) }

// Implement ArrayEncoder interface
func (e *objectEncoder) AppendDuration(v time.Duration) {
	if encoder := e.EncodeDuration; encoder != nil {
		encoder(v, e)
	} else {
		e.buf.AppendString(v.String())
	}
}

func (e *objectEncoder) AppendTime(v time.Time) {
	if encoder := e.EncodeTime; encoder != nil {
		encoder(v.UTC(), e)
	} else {
		e.buf.AppendTime(v.UTC(), e.formatTime)
	}
}
func (e *objectEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	e.buf.AppendByte('[')
	v.MarshalLogArray(e)
	e.buf.AppendByte(']')
	return nil
}
func (e *objectEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	e.buf.AppendByte('{')
	v.MarshalLogObject(e)
	e.buf.AppendByte('}')
	return nil
}

func (e *objectEncoder) AppendReflected(value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	e.buf.AppendString(string(b))
	return nil
}

func (e *objectEncoder) addKey(key string) {
	e.addElementSeparator()
	e.AppendString(key)
	e.buf.AppendByte(':')
}

func (e *objectEncoder) addElementSeparator() {
	last := e.buf.Len() - 1
	if last < 0 {
		return
	}
	switch e.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		e.buf.AppendByte(',')
	}
}

// safeAppendString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the user
// from browser vulnerabilities or JSONP-related problems.
func (e *objectEncoder) safeAppendString(s string) {
	for i := 0; i < len(s); {
		if e.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if e.tryAddRuneError(r, size) {
			i++
			continue
		}
		e.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (e *objectEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if e.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if e.tryAddRuneError(r, size) {
			i++
			continue
		}
		e.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (e *objectEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		e.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		e.buf.AppendByte('\\')
		e.buf.AppendByte(b)
	case '\n':
		e.buf.AppendByte('\\')
		e.buf.AppendByte('n')
	case '\r':
		e.buf.AppendByte('\\')
		e.buf.AppendByte('r')
	case '\t':
		e.buf.AppendByte('\\')
		e.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		e.buf.AppendString(`\u00`)
		e.buf.AppendByte(_hex[b>>4])
		e.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (e *objectEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		e.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}
