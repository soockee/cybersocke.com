package htmlhandler

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage/models"
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
		handlers.WriteHTTPError(w, r, h.log, handlers.ErrMethodNotAllowed)
		return
	}
	if err := h.GetFragments(w, r); err != nil {
		handlers.WriteHTTPError(w, r, h.log, err)
	}
}

// GetFragments returns a batch of post fragments for a tag.
// Route: /posts/fragments?tag=observability&limit=5
// GetPostFragments godoc
// @Summary Fetch post fragments by tag
// @Description Returns a batch of HTML fragments for posts matching a given tag.
// @Tags public
// @Produce text/html
// @Param tag query string true "Tag identifier"
// @Param limit query int false "Maximum number of fragments to return (default 5)"
// @Success 200 {string} string "HTML fragment batch"
// @Failure 400 {object} map[string]string "Missing tag query"
// @Failure 404 {string} string "No posts found for tag"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /posts/fragments [get]
func (h *PostFragmentsHandler) GetFragments(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	tag := r.URL.Query().Get("tag")
	if tag == "" {
		return handlers.BadRequest("missing tag", nil)
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
		if err := htmlresp.Render(w, http.StatusNotFound, ctx, components.FragmentBatch(tag, true, nil)); err != nil {
			return err
		}
		return nil
	}
	// Aggregate fragments via FragmentBatch component (no layout wrapper).
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
			Related:     []*models.Post{},
			Lead:        p.Meta.Lead,
			Created:     p.Meta.Created,
			Updated:     p.Meta.Updated,
			Published:   p.Meta.Published,
			TagFamilies: families,
		}
		viewProps = append(viewProps, props)
	}
	if err := htmlresp.Render(w, http.StatusOK, ctx, components.FragmentBatch(tag, false, viewProps)); err != nil {
		return err
	}
	return nil
}
