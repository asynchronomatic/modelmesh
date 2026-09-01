package api

import (
	"modelmesh/pkg/jsonclient"
)

type Error = jsonclient.RequestError

func NewError(code int, msg string) *Error {
	return jsonclient.NewRequestError(code, msg)
}
