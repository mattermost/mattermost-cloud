package api

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// ResponseWriterWrapper is a wrapper for writing http responses with custom
// status code logic.
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode        int
	statusCodeWritten bool
	hijacker          http.Hijacker
	flusher           http.Flusher
}

// NewWrappedWriter returns a new ResponseWriterWrapper.
func NewWrappedWriter(original http.ResponseWriter) *ResponseWriterWrapper {
	hijacker, _ := original.(http.Hijacker)
	flusher, _ := original.(http.Flusher)
	return &ResponseWriterWrapper{
		ResponseWriter:    original,
		statusCodeWritten: false,
		hijacker:          hijacker,
		flusher:           flusher,
	}
}

// StatusCode returns the last written status code.
func (rw *ResponseWriterWrapper) StatusCode() int {
	return rw.statusCode
}

// WriteHeader stores the provided status code and writes it.
func (rw *ResponseWriterWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.statusCodeWritten = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write writes the provided data.
func (rw *ResponseWriterWrapper) Write(data []byte) (int, error) {
	if !rw.statusCodeWritten {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(data)
}

// Hijack calls the underlying writer's Hijack output.
func (rw *ResponseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if rw.hijacker == nil {
		return nil, nil, errors.New("Hijacker interface not supported by the wrapped ResponseWriter")
	}
	return rw.hijacker.Hijack()
}

// Flush flushes the response writer.
func (rw *ResponseWriterWrapper) Flush() {
	if rw.flusher != nil {
		rw.flusher.Flush()
	}
}
