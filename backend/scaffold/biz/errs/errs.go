// Package errs holds the errors a client is meant to see. Services return them;
// handlers pass any error to resp.Fail, which turns an *Error into its status,
// code and message and everything else into a logged 500.
package errs

import "net/http"

const (
	CodeInternal     = 1000
	CodeInvalid      = 1001
	CodeUnauthorized = 1002
	CodeForbidden    = 1003
	CodeNotFound     = 1004
	CodeConflict     = 1005
)

type Error struct {
	Status int
	Code   int32
	Msg    string
}

func (e *Error) Error() string { return e.Msg }

func Invalid(msg string) *Error      { return &Error{http.StatusBadRequest, CodeInvalid, msg} }
func Unauthorized(msg string) *Error { return &Error{http.StatusUnauthorized, CodeUnauthorized, msg} }
func Forbidden(msg string) *Error    { return &Error{http.StatusForbidden, CodeForbidden, msg} }
func NotFound(msg string) *Error     { return &Error{http.StatusNotFound, CodeNotFound, msg} }
func Conflict(msg string) *Error     { return &Error{http.StatusConflict, CodeConflict, msg} }
