package httplog

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
)

const (
	// MessageKey is the key used to store the log message.
	MessageKey = "message"
	// RequestKey is the key used to keep the request if enabled.
	RequestKey = "request"
	// ResponseKey is the key used to keep the response if enabled.
	ResponseKey = "response"
	// ErrorKey is the key used to store an error.
	ErrorKey = "error"
)

// ResponseLogger defines an interface that a responseWrite can implement to
// support the capture of the status code, the number of bytes written and
// extra log entry fields.
type ResponseLogger interface {
	http.ResponseWriter
	Size() int
	StatusCode() int
	Fields() map[string]interface{}
	WithField(key string, value interface{})
	WithFields(map[string]interface{})
	Field(key string) (interface{}, bool)
	Request() (*Request, bool)
}

// RawResponseLogger defines an interface that a responseWrite can implement to
// support the capture of the status code, the number of bytes written and extra
// log entry fields.
type RawResponseLogger interface {
	ResponseLogger
	Buffer() *bytes.Buffer
	Response() []byte
}

// NewResponseLogger wraps the given response writer with methods to capture the
// status code, the number of bytes written, and methods to add new log entries.
// It won't wrap the response writer if it's already a ResponseLogger.
func NewResponseLogger(w http.ResponseWriter) ResponseLogger {
	if rw, ok := w.(ResponseLogger); ok {
		return rw
	}
	return wrapLogger(w)
}

// NewRawResponseLogger wraps the given response writer with the methods to
// capture the status code, the bytes written, and methods to add new log
// entries. It won't wrap the response writer if it's already a
// RawResponseLogger.
func NewRawResponseLogger(w http.ResponseWriter) RawResponseLogger {
	if rw, ok := w.(RawResponseLogger); ok {
		return rw
	}
	return &rwDefaultRaw{
		ResponseLogger: wrapLogger(w),
		response:       new(bytes.Buffer),
	}
}

func wrapLogger(w http.ResponseWriter) (rw ResponseLogger) {
	rw = &rwDefault{w, 200, 0, nil}
	if f, ok := w.(http.Flusher); ok {
		rw = &rwFlusher{rw, f}
	}
	if h, ok := w.(http.Hijacker); ok {
		rw = &rwHijacker{rw, h}
	}
	if p, ok := w.(http.Pusher); ok {
		rw = &rwPusher{rw, p}
	}
	return
}

type rwDefault struct {
	http.ResponseWriter
	code   int
	size   int
	fields map[string]interface{}
}

type rwDefaultRaw struct {
	ResponseLogger
	response *bytes.Buffer
}

func (r *rwDefaultRaw) Buffer() *bytes.Buffer {
	return r.response
}

func (r *rwDefaultRaw) Response() []byte {
	return r.response.Bytes()
}

func (r *rwDefaultRaw) Write(p []byte) (n int, err error) {
	r.response.Write(p)
	_, err = r.ResponseLogger.Write(p)
	return
}

func (r *rwDefault) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *rwDefault) Write(p []byte) (n int, err error) {
	n, err = r.ResponseWriter.Write(p)
	r.size += n
	return
}

func (r *rwDefault) WriteHeader(code int) {
	r.ResponseWriter.WriteHeader(code)
	r.code = code
}

func (r *rwDefault) Size() int {
	return r.size
}

func (r *rwDefault) StatusCode() int {
	return r.code
}

func (r *rwDefault) Request() (*Request, bool) {
	req, ok := r.fields["request"].(*Request)
	return req, ok
}

func (r *rwDefault) Field(key string) (v interface{}, ok bool) {
	v, ok = r.fields[key]
	return
}

func (r *rwDefault) Fields() map[string]interface{} {
	return r.fields
}

func (r *rwDefault) WithField(key string, value interface{}) {
	if r.fields == nil {
		r.fields = make(map[string]interface{})
	}
	r.fields[key] = value
}

func (r *rwDefault) WithFields(fields map[string]interface{}) {
	if r.fields == nil {
		r.fields = make(map[string]interface{}, len(fields))
	}
	for k, v := range fields {
		r.fields[k] = v
	}
}

type rwFlusher struct {
	ResponseLogger
	f http.Flusher
}

func (r *rwFlusher) Flush() {
	r.f.Flush()
}

type rwHijacker struct {
	ResponseLogger
	h http.Hijacker
}

func (r *rwHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.h.Hijack()
}

type rwPusher struct {
	ResponseLogger
	p http.Pusher
}

func (rw *rwPusher) Push(target string, opts *http.PushOptions) error {
	return rw.p.Push(target, opts)
}
