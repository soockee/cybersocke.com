package handlers

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
)

type PostHandler struct {
	Log *slog.Logger
}

func NewPostHandler(log *slog.Logger) *PostHandler {
	return &PostHandler{
		Log: log,
	}
}

func (h *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return errors.New("method not allowed")
	}
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) error {
	idStr := mux.Vars(r)["id"]
	data := services.GetPost(idStr)
	md := services.RenderMD(data)
	h.View(w, r, ViewProps{
		Content: md,
	})
	return nil
}

func (h *PostHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}

type ViewProps struct {
	Content bytes.Buffer
}

func (h *PostHandler) View(w http.ResponseWriter, r *http.Request, props ViewProps) {
	components.Post(props.Content).Render(r.Context(), w)
}
