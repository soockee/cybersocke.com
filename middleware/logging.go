package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for logging.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

// Write captures implicit 200 responses where handlers write a body without calling WriteHeader.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

func WithLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.Error("Recovered from panic", err, string(debug.Stack()), err)
				}
			}()

			start := time.Now()
			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)
			// Fallback: if no header/body write occurred (status still zero), assume 200 for successful handlers.
			if wrapped.status == 0 {
				wrapped.status = http.StatusOK
			}
			status := wrapped.status
			path := r.URL.EscapedPath()

			logger.Info("http request", slog.Int("status", status), slog.String("method", r.Method), slog.String("path", path), slog.Duration("duration", time.Since(start)))
		}

		return http.HandlerFunc(fn)
	}
}
