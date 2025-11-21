package httpx

import (
	"errors"
	"net/http"
)

// HTTPError provides typed status + public-safe message returned by handlers.
// Message should be generic (no internal details). Optional Cause retains internal error for logging.
type HTTPError struct {
	Status  int
	Message string
	Cause   error // internal details, not sent to client
}

func (e *HTTPError) Error() string { return e.Message }
func (e *HTTPError) Unwrap() error { return e.Cause }

// Constructors
func New(status int, msg string) *HTTPError { return &HTTPError{Status: status, Message: msg} }
func Internal(cause error) *HTTPError {
	return &HTTPError{Status: http.StatusInternalServerError, Message: "internal server error", Cause: cause}
}
func BadRequest(msg string, cause error) *HTTPError {
	return &HTTPError{Status: http.StatusBadRequest, Message: msg, Cause: cause}
}
func Unauthorized() *HTTPError {
	return &HTTPError{Status: http.StatusUnauthorized, Message: "unauthorized"}
}
func NotFound(msg string) *HTTPError { return &HTTPError{Status: http.StatusNotFound, Message: msg} }
func MethodNotAllowed() *HTTPError {
	return &HTTPError{Status: http.StatusMethodNotAllowed, Message: "method not allowed"}
}

// Sentinels
var (
	ErrUnauthorized     = Unauthorized()
	ErrBadRequest       = BadRequest("bad request", nil)
	ErrInternalServer   = Internal(nil)
	ErrNotFound         = NotFound("not found")
	ErrMethodNotAllowed = MethodNotAllowed()
)

// Classify converts any error into *HTTPError, preserving existing HTTPError semantics.
// Non-HTTP errors become Internal errors.
func Classify(err error) *HTTPError {
	if err == nil {
		return nil
	}
	var he *HTTPError
	if errors.As(err, &he) {
		return he
	}
	return Internal(err)
}
