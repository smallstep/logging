package encoder

import (
	"encoding/json"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var pool = buffer.NewPool()

// genericEncoder implements the zapcore.ArrayEncoder interface, and adds the
// given values to a buffer.
type genericEncoder struct {
	*zapcore.EncoderConfig
	buf        *buffer.Buffer
	formatTime string
}

func newGenericEncoder(config zapcore.EncoderConfig) *genericEncoder {
	return &genericEncoder{
		EncoderConfig: &config,
		buf:           pool.Get(),
		formatTime:    time.RFC3339,
	}
}

// CloneWithBuffer clones the generic encoder with the given buffer.
func (e *genericEncoder) CloneWithBuffer(buf *buffer.Buffer) *genericEncoder {
	return &genericEncoder{
		EncoderConfig: e.EncoderConfig,
		buf:           buf,
		formatTime:    e.formatTime,
	}
}

// GetObjectEncoder returns an type that encodes arrays and objects using json.
func (e *genericEncoder) GetObjectEncoder() *objectEncoder {
	return &objectEncoder{
		EncoderConfig: e.EncoderConfig,
		buf:           e.buf,
		formatTime:    e.formatTime,
	}
}

// Implement PrimitiveArrayEncoder interface.
func (e *genericEncoder) AppendBool(v bool)           { e.buf.AppendBool(v) }
func (e *genericEncoder) AppendByteString(v []byte)   { e.buf.AppendString(string(v)) }
func (e *genericEncoder) AppendFloat64(v float64)     { e.buf.AppendFloat(v, 64) }
func (e *genericEncoder) AppendFloat32(v float32)     { e.buf.AppendFloat(float64(v), 32) }
func (e *genericEncoder) AppendInt(v int)             { e.buf.AppendInt(int64(v)) }
func (e *genericEncoder) AppendInt64(v int64)         { e.buf.AppendInt(v) }
func (e *genericEncoder) AppendInt32(v int32)         { e.buf.AppendInt(int64(v)) }
func (e *genericEncoder) AppendInt16(v int16)         { e.buf.AppendInt(int64(v)) }
func (e *genericEncoder) AppendInt8(v int8)           { e.buf.AppendInt(int64(v)) }
func (e *genericEncoder) AppendString(v string)       { e.buf.AppendString(v) }
func (e *genericEncoder) AppendUint(v uint)           { e.buf.AppendUint(uint64(v)) }
func (e *genericEncoder) AppendUint64(v uint64)       { e.buf.AppendUint(v) }
func (e *genericEncoder) AppendUint32(v uint32)       { e.buf.AppendUint(uint64(v)) }
func (e *genericEncoder) AppendUint16(v uint16)       { e.buf.AppendUint(uint64(v)) }
func (e *genericEncoder) AppendUint8(v uint8)         { e.buf.AppendUint(uint64(v)) }
func (e *genericEncoder) AppendUintptr(v uintptr)     { e.buf.AppendUint(uint64(v)) }
func (e *genericEncoder) AppendComplex64(v complex64) { e.AppendComplex128(complex128(v)) }

func (e *genericEncoder) AppendComplex128(v complex128) {
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

// Implement ArrayEncoder interface
func (e *genericEncoder) AppendDuration(v time.Duration) {
	if encoder := e.EncodeDuration; encoder != nil {
		encoder(v, e)
	} else {
		e.buf.AppendString(v.String())
	}
}

func (e *genericEncoder) AppendTime(v time.Time) {
	if encoder := e.EncodeTime; encoder != nil {
		encoder(v, e)
	} else {
		e.buf.AppendTime(v, e.formatTime)
	}
}
func (e *genericEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	e.buf.AppendByte('[')
	v.MarshalLogArray(e.GetObjectEncoder())
	e.buf.AppendByte(']')
	return nil
}
func (e *genericEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	e.buf.AppendByte('{')
	v.MarshalLogObject(e.GetObjectEncoder())
	e.buf.AppendByte('}')
	return nil
}

func (e *genericEncoder) AppendReflected(value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	e.buf.AppendString(string(b))
	return nil
}
