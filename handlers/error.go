package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// writeHTTPError centralizes error classification, logging, and response writing.
// It replaces the previous makeHTTPHandleFunc wrapper now that handlers implement http.Handler directly.
func WriteHTTPError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	if err == nil {
		return
	}
	status := http.StatusInternalServerError
	msg := err.Error()
	var he *HTTPError
	if errors.As(err, &he) {
		status = he.Status
		msg = he.Message
		// Prefer underlying cause for logging context when present.
		if he.Cause != nil {
			err = he.Cause
		}
	}
	if logger != nil {
		logger.Error("request error", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", status), slog.Any("err", err))
	}
	// Decide response format: /api prefix uses JSON; everything else plain text.
	if strings.HasPrefix(r.URL.Path, "/api") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": msg, "status": status})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}
