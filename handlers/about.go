package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
)

type AboutHandler struct {
	Log     *slog.Logger
	service *services.AboutService
}

func NewAboutHandler(service *services.AboutService, log *slog.Logger) *AboutHandler {
	return &AboutHandler{
		Log:     log,
		service: service,
	}
}

func (h *AboutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return errors.New("method not allowed")
	}
}

func (h *AboutHandler) Get(w http.ResponseWriter, r *http.Request) error {
	data := h.service.GetAbout()
	md := services.RenderMD(data)
	h.View(w, r, components.AboutViewProps{
		Content: md,
	})
	return nil
}

func (h *AboutHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}

func (h *AboutHandler) View(w http.ResponseWriter, r *http.Request, props components.AboutViewProps) {
	components.About(props).Render(r.Context(), w)
}
