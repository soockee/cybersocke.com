package middleware

import (
	"net/http"

	"github.com/gorilla/csrf"
)

func WithDebugContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = csrf.PlaintextHTTPRequest(r)
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
