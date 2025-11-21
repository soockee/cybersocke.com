package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/session"
)

// WithAuthentication verifies a Firebase ID token stored in the session and attaches it to the context.
// Adds structured debug logging for failure reasons.
func WithAuthentication(authService *services.AuthService, sessionStore *sessions.CookieStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Load session first
			s, err := sessionStore.Get(r, "cybersocke-session")
			if err != nil {
				if logger != nil {
					logger.Error("auth session load failed", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal server error"))
				return
			}
			token, ok := s.Values["id_token"].(string)
			if !ok || token == "" {
				if logger != nil {
					logger.Info("auth missing token", slog.String("path", r.URL.Path), slog.String("method", r.Method))
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			// Verify token
			verified_token, err := authService.Verify(token, r.Context())
			if err != nil {
				if logger != nil {
					logger.Info("auth token verify failed", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}

			ctx := context.WithValue(r.Context(), session.IdTokenKey, verified_token)
			if logger != nil {
				logger.Debug("auth token verified", slog.String("uid", verified_token.UID), slog.String("path", r.URL.Path), slog.String("method", r.Method))
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
