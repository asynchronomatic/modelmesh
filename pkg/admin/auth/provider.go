package auth

import (
	"net/http"
)

type Properties struct {
	User  string
	Group string
}

type Provider interface {
	DoAuth(w http.ResponseWriter, r *http.Request) (*Properties, int)
}
