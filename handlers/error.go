package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
)

type ErrorHandler struct {
	Log     *slog.Logger
	Service *services.AuthService
}

func NewErrorHandler(log *slog.Logger) *ErrorHandler {
	return &ErrorHandler{
		Log: log,
	}
}

func (h *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return errors.New("method not allowed")
	}
}

func (h *ErrorHandler) Get(w http.ResponseWriter, r *http.Request) error {
	// get error from flash or context
	s := middleware.GetSession(r)
	errs := []string{}
	if _, ok := s.Values["errors"]; ok {
		errs = s.Values["errors"].([]string)
	}

	h.View(w, r, components.ErrorViewProps{
		Messages: errs,
	})
	return nil
}

func (h *ErrorHandler) View(w http.ResponseWriter, r *http.Request, props components.ErrorViewProps) {
	components.Error(props).Render(r.Context(), w)
}
