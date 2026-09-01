package jsonclient

import (
	"errors"
	"fmt"
)

type RequestError struct {
	code int
	err  error
}

func (re *RequestError) Code() int {
	return re.code
}

func (re *RequestError) Message() string {
	return re.err.Error()
}

func (re *RequestError) Error() string {
	return fmt.Sprintf("%d:%s", re.code, re.err)
}

func NewRequestError(code int, err string) *RequestError {
	return &RequestError{
		code: code,
		err:  errors.New(err),
	}
}

type RequestStatus struct {
	Code   int
	Status string
}

func NewRequestFromError(code int, err error) *RequestError {
	return &RequestError{
		code: code,
		err:  err,
	}
}
