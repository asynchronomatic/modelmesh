package auth

import (
	"net/http"
	"strings"
)

type TokenAuth struct {
	users map[string]TokenUser
}

type TokenUser struct {
	User  string
	Group string
}

func (a *TokenAuth) DoAuth(w http.ResponseWriter, r *http.Request) (*Properties, int) {
	auth := r.Header.Get("Authorization")
	const prefix = "bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		token := strings.TrimSpace(auth[len(prefix):])

		user, ok := a.users[token]
		if !ok {
			return nil, http.StatusUnauthorized
		}

		return &Properties{
			User:  user.User,
			Group: user.Group,
		}, http.StatusOK

	}
	return nil, http.StatusUnauthorized
}

func (a *TokenAuth) AddUser(user string, group string, password string) error {
	a.users[password] = TokenUser{
		User:  user,
		Group: group,
	}
	return nil
}

func (a *TokenAuth) DeleteUser(user string) error {
	return nil
}

func NewTokenAuth() *TokenAuth {
	return &TokenAuth{
		users: make(map[string]TokenUser),
	}
}
