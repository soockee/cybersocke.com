package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/segmentio/ksuid"
)

type Session struct {
	Next     http.Handler
	Secure   bool
	HTTPOnly bool
}

type SessionOpts func(*Session)

func ID(r *http.Request) (id string) {
	cookie, err := r.Cookie("sessionID")
	if err != nil {
		return
	}
	return cookie.Value
}

func WithSession(logger *slog.Logger, secure bool, HTTPOnly bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.Error("Recovered from panic", err, string(debug.Stack()), err)
				}
			}()

			id := ID(r)
			if id == "" {
				id = ksuid.New().String()
				http.SetCookie(w, &http.Cookie{Name: "sessionID", Value: id, Secure: secure, HttpOnly: HTTPOnly})
			}
			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)
		}

		return http.HandlerFunc(fn)
	}
}
