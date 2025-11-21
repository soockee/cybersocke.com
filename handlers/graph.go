package handlers

import (
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
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

func (h *GraphHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeHTTPError(w, r, h.Log, ErrMethodNotAllowed)
		return
	}
	posts, err := h.PostService.GetPosts(r.Context())
	if err != nil {
		writeHTTPError(w, r, h.Log, err)
		return
	}
	tagCounts := h.GraphService.ComputeTagCounts(posts)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	authed := isAuthed(r)
	if err := components.TagNoteGraph(components.TagNoteGraphProps{Posts: posts, TagCounts: tagCounts, Authed: authed}).Render(r.Context(), w); err != nil {
		writeHTTPError(w, r, h.Log, err)
	}
}
