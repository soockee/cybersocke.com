package htmlhandler

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage/models"
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

func (h *TagPostsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.log, handlers.ErrMethodNotAllowed)
		return
	}
	if err := h.Get(w, r); err != nil {
		handlers.WriteHTTPError(w, r, h.log, err)
	}
}

// GetTagPosts godoc
// @Summary Fetch posts for a tag
// @Description Returns HTML fragments for posts containing the given tag.
// @Tags public
// @Produce text/html
// @Param tag path string true "Tag identifier"
// @Param limit query int false "Maximum posts to return (default 10)"
// @Success 200 {string} string "HTML fragments"
// @Failure 400 {object} map[string]string "Missing tag"
// @Failure 404 {string} string "No posts found for tag"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tags/{tag}/posts [get]
func (h *TagPostsHandler) Get(w http.ResponseWriter, r *http.Request) error {
	tag := r.PathValue("tag")
	if tag == "" {
		return handlers.BadRequest("missing tag", nil)
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		components.FragmentBatch(tag, true, nil).Render(r.Context(), w)
		return nil
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	var viewProps []components.PostViewProps
	for _, p := range posts {
		// Render markdown + build tag families for each fragment
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
	components.FragmentBatch(tag, false, viewProps).Render(r.Context(), w)
	return nil
}
