package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/session"
)

func WithAuthentication(authService *services.AuthService, sessionStore *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Load session first
			s, err := sessionStore.Get(r, "cybersocke-session")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			token, ok := s.Values["id_token"].(string)
			if !ok || token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Verify token
			verified_token, err := authService.Verify(token, r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), session.IdTokenKey, verified_token)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
