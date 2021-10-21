package ws

import (
	"io"
	"net/http"
)

type responseWriter struct {
	header http.Header
	code   int
	closed chan bool
	io.Writer
}

func newResponseWriter(w io.Writer) *responseWriter {
	return &responseWriter{
		Writer: w,
		header: http.Header{},
		closed: make(chan bool, 1),
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *responseWriter) Header() http.Header {
	return w.header
}

func (w *responseWriter) WriteHeader(code int) {
	w.code = code
}

func (w *responseWriter) CloseNotify() <-chan bool {
	return w.closed
}

func (w *responseWriter) Flush() {}
