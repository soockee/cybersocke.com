package services

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/soockee/cybersocke.com/storage"
)

// GraphBuilder defines the minimal interface required to build a tag graph.
// *storage.GCSStore already satisfies this.
type GraphBuilder interface {
	BuildTagGraph(ctx context.Context, opts storage.TagGraphOptions) (*storage.TagGraph, error)
}

// GraphService orchestrates graph option parsing and delegates build calls.
type GraphService struct {
	builder    GraphBuilder
	TagService *TagService
}

func NewGraphService(builder GraphBuilder, tagService *TagService) *GraphService {
	return &GraphService{builder: builder, TagService: tagService}
}

// ParseOptions converts query parameters into TagGraphOptions.
// Recognized params: minSharedTags, includeTags (comma list), maxEdges.
func (gs *GraphService) ParseOptions(values url.Values) storage.TagGraphOptions {
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
	return storage.TagGraphOptions{MinSharedTags: minShared, IncludeTags: include, MaxEdges: maxEdges}
}

// Build executes the underlying builder with parsed options.
func (gs *GraphService) Build(ctx context.Context, opts storage.TagGraphOptions) (*storage.TagGraph, error) {
	return gs.builder.BuildTagGraph(ctx, opts)
}
