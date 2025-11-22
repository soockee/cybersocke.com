package jsonhandler

import (
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/jsonresp"
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

// GetGraph godoc
// @Summary Get posts/tag graph
// @Description Returns a weighted graph of posts and tags, filtered by optional query parameters.
// @Tags graph
// @Produce json
// @Param minSharedTags query int false "Minimum shared tags between posts"
// @Param includeTags query string false "Comma-separated list of tags to include"
// @Param maxEdges query int false "Maximum number of edges to include"
// @Success 200 {object} interface{} "Graph JSON structure"
// @Failure 400 {object} map[string]string "Bad request or invalid parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/graph [get]
func (h *GraphAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}
	opts := h.GraphService.ParseOptions(r.URL.Query())
	graph, err := h.GraphService.Build(r.Context(), opts)
	if err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
		return
	}
	if err := jsonresp.WritePretty(w, http.StatusOK, graph); err != nil {
		handlers.WriteHTTPError(w, r, h.Log, handlers.Internal(err))
		return
	}
}
