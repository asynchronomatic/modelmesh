package mesh

import (
	"fmt"
	"net/http"

	"github.com/libp2p/go-libp2p/core/network"
)

var _ http.ResponseWriter = (*streamResponseWriter)(nil)

type streamResponseWriter struct {
	s        network.Stream
	hdr      http.Header
	status   int
	wroteHdr bool
}

func (w *streamResponseWriter) Header() http.Header {
	if w.hdr == nil {
		w.hdr = make(http.Header)
	}
	return w.hdr
}

func (w *streamResponseWriter) WriteHeader(code int) {
	if w.wroteHdr {
		return
	}
	w.wroteHdr = true
	w.status = code
	fmt.Fprintf(w.s, "HTTP/1.1 %d %s\r\n", code, http.StatusText(code))
	_ = w.hdr.Write(w.s)
	_, _ = w.s.Write([]byte("\r\n"))
}

func (w *streamResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHdr {
		w.WriteHeader(http.StatusOK)
	}
	return w.s.Write(p)
}
