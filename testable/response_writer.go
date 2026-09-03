package testable

import (
	"bytes"
	"fmt"
	"net/http"
)

var _ http.ResponseWriter = (*testResponseWriter)(nil)

type testResponseWriter struct {
	s        *bytes.Buffer
	hdr      http.Header
	status   int
	wroteHdr bool
}

func (w *testResponseWriter) Header() http.Header {
	if w.hdr == nil {
		w.hdr = make(http.Header)
	}
	return w.hdr
}

func (w *testResponseWriter) WriteHeader(code int) {
	if w.wroteHdr {
		return
	}
	w.wroteHdr = true
	w.status = code
	_, _ = fmt.Fprintf(w.s, "HTTP/1.1 %d %s\r\n", code, http.StatusText(code))
	_ = w.hdr.Write(w.s)
	_, _ = w.s.Write([]byte("\r\n"))
}

func (w *testResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHdr {
		w.WriteHeader(http.StatusOK)
	}
	return w.s.Write(p)
}

func NewTestResponseWriter() *testResponseWriter {
	return &testResponseWriter{
		s: bytes.NewBuffer(nil),
	}
}
