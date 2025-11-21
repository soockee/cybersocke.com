package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

type PostFragmentsHandler struct {
	log         *slog.Logger
	postService *services.PostService
}

// NewPostFragmentsHandler constructs a handler for returning batches of post fragments filtered by tag.
func NewPostFragmentsHandler(post *services.PostService, log *slog.Logger) *PostFragmentsHandler {
	return &PostFragmentsHandler{log: log, postService: post}
}

func (h *PostFragmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.New("method not allowed")
	}
	return h.GetFragments(w, r)
}

// GetFragments returns a batch of post fragments for a tag.
// Route: /posts/fragments?tag=observability&limit=5
func (h *PostFragmentsHandler) GetFragments(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	tag := r.URL.Query().Get("tag")
	if tag == "" {
		return errors.New("missing tag")
	}
	limit := 5
	if lstr := r.URL.Query().Get("limit"); lstr != "" {
		if li, err := strconv.Atoi(lstr); err == nil && li > 0 {
			limit = li
		}
	}
	posts, err := h.postService.GetPostsByTag(tag, limit, ctx)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<div class=\"fragment-batch empty\" data-tag=\"" + tag + "\"><p>No posts for tag.</p></div>"))
		return nil
	}
	// Aggregate fragments. We intentionally avoid layout wrapper.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<div class=\"fragment-batch\" data-tag=\"" + tag + "\">"))
	for _, p := range posts {
		components.PostFragment(components.PostFragmentProps{Post: p, Related: []*storage.Post{}}).Render(ctx, w)
	}
	w.Write([]byte("</div>"))
	return nil
}
