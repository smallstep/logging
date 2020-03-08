package tracing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

var randReader = rand.Reader

const (
	// TraceVersion defines the version used in the Traceparent header.
	TraceVersion = byte(1)

	// FlagSampled is the flag used to
	FlagSampled = byte(1)
)

// Traceparent defines the header used for distributed tracing. A Traceparent is
// based on the W3C trace-context specification available at
// https://w3c.github.io/trace-context/#version.
type Traceparent struct {
	version    byte
	traceID    [16]byte
	parentID   [8]byte
	traceFlags byte
}

// New generates a new Traceparent.
func New() (*Traceparent, error) {
	seed := make([]byte, 24)
	if _, err := io.ReadFull(randReader, seed); err != nil {
		return nil, err
	}
	t := &Traceparent{
		version:    TraceVersion,
		traceFlags: 0,
	}
	copy(t.traceID[:], seed[:16])
	copy(t.parentID[:], seed[16:])
	return t, nil
}

// Must generates a new Traceparent but panics if it fails to do it.
func Must() *Traceparent {
	t, err := New()
	if err != nil {
		panic(fmt.Errorf("tracing: cannot generate random number: %v", err))
	}
	return t
}

// Parse parses a Tranceparent string and returns a Tranceparent value.
func Parse(s string) (*Traceparent, error) {
	fail := func() (*Traceparent, error) {
		return nil, fmt.Errorf("tracing: tranceparent %s is not valid", s)
	}

	parts := strings.Split(s, "-")
	if len(parts) != 4 {
		return fail()
	}
	version, err := hex.DecodeString(parts[0])
	if err != nil || len(version) != 1 {
		return fail()
	}
	traceID, err := hex.DecodeString(parts[1])
	if err != nil || len(traceID) != 16 {
		return fail()
	}
	parentID, err := hex.DecodeString(parts[2])
	if err != nil || len(parentID) != 8 {
		return fail()
	}
	traceFlags, err := hex.DecodeString(parts[3])
	if err != nil || len(traceFlags) != 1 {
		return fail()
	}

	t := &Traceparent{
		version:    version[0],
		traceFlags: traceFlags[0],
	}
	copy(t.traceID[:], traceID)
	copy(t.parentID[:], parentID)
	return t, nil
}

// NewSpan returns a traceparent with a different parent-id.
func (t *Traceparent) NewSpan() (*Traceparent, error) {
	id := make([]byte, 8)
	if _, err := io.ReadFull(randReader, id); err != nil {
		return nil, err
	}
	tt := &Traceparent{
		version:    t.version,
		traceID:    t.traceID,
		traceFlags: t.traceFlags,
	}
	copy(tt.parentID[:], id[:])
	return tt, nil
}

// String returns the string representation of the traceparent.
func (t *Traceparent) String() string {
	// This is around 4x faster than fmt.
	return hex.EncodeToString([]byte{t.version}) + "-" +
		hex.EncodeToString(t.traceID[:]) + "-" +
		hex.EncodeToString(t.parentID[:]) + "-" +
		hex.EncodeToString([]byte{t.traceFlags})
}

// Version returns the version number.
func (t *Traceparent) Version() int {
	return int(t.version)
}

// TraceID returns the string representation of the trace-id.
func (t *Traceparent) TraceID() string {
	return hex.EncodeToString(t.traceID[:])
}

// ParentID returns the string representation of the parent-id.
func (t *Traceparent) ParentID() string {
	return hex.EncodeToString(t.parentID[:])
}

// SpanID returns the string representation of the parent-id, also called
// span-id.
func (t *Traceparent) SpanID() string {
	return t.ParentID()
}

// TraceFlags returns the trace-flags.
func (t *Traceparent) TraceFlags() byte {
	return t.traceFlags
}

// Sampled returns if the flag sampled is set.
func (t *Traceparent) Sampled() bool {
	return (t.traceFlags & FlagSampled) == FlagSampled
}

// Sample sets the sample flag.
func (t *Traceparent) Sample() {
	t.traceFlags |= FlagSampled
}

// SetTraceFlags set the given flags.
func (t *Traceparent) SetTraceFlags(flags byte) {
	t.traceFlags = flags
}
