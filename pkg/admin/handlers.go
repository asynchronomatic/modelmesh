package admin

import (
	"fmt"
	"net/http"
	"time"

	"modelmesh/pkg/log"
)

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	host := r.Header.Get("x-forwarded-for")
	if host == "" {
		host = r.RemoteAddr
	}
	user := "--"

	log.WithName("admin").Errorf("%s %s %d -- %s %s\n", host, user, http.StatusOK, r.Method, r.URL.Path)
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404 Not Found: %s", r.URL.Path)
}

func (s *Server) logRequest(r *http.Request, user string, start time.Time) {
	host := r.Header.Get("x-forwarded-for")
	if host == "" {
		host = r.RemoteAddr
	}
	if user == "" {
		user = "--"
	}

	d := time.Since(start).Round(time.Millisecond)
	log.WithName("admin").Infof("%s %s %s %s %s\n", host, d.String(), user, r.Method, r.RequestURI)
}
