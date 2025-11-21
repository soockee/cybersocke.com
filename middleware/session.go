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
// sessionWriter saves the session BEFORE headers are flushed. It intercepts WriteHeader.
type sessionWriter struct {
	http.ResponseWriter
	sess   *sessions.Session
	req    *http.Request
	logger *slog.Logger
	saved  bool
}

func (sw *sessionWriter) save() {
	if sw.saved || sw.sess == nil {
		return
	}
	if err := sw.sess.Save(sw.req, sw.ResponseWriter); err != nil && sw.logger != nil {
		sw.logger.Error("session save failed", slog.String("path", sw.req.URL.Path), slog.String("method", sw.req.Method), slog.Any("err", err))
	}
	sw.saved = true
}

func (sw *sessionWriter) WriteHeader(code int) {
	// Ensure cookie set before header flush
	sw.save()
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *sessionWriter) Write(b []byte) (int, error) {
	// If body write occurs without explicit WriteHeader first
	sw.save()
	return sw.ResponseWriter.Write(b)
}

// WithSession attaches session to context and ensures modifications are persisted before any header/body write.
func WithSession(store *sessions.CookieStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, err := store.Get(r, "cybersocke-session")
			if err != nil && logger != nil {
				logger.Error("session load failed", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
			}
			ctx := context.WithValue(r.Context(), session.SessionKey, s)
			sw := &sessionWriter{ResponseWriter: w, sess: s, req: r, logger: logger}
			next.ServeHTTP(sw, r.WithContext(ctx))
			// Edge case: handler wrote nothing (no WriteHeader/Write); persist now if not yet saved.
			sw.save()
		})
	}
}
