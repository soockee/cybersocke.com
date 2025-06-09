package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
)

type PostHandler struct {
	Log     *slog.Logger
	service *services.PostService
}

func NewPostHandler(service *services.PostService, log *slog.Logger) *PostHandler {
	return &PostHandler{
		service: service,
		Log:     log,
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
	data := h.service.GetPost(idStr)
	md := services.RenderMD(data)
	h.View(w, r, components.PostViewProps{
		Content: md,
	})
	return nil
}

func (h *PostHandler) Post(w http.ResponseWriter, r *http.Request) error {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return nil
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file content", http.StatusInternalServerError)
		return nil
	}

	// TODO: handle markdown content
	fmt.Println(string(content))
	return nil
}

func (h *PostHandler) View(w http.ResponseWriter, r *http.Request, props components.PostViewProps) {
	components.Post(props).Render(r.Context(), w)
}
