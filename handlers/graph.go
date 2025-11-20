package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
)

// GraphHandler provides primitive HTML & JSON views of the content tag graph.
// Routes suggestion:
//
//	GET /graph      -> HTML list representation
//	GET /graph.json -> Raw JSON graph
//
// Query params (optional): minSharedTags, includeTags (comma-separated), maxEdges.
type GraphHandler struct {
	Log          *slog.Logger
	GraphService *services.GraphService
}

func NewGraphHandler(log *slog.Logger, gs *services.GraphService) *GraphHandler {
	return &GraphHandler{Log: log, GraphService: gs}
}

func (h *GraphHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	opts := h.GraphService.ParseOptions(r.URL.Query())
	graph, err := h.GraphService.Build(r.Context(), opts)
	if err != nil {
		return err
	}
	// Content negotiation: if path ends with .json or Accept asks for JSON -> JSON
	if r.URL.Path == "/graph.json" || r.URL.Path == "/graph" && r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(graph)
	}
	// Render via templ component
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return components.Graph(components.GraphViewProps{Graph: graph, MinSharedTags: opts.MinSharedTags, IncludeTags: opts.IncludeTags}).Render(r.Context(), w)
}
