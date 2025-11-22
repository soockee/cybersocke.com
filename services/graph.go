package services

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/soockee/cybersocke.com/storage/graph"
	"github.com/soockee/cybersocke.com/storage/models"
)

// GraphBuilder defines the minimal interface required to build a tag graph.
// *storage.GCSStore already satisfies this.
type GraphBuilder interface {
	BuildGraph(ctx context.Context, opts graph.Options) (*graph.Graph, error)
}

// GraphService orchestrates graph option parsing and delegates build calls.
type GraphService struct {
	builder    GraphBuilder
	TagService *TagService
}

func NewGraphService(builder GraphBuilder, tagService *TagService) *GraphService {
	return &GraphService{builder: builder, TagService: tagService}
}

// ParseOptions converts query parameters into graph.Options.
// Recognized params: minSharedTags, includeTags (comma list), maxEdges.
func (gs *GraphService) ParseOptions(values url.Values) graph.Options {
	minShared := 1
	if v := values.Get("minSharedTags"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minShared = n
		}
	}
	maxEdges := 0
	if v := values.Get("maxEdges"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			maxEdges = n
		}
	}
	includeRaw := values.Get("includeTags")
	include := []string{}
	if strings.TrimSpace(includeRaw) != "" {
		include = gs.TagService.ParseSelectedTags(includeRaw)
	}
	return graph.Options{MinSharedTags: minShared, IncludeTags: include, MaxEdges: maxEdges}
}

// Build executes the underlying builder with parsed options.
func (gs *GraphService) Build(ctx context.Context, opts graph.Options) (*graph.Graph, error) {
	return gs.builder.BuildGraph(ctx, opts)
}

// ComputeTagCounts returns a map of tag -> number of posts containing that tag (duplicates in a single post ignored).
// The input is a map of slug->*Post as returned by PostService.GetPosts for efficiency.
func (gs *GraphService) ComputeTagCounts(posts map[string]*models.Post) map[string]int {
	counts := make(map[string]int)
	for _, p := range posts {
		seen := make(map[string]struct{}, len(p.Meta.Tags))
		for _, t := range p.Meta.Tags {
			if _, dup := seen[t]; dup { // ignore duplicates within same post
				continue
			}
			seen[t] = struct{}{}
			counts[t]++
		}
	}
	return counts
}
