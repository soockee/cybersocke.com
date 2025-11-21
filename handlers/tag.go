package handlers

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/internal/httpx"
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
		return httpx.ErrMethodNotAllowed
	}
	return h.Get(w, r)
}

// Get returns fragments for posts containing the tag. HTML container carries data-tag attribute.
func (h *TagPostsHandler) Get(w http.ResponseWriter, r *http.Request) error {
	tag := r.PathValue("tag")
	if tag == "" {
		return httpx.BadRequest("missing tag", nil)
	}
	limit := 10
	if lstr := r.URL.Query().Get("limit"); lstr != "" {
		if li, err := strconv.Atoi(lstr); err == nil && li > 0 {
			limit = li
		}
	}
	posts, err := h.postService.GetPostsByTag(tag, limit, r.Context())
	if err != nil {
		return httpx.Classify(err)
	}
	if len(posts) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<div class=\"fragment-batch empty\" data-tag=\"" + tag + "\"><p>No posts for tag.</p></div>"))
		return nil
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<div class=\"fragment-batch\" data-tag=\"" + tag + "\">"))
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
			Related:     []*storage.Post{},
			Lead:        p.Meta.Lead,
			Created:     p.Meta.Created,
			Updated:     p.Meta.Updated,
			Published:   p.Meta.Published,
			Aliases:     p.Meta.Aliases,
			TagFamilies: families,
		}
		components.PostFragment(props).Render(r.Context(), w)
	}
	w.Write([]byte("</div>"))
	return nil
}
