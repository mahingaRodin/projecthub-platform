// Package apperror defines errors that carry an HTTP status. Services return them; controllers and
// middleware translate them into JSON responses.
package apperror

import (
	"errors"
	"net/http"
)

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

func New(status int, message string) *Error { return &Error{Status: status, Message: message} }

func BadRequest(msg string) *Error   { return New(http.StatusBadRequest, msg) }
func Unauthorized(msg string) *Error { return New(http.StatusUnauthorized, msg) }
func Forbidden(msg string) *Error    { return New(http.StatusForbidden, msg) }
func NotFound(msg string) *Error     { return New(http.StatusNotFound, msg) }
func Conflict(msg string) *Error     { return New(http.StatusConflict, msg) }

// HTTP maps any error to a status code and a message that is safe to show to clients.
// Errors that are not *Error become 500 so internal details are never leaked.
func HTTP(err error) (int, string) {
	var e *Error
	if errors.As(err, &e) {
		return e.Status, e.Message
	}
	return http.StatusInternalServerError, "internal server error"
}
