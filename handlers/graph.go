package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
)

// GraphHandler now serves a bipartite tag↔note graph in HTML and retains JSON tag graph output.
// Routes:
//
//	GET /graph       -> Bipartite Cytoscape visualization (tags & notes)
//	GET /graph.json  -> Raw JSON tag graph (existing format)
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
	opts := h.GraphService.ParseOptions(r.URL.Query())
	// Always build tag graph for JSON requests
	if r.URL.Path == "/graph.json" || (r.URL.Path == "/graph" && r.Header.Get("Accept") == "application/json") {
		graph, err := h.GraphService.Build(r.Context(), opts)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(graph)
	}

	// HTML bipartite view: collect posts & aggregate tags
	posts, err := h.PostService.GetPosts(r.Context())
	if err != nil {
		return err
	}
	// Delegate tag count aggregation to service (domain logic)
	tagCounts := h.GraphService.ComputeTagCounts(posts)
	// Render template with spans (handled by TagNoteGraph component)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	authed := isAuthed(r)
	return components.TagNoteGraph(components.TagNoteGraphProps{Posts: posts, TagCounts: tagCounts, Authed: authed}).Render(r.Context(), w)
}
