package admin

import (
	"encoding/json"
	"net/http"
)

type JsonRPC struct {
	w    http.ResponseWriter
	r    *http.Request
	user string
}

func (c *JsonRPC) Request() *http.Request {
	return c.r
}

func (c *JsonRPC) User() string {
	return c.user
}

func (c *JsonRPC) PathVar(name string) string {
	return c.r.PathValue(name)
}

func (c *JsonRPC) GetObject(obj any) error {
	defer c.r.Body.Close()
	return json.NewDecoder(c.r.Body).Decode(obj)
}

func (c *JsonRPC) ReplyObject(obj any) error {
	c.w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.w).Encode(obj)
}

func (c *JsonRPC) Error(code int, msg string) {
	http.Error(c.w, msg, code)
}
