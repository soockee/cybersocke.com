package middleware

import (
	"net/http"

	"github.com/gorilla/csrf"
)

func WithCSRF(secret string, secure bool) func(http.Handler) http.Handler {
	return csrf.Protect(
		[]byte(secret),
		csrf.Secure(secure),                // true in production
		csrf.RequestHeader("X-CSRF-Token"), // Must be in CORS Allowed and Exposed Headers
	)
}
