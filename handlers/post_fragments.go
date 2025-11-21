package handlers

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

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

func (h *PostFragmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeHTTPError(w, r, h.log, ErrMethodNotAllowed)
		return
	}
	if err := h.GetFragments(w, r); err != nil {
		writeHTTPError(w, r, h.log, err)
	}
}

// GetFragments returns a batch of post fragments for a tag.
// Route: /posts/fragments?tag=observability&limit=5
func (h *PostFragmentsHandler) GetFragments(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	tag := r.URL.Query().Get("tag")
	if tag == "" {
		return BadRequest("missing tag", nil)
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		components.FragmentBatch(tag, true, nil).Render(ctx, w)
		return nil
	}
	// Aggregate fragments via FragmentBatch component (no layout wrapper).
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	var viewProps []components.PostViewProps
	for _, p := range posts {
		clean := services.StripDataview(p.Content)
		md := services.RenderMD(clean)
		families := map[string][]string{}
		for _, t := range p.Meta.Tags {
			parts := strings.SplitN(t, "/", 2)
			if len(parts) != 2 {
				continue
			}
			families[parts[0]] = append(families[parts[0]], parts[1])
		}
		for k := range families {
			sort.Strings(families[k])
		}
		props := components.PostViewProps{
			Content:     md,
			Title:       p.Meta.Name,
			Slug:        p.Meta.Slug,
			Tags:        p.Meta.Tags,
			Related:     []*storage.Post{},
			Lead:        p.Meta.Lead,
			Created:     p.Meta.Created,
			Updated:     p.Meta.Updated,
			Published:   p.Meta.Published,
			Aliases:     p.Meta.Aliases,
			TagFamilies: families,
		}
		viewProps = append(viewProps, props)
	}
	components.FragmentBatch(tag, false, viewProps).Render(ctx, w)
	return nil
}
