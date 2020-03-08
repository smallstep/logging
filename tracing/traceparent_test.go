package tracing

import (
	"bytes"
	"reflect"
	"testing"
)

func fakeRandReader(b []byte) func() {
	tmp := randReader
	randReader = bytes.NewReader(b)
	return func() {
		randReader = tmp
	}
}

func fakeBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 10)
	}
	return b
}

func TestNew(t *testing.T) {
	t.Cleanup(fakeRandReader(fakeBytes(24)))

	tests := []struct {
		name    string
		want    *Traceparent
		wantErr bool
	}{
		{"ok", &Traceparent{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{6, 7, 8, 9, 0, 1, 2, 3},
			traceFlags: 0,
		}, false},
		{"fail", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMust(t *testing.T) {
	t.Cleanup(fakeRandReader(fakeBytes(24)))

	tests := []struct {
		name    string
		want    *Traceparent
		wantErr bool
	}{
		{"ok", &Traceparent{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{6, 7, 8, 9, 0, 1, 2, 3},
			traceFlags: 0,
		}, false},
		{"fail", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if err := recover(); (err != nil) != tt.wantErr {
					t.Errorf("Must() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()
			if got := Must(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Must() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *Traceparent
		wantErr bool
	}{
		{"ok", args{"01-00010203040506070809000102030405-0607080900010203-00"}, &Traceparent{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{6, 7, 8, 9, 0, 1, 2, 3},
			traceFlags: 0,
		}, false},
		{"ok", args{"01-00010203040506070809000102030405-0607080900010203-01"}, &Traceparent{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{6, 7, 8, 9, 0, 1, 2, 3},
			traceFlags: FlagSampled,
		}, false},
		{"fail", args{""}, nil, true},
		{"fail", args{"1-00010203040506070809000102030405-0607080900010203-01"}, nil, true},
		{"fail", args{"01-0010203040506070809000102030405-0607080900010203-01"}, nil, true},
		{"fail", args{"01-00010203040506070809000102030405-607080900010203-01"}, nil, true},
		{"fail", args{"01-00010203040506070809000102030405-0607080900010203-1"}, nil, true},
		{"fail", args{"0x-00010203040506070809000102030405-0607080900010203-01"}, nil, true},
		{"fail", args{"01-0001020304050607080900010203040x-0607080900010203-01"}, nil, true},
		{"fail", args{"01-00010203040506070809000102030405-060708090001020x-01"}, nil, true},
		{"fail", args{"01-00010203040506070809000102030405-0607080900010203-0x"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTraceparent_NewSpan(t *testing.T) {
	t.Cleanup(fakeRandReader(fakeBytes(8)))

	type fields struct {
		version    byte
		traceID    [16]byte
		parentID   [8]byte
		traceFlags byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    *Traceparent
		wantErr bool
	}{
		{"ok", fields{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{6, 7, 8, 9, 0, 1, 2, 3},
			traceFlags: 0,
		}, &Traceparent{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{0, 1, 2, 3, 4, 5, 6, 7},
			traceFlags: 0,
		}, false},
		{"fail", fields{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{6, 7, 8, 9, 0, 1, 2, 3},
			traceFlags: 0,
		}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := &Traceparent{
				version:    tt.fields.version,
				traceID:    tt.fields.traceID,
				parentID:   tt.fields.parentID,
				traceFlags: tt.fields.traceFlags,
			}
			got, err := tp.NewSpan()
			if (err != nil) != tt.wantErr {
				t.Errorf("Traceparent.NewSpan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Traceparent.NewSpan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTraceparent_String(t *testing.T) {
	type fields struct {
		version    byte
		traceID    [16]byte
		parentID   [8]byte
		traceFlags byte
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"ok", fields{
			version:    TraceVersion,
			traceID:    [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5},
			parentID:   [8]byte{6, 7, 8, 9, 0, 1, 2, 3},
			traceFlags: 0,
		}, "01-00010203040506070809000102030405-0607080900010203-00"},
		{"ok", fields{
			version:    0x2,
			traceID:    [16]byte{0xd7, 0x40, 0x67, 0xf8, 0x3f, 0xa7, 0x46, 0x15, 0x35, 0x57, 0xc6, 0x2d, 0x2e, 0x26, 0x6, 0x1b},
			parentID:   [8]byte{0x55, 0x9c, 0x1a, 0x5f, 0x60, 0xe6, 0x7a, 0x81},
			traceFlags: 0x70,
		}, "02-d74067f83fa746153557c62d2e26061b-559c1a5f60e67a81-70"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := &Traceparent{
				version:    tt.fields.version,
				traceID:    tt.fields.traceID,
				parentID:   tt.fields.parentID,
				traceFlags: tt.fields.traceFlags,
			}
			if got := tp.String(); got != tt.want {
				t.Errorf("Traceparent.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTraceparent(t *testing.T) {
	t.Cleanup(fakeRandReader(fakeBytes(24)))

	tp, err := New()
	if err != nil {
		t.Errorf("New() error = %v", err)
	}
	if v := tp.Version(); v != 1 {
		t.Errorf("Traceparent.Version() = %v, want %v", v, 1)
	}
	if v := tp.TraceID(); v != "00010203040506070809000102030405" {
		t.Errorf("Traceparent.TraceID() = %v, want 00010203040506070809000102030405", v)
	}
	if v := tp.ParentID(); v != "0607080900010203" {
		t.Errorf("Traceparent.ParentID() = %v, want 0607080900010203", v)
	}
	if v := tp.SpanID(); v != "0607080900010203" {
		t.Errorf("Traceparent.SpanID() = %v, want 0607080900010203", v)
	}
	if v := tp.TraceFlags(); v != 0 {
		t.Errorf("Traceparent.TraceFlags() = %v, want 0x00", v)
	}
	if v := tp.Sampled(); v != false {
		t.Errorf("Traceparent.Sampled() = %v, want false", v)
	}
	tp.Sample()
	if v := tp.Sampled(); v != true {
		t.Errorf("Traceparent.Sampled() = %v, want true", v)
	}
}

func TestTraceparent_Sample(t *testing.T) {
	type fields struct {
		traceFlags byte
	}
	tests := []struct {
		name   string
		fields fields
		want   byte
	}{
		{"ok 0x00", fields{0x00}, 0x01},
		{"ok 0x01", fields{0x01}, 0x01},
		{"ok 0xf0", fields{0xf0}, 0xf1},
		{"ok 0xff", fields{0xff}, 0xff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := &Traceparent{
				traceFlags: tt.fields.traceFlags,
			}
			tp.Sample()
			if tp.traceFlags != tt.want {
				t.Errorf("Traceparent.Sample() = %v, want %v", tp.traceFlags, tt.want)
			}
		})
	}
}

func TestTraceparent_SetTraceFlags(t *testing.T) {
	type fields struct {
		traceFlags byte
	}
	type args struct {
		flags byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   byte
	}{
		{"ok", fields{0x00}, args{0x01}, 0x01},
		{"ok", fields{0xff}, args{0x00}, 0x00},
		{"ok", fields{0x00}, args{0xfa}, 0xfa},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := &Traceparent{
				traceFlags: tt.fields.traceFlags,
			}
			tp.SetTraceFlags(tt.args.flags)
			if tp.traceFlags != tt.want {
				t.Errorf("Traceparent.SetTraceFlags() = %v, want %v", tp.traceFlags, tt.want)
			}
		})
	}
}
