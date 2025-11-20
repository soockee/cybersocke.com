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
	Log         *slog.Logger
	postService *services.PostService
}

func NewPostHandler(postService *services.PostService, log *slog.Logger) *PostHandler {
	return &PostHandler{
		Log:         log,
		postService: postService,
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
	data, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return err
	}
	md := services.RenderMD(data)
	h.View(w, r, components.PostViewProps{
		Content: md,
	})
	return nil
}

func (h *PostHandler) Post(w http.ResponseWriter, r *http.Request) error {
	file, _, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// TODO: handle markdown content
	fmt.Println(string(content))

	// Role & authentication enforced via middleware; proceed to create
	if err := h.postService.CreatePost(content, r.Context()); err != nil {
		return err
	}
	return nil
}

func (h *PostHandler) View(w http.ResponseWriter, r *http.Request, props components.PostViewProps) {
	components.Post(props).Render(r.Context(), w)
}

// hasWriterRole determines if the token has permission to write posts.
// Accepts role patterns: role==writer/admin, roles slice contains writer/admin, writer=true or admin=true.
// Role enforcement moved to middleware.WithRole("writer") for POST /posts.
