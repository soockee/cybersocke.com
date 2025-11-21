package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

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

func (h *AdjacencyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeHTTPError(w, r, h.log, fmt.Errorf("GET method not allowed"))
		return
	}
	slug := r.PathValue("id")
	if slug == "" {
		writeHTTPError(w, r, h.log, fmt.Errorf("missing slug"))
		return
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
		writeHTTPError(w, r, h.log, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(adjacencyResponse{Slug: slug, Neighbors: neighbors}); err != nil {
		writeHTTPError(w, r, h.log, Internal(err))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
