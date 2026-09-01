package admin

import (
	"encoding/json"
	"net/http"
)

type Context struct {
	w    http.ResponseWriter
	r    *http.Request
	user string
}

func (c *Context) Request() *http.Request {
	return c.r
}

func (c *Context) User() string {
	return c.user
}

func (c *Context) PathVar(name string) string {
	return c.r.PathValue(name)
}

func (c *Context) GetObject(obj any) error {
	defer c.r.Body.Close()
	return json.NewDecoder(c.r.Body).Decode(obj)
}

func (c *Context) ReplyObject(obj any) error {
	c.w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.w).Encode(obj)
}

func (c *Context) Error(code int, msg string) {
	http.Error(c.w, msg, code)
}
