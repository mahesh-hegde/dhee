package server

import (
	"bytes"
	"net/http"
)

// BufferedResponseWriter implements http.ResponseWriter by buffering all writes to memory.
// This allows us to capture the full response (status, headers, body) before sending it to Lambda.
// It enforces a maximum response size to prevent unbounded memory consumption.
type BufferedResponseWriter struct {
	StatusCode   int
	Headers      http.Header
	Body         bytes.Buffer
	committed    bool
	sizeLimit    int64
	bytesWritten int64
	truncated    bool
}

// NewBufferedResponseWriter creates a new buffered response writer with default 6MB limit.
func NewBufferedResponseWriter() *BufferedResponseWriter {
	return NewBufferedResponseWriterWithLimit(6 * 1024 * 1024) // 6MB Lambda limit
}

// NewBufferedResponseWriterWithLimit creates a new buffered response writer with a custom size limit.
// limit is in bytes. Use 0 for unlimited (not recommended for Lambda).
func NewBufferedResponseWriterWithLimit(limit int64) *BufferedResponseWriter {
	return &BufferedResponseWriter{
		StatusCode: http.StatusOK,
		Headers:    make(http.Header),
		committed:  false,
		sizeLimit:  limit,
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (w *BufferedResponseWriter) Header() http.Header {
	return w.Headers
}

// Write writes the data to the connection as part of an HTTP reply.
// If the write would exceed the size limit, data is silently dropped and truncated flag is set.
// WriteHeader is called with http.StatusOK if not yet called.
func (w *BufferedResponseWriter) Write(b []byte) (int, error) {
	if !w.committed {
		w.WriteHeader(http.StatusOK)
	}

	// Check if we're already truncated
	if w.truncated {
		return len(b), nil // Pretend we wrote it to keep the handler happy
	}

	// Check if this write would exceed the limit
	if w.sizeLimit > 0 && w.bytesWritten+int64(len(b)) > w.sizeLimit {
		w.truncated = true
		return len(b), nil // Pretend we wrote it
	}

	n, err := w.Body.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

// WriteHeader sends an HTTP response header with the provided status code.
// It can be called multiple times, but only the first call matters (to match net/http behavior).
func (w *BufferedResponseWriter) WriteHeader(statusCode int) {
	if w.committed {
		return // Ignore subsequent calls
	}
	w.StatusCode = statusCode
	w.committed = true
}

// Flush is a no-op for buffered writer (buffering to memory, not streaming).
func (w *BufferedResponseWriter) Flush() {
	// No-op for buffered responses
}

// IsTruncated returns true if the response was truncated due to size limit.
func (w *BufferedResponseWriter) IsTruncated() bool {
	return w.truncated
}

// BytesWritten returns the number of bytes written to the response body.
func (w *BufferedResponseWriter) BytesWritten() int64 {
	return w.bytesWritten
}
