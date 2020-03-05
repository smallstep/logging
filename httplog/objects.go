package httplog

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"go.uber.org/zap/zapcore"
)

// Request represents an HTTP request, it implements a zapcore.ObjectMarshaller
// on it.
type Request struct {
	Headers Headers
	Body    []byte
}

// NewRequest creates a Request object from an http.Request.
func NewRequest(r *http.Request) (*Request, error) {
	b, err := dumpRequestBody(r)
	if err != nil {
		return nil, err
	}
	return &Request{
		Headers: Headers(r.Header),
		Body:    b,
	}, nil
}

// MarshalLogObject adds the properties of the request in the log.
func (r *Request) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	_ = enc.AddObject("headers", r.Headers)
	enc.AddBinary("body", r.Body)
	if ct := r.Headers.Get("Content-Type"); ct != "" {
		if s, ok := stringBody(ct, r.Body); ok {
			enc.AddString("text", s)
		}
	}
	return nil
}

// Response represents an HTTP response, it implements a
// zapcore.ObjectMarshaller on it.
type Response struct {
	Headers Headers
	Body    []byte
}

// MarshalLogObject adds the properties of the response in the log.
func (r *Response) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	_ = enc.AddObject("headers", r.Headers)
	enc.AddBinary("body", r.Body)
	if ct := r.Headers.Get("Content-Type"); ct != "" {
		if s, ok := stringBody(ct, r.Body); ok {
			enc.AddString("text", s)
		}
	}
	return nil
}

// Headers represents key-value pairs in the HTTP headers.
type Headers http.Header

// Get gets the first value associated with the given key. If there are no
// values associated with the key, Get returns "".
func (h Headers) Get(key string) string {
	return http.Header(h).Get(key)
}

// Values returns all values associated with the given key.
func (h Headers) Values(key string) []string {
	return http.Header(h).Values(key)
}

// MarshalLogObject adds the headers in the log.
func (h Headers) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for k, v := range h {
		_ = enc.AddArray(k, zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
			for i := range v {
				enc.AppendString(v[i])
			}
			return nil
		}))
	}
	return nil
}

func dumpRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return nil, nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		return nil, err
	}
	if err := r.Body.Close(); err != nil {
		return nil, err
	}
	r.Body = ioutil.NopCloser(&buf)
	return buf.Bytes(), nil
}

func stringBody(contentType string, body []byte) (string, bool) {
	if strings.HasPrefix(contentType, "text") ||
		strings.HasSuffix(contentType, "json") ||
		strings.HasSuffix(contentType, "x-www-form-urlencoded") ||
		strings.HasSuffix(contentType, "xml") {
		return string(body), true
	}
	return "", false
}
