package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/soockee/cybersocke.com/internal/httpx"
	"github.com/soockee/cybersocke.com/services"
)

// AdjacencyHandler returns weighted neighboring posts for a given post slug.
// Route: /api/posts/{id}/adjacency?includeTags=tagA,tagB&minShared=1&limit=12
// If includeTags provided, only those tags are considered for shared tag calculation.
// minShared defaults to 1. limit defaults to 12 (<=0 means unlimited).
// Response: JSON { slug: string, neighbors: [ { slug, name, weight, shared_tags, date } ] }
type AdjacencyHandler struct {
	log         *slog.Logger
	postService *services.PostService
	tagService  *services.TagService
}

func NewAdjacencyHandler(posts *services.PostService, tags *services.TagService, log *slog.Logger) *AdjacencyHandler {
	return &AdjacencyHandler{log: log, postService: posts, tagService: tags}
}

type adjacencyResponse struct {
	Slug      string                  `json:"slug"`
	Neighbors []services.AdjacentPost `json:"neighbors"`
}

func (h *AdjacencyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return httpx.ErrMethodNotAllowed
	}
	slug := r.PathValue("id")
	if slug == "" {
		return httpx.BadRequest("missing slug", nil)
	}
	includeRaw := r.URL.Query().Get("includeTags")
	includeTags := h.tagService.ParseSelectedTags(includeRaw)
	includeSet := map[string]struct{}{}
	for _, t := range includeTags {
		includeSet[t] = struct{}{}
	}
	minShared := 1
	if v := r.URL.Query().Get("minShared"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minShared = n
		}
	}
	limit := 12
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	neighbors, err := services.ComputeAdjacency(h.postService, slug, includeSet, minShared, limit, r.Context())
	if err != nil {
		return httpx.Classify(err)
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(adjacencyResponse{Slug: slug, Neighbors: neighbors})
}
