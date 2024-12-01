package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
)

type HomeHandler struct {
	Log *slog.Logger
}

func NewHomeHandler(log *slog.Logger) *HomeHandler {
	return &HomeHandler{
		Log: log,
	}
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return errors.New("method not allowed")
	}
}

func (h *HomeHandler) Get(w http.ResponseWriter, r *http.Request) error {
	h.View(w, r)
	return nil
}

func (h *HomeHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}

func (h *HomeHandler) View(w http.ResponseWriter, r *http.Request) {
	components.Home().Render(r.Context(), w)
}
