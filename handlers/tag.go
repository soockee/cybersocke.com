package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

// TagPostsHandler serves lightweight post fragments filtered by a single tag.
// Route: /tags/{tag}/posts?limit=5
type TagPostsHandler struct {
	log         *slog.Logger
	postService *services.PostService
}

func NewTagPostsHandler(posts *services.PostService, log *slog.Logger) *TagPostsHandler {
	return &TagPostsHandler{log: log, postService: posts}
}

func (h *TagPostsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.New("method not allowed")
	}
	return h.Get(w, r)
}

// Get returns fragments for posts containing the tag. HTML container carries data-tag attribute.
func (h *TagPostsHandler) Get(w http.ResponseWriter, r *http.Request) error {
	tag := mux.Vars(r)["tag"]
	if tag == "" {
		return errors.New("missing tag")
	}
	limit := 10
	if lstr := r.URL.Query().Get("limit"); lstr != "" {
		if li, err := strconv.Atoi(lstr); err == nil && li > 0 {
			limit = li
		}
	}
	posts, err := h.postService.GetPostsByTag(tag, limit, r.Context())
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<div class=\"fragment-batch empty\" data-tag=\"" + tag + "\"><p>No posts for tag.</p></div>"))
		return nil
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<div class=\"fragment-batch\" data-tag=\"" + tag + "\">"))
	for _, p := range posts {
		components.PostFragment(components.PostFragmentProps{Post: p, Related: []*storage.Post{}}).Render(r.Context(), w)
	}
	w.Write([]byte("</div>"))
	return nil
}
