package htmlhandler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage/models"
)

// GraphHandler serves the bipartite tag↔note graph.
// Single consolidated endpoint:
//
//	GET /graph with Accept: application/json -> Raw JSON graph
//	GET /graph (default Accept)              -> HTML visualization
//
// Query params forwarded to tag graph JSON: minSharedTags, includeTags, maxEdges.
type GraphHandler struct {
	Log          *slog.Logger
	GraphService *services.GraphService
	PostService  *services.PostService
}

func NewGraphHandler(log *slog.Logger, gs *services.GraphService, ps *services.PostService) *GraphHandler {
	return &GraphHandler{Log: log, GraphService: gs, PostService: ps}
}

// GetGraphPage godoc
// @Summary Graph explorer page
// @Description Renders the interactive tag-note graph visualization for published posts.
// @Tags graph
// @Produce text/html
// @Success 200 {string} string "HTML page"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /graph [get]
func (h *GraphHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}
	opts := h.GraphService.ParseOptions(r.URL.Query())
	posts, err := h.selectPostsForGraph(r.Context(), opts.IncludeTags)
	if err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
		return
	}
	tagCounts := h.GraphService.ComputeTagCounts(posts)
	authed := handlers.IsAuthed(r)
	navUser := handlers.NavUser(r)
	filters := components.GraphFilterInfo{
		IncludeTags:   opts.IncludeTags,
		MinSharedTags: opts.MinSharedTags,
		MaxEdges:      opts.MaxEdges,
		PostCount:     len(posts),
		TagCount:      len(tagCounts),
		HasFilter:     len(opts.IncludeTags) > 0,
	}
	props := components.TagNoteGraphProps{
		Posts:     posts,
		TagCounts: tagCounts,
		Authed:    authed,
		Filters:   filters,
		User:      navUser,
	}
	if err := htmlresp.Render(w, http.StatusOK, r.Context(), components.TagNoteGraph(props)); err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
	}
}

func (h *GraphHandler) selectPostsForGraph(ctx context.Context, includeTags []string) (map[string]*models.Post, error) {
	if len(includeTags) == 0 {
		return h.PostService.GetPosts(ctx)
	}
	return h.PostService.GetPostsByTags(includeTags, false, ctx)
}
