package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
)

type HomeHandler struct {
	Log     *slog.Logger
	service *services.PostService
}

func NewHomeHandler(service *services.PostService, log *slog.Logger) *HomeHandler {
	return &HomeHandler{
		Log: log,
		service: service,
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
	posts := h.service.GetPosts()
	h.View(w, r, components.HomeViewProps{
		Posts: posts,
	})
	return nil
}

func (h *HomeHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}

func (h *HomeHandler) View(w http.ResponseWriter, r *http.Request, props components.HomeViewProps) {
	components.Home(props).Render(r.Context(), w)
}
