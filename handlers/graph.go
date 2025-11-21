package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/internal/httpx"
	"github.com/soockee/cybersocke.com/services"
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

func (h *GraphHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return httpx.ErrMethodNotAllowed
	}
	posts, err := h.PostService.GetPosts(r.Context())
	if err != nil {
		return httpx.Classify(err)
	}
	tagCounts := h.GraphService.ComputeTagCounts(posts)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	authed := isAuthed(r)
	return components.TagNoteGraph(components.TagNoteGraphProps{Posts: posts, TagCounts: tagCounts, Authed: authed}).Render(r.Context(), w)
}

// containsJSON performs a loose substring match allowing Accept headers with multiple types.
// containsJSON retained only for potential future negotiation reuse (currently unused after split).
func containsJSON(accept string) bool {
	if accept == "" {
		return false
	}
	lower := strings.ToLower(accept)
	return strings.Contains(lower, "application/json")
}
