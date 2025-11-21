package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/soockee/cybersocke.com/session"
)

// WithSession attaches the session to the context and persists it after handler execution.
// Errors during save are ignored (non-critical for response); this keeps wrapper logic slim.
func WithSession(store *sessions.CookieStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, err := store.Get(r, "cybersocke-session")
			if err != nil && logger != nil {
				logger.Error("session load failed", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
			}
			ctx := context.WithValue(r.Context(), session.SessionKey, s)
			next.ServeHTTP(w, r.WithContext(ctx))
			if err := s.Save(r, w); err != nil && logger != nil {
				logger.Error("session save failed", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
			}
		})
	}
}
