package logger

import "net/http"

// ResponseWrapper is a type that wraps a response.
type ResponseWrapper interface {
	InnerResponse() http.ResponseWriter
}

// NewResponseWriter creates a new response writer.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		innerResponse: w,
	}
}

// ResponseWriter a better response writer
type ResponseWriter struct {
	innerResponse http.ResponseWriter
	statusCode    int
	contentLength int
}

// Write writes the data to the response.
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	bytesWritten, err := rw.innerResponse.Write(b)
	rw.contentLength = rw.contentLength + bytesWritten
	return bytesWritten, err
}

// Header accesses the response header collection.
func (rw *ResponseWriter) Header() http.Header {
	return rw.innerResponse.Header()
}

// WriteHeader is actually a terrible name and this writes the status code.
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.innerResponse.WriteHeader(code)
}

// InnerWriter returns the backing writer.
func (rw *ResponseWriter) InnerWriter() http.ResponseWriter {
	return rw.innerResponse
}

// Flush is a no op on raw response writers.
func (rw *ResponseWriter) Flush() error {
	return nil
}

// StatusCode returns the status code.
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

// ContentLength returns the content length
func (rw *ResponseWriter) ContentLength() int {
	return rw.contentLength
}
