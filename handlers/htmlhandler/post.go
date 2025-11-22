package htmlhandler

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage/models"
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

func (h *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/fragment") {
		if err := h.Fragment(w, r); err != nil {
			handlers.WriteHTTPError(w, r, h.Log, err)
		}
		return
	}

	if err := h.Get(w, r); err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
	}
}

// GetPostPage godoc
// @Summary View a blog post
// @Description Renders the full HTML explorer layout for a single blog post, including adjacency metadata.
// @Tags public
// @Produce text/html
// @Param id path string true "Post identifier or slug"
// @Success 200 {string} string "HTML page"
// @Failure 404 {object} map[string]string "Post not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /posts/{id} [get]
func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	post, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return err
	}
	if post == nil {
		return handlers.NotFound("post not found")
	}
	cleaned := services.StripDataview(post.Content)
	md := services.RenderMD(cleaned)
	// Build adjacency via service (include all tags, minShared=1, limit=12)
	neighbors, err := services.ComputeAdjacency(h.postService, post.Meta.Slug, map[string]struct{}{}, 1, 12, r.Context())
	if err != nil {
		return err
	}
	entries := make([]components.AdjacencyEntry, 0, len(neighbors))
	for _, n := range neighbors {
		entries = append(entries, components.AdjacencyEntry{Slug: n.Slug, Name: n.Name, Weight: n.Weight, SharedTags: n.SharedTags, Date: n.Date})
	}
	// Created already parsed during validation / load
	createdTs := post.Meta.Created
	// Group tags by family for refined display
	families := map[string][]string{}
	for _, t := range post.Meta.Tags {
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
		Title:       post.Meta.Name,
		Slug:        post.Meta.Slug,
		Tags:        post.Meta.Tags,
		Related:     []*models.Post{},
		Lead:        post.Meta.Lead,
		Created:     createdTs,
		Updated:     post.Meta.Updated,
		Published:   post.Meta.Published,
		TagFamilies: families,
	}
	props.Authed = handlers.IsAuthed(r)
	props.User = handlers.NavUser(r)
	if err := htmlresp.Render(w, http.StatusOK, r.Context(), components.ExplorerLayout(components.ExplorerLayoutProps{Post: props, Adjacency: entries, Tags: []string{}, Authed: props.Authed, User: props.User})); err != nil {
		return err
	}
	return nil
}

// GetPostFragment godoc
// @Summary Get post fragment HTML
// @Description Returns a lightweight HTML snippet for overlay navigation (without layout) for a post.
// @Tags public
// @Produce text/html
// @Param id path string true "Post identifier or slug"
// @Success 200 {string} string "HTML fragment"
// @Failure 404 {object} map[string]string "Post not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /posts/{id}/fragment [get]
func (h *PostHandler) Fragment(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	post, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return err
	}
	related, _ := h.postService.GetRelatedPosts(post.Meta.Slug, 12, r.Context()) // ignore classification for related fetch errors
	// Render markdown for fragment (same as full post view)
	cleaned := services.StripDataview(post.Content)
	md := services.RenderMD(cleaned)
	// Build tag families
	families := map[string][]string{}
	for _, t := range post.Meta.Tags {
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
		Title:       post.Meta.Name,
		Slug:        post.Meta.Slug,
		Tags:        post.Meta.Tags,
		Related:     related, // Assuming related is of type []*models.Post
		Lead:        post.Meta.Lead,
		Created:     post.Meta.Created,
		Updated:     post.Meta.Updated,
		Published:   post.Meta.Published,
		TagFamilies: families,
	}
	return htmlresp.Render(w, http.StatusOK, r.Context(), components.PostFragment(props))
}
