package handlers

import "net/http"

// HTTPError represents an application error with a status code.
// Message is sent to client; Cause is logged if present.
type HTTPError struct {
	Status  int
	Message string
	Cause   error
}

func (e *HTTPError) Error() string { return e.Message }

func BadRequest(message string, cause error) error {
	return &HTTPError{Status: http.StatusBadRequest, Message: message, Cause: cause}
}

func NotFound(message string) error {
	return &HTTPError{Status: http.StatusNotFound, Message: message}
}

func Internal(cause error) error {
	return &HTTPError{Status: http.StatusInternalServerError, Message: "internal error", Cause: cause}
}

var ErrMethodNotAllowed = &HTTPError{Status: http.StatusMethodNotAllowed, Message: "method not allowed"}
