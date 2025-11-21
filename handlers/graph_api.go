package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/internal/httpx"
	"github.com/soockee/cybersocke.com/services"
)

// GraphAPIHandler serves the raw tag graph JSON at /api/graph.
// Query params: minSharedTags, includeTags, maxEdges.
type GraphAPIHandler struct {
	Log          *slog.Logger
	GraphService *services.GraphService
}

func NewGraphAPIHandler(log *slog.Logger, gs *services.GraphService) *GraphAPIHandler {
	return &GraphAPIHandler{Log: log, GraphService: gs}
}

func (h *GraphAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return httpx.ErrMethodNotAllowed
	}
	opts := h.GraphService.ParseOptions(r.URL.Query())
	graph, err := h.GraphService.Build(r.Context(), opts)
	if err != nil {
		return httpx.Classify(err)
	}
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(graph); err != nil {
		return httpx.Internal(err)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
	return nil
}
