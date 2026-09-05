package api

import (
	"github.com/asynchronomatic/speakeasy/pkg/jsonclient"
)

type Error = jsonclient.RequestError

func NewError(code int, msg string) *Error {
	return jsonclient.NewRequestError(code, msg)
}
