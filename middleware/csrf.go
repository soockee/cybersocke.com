package middleware

import (
	"net/http"
	"os"

	"github.com/gorilla/csrf"
)

func WithCSRF() func(http.Handler) http.Handler {
	secret := os.Getenv("CSRF_SECRET")

	return csrf.Protect(
		[]byte(secret),
		csrf.Secure(false),                 // false in development only!
		csrf.RequestHeader("X-CSRF-Token"), // Must be in CORS Allowed and Exposed Headers
	)
}
