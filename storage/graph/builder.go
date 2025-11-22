package graph

import (
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/soockee/cybersocke.com/storage/models"
	"github.com/soockee/cybersocke.com/storage/tags"
)

// Options controls graph construction.
type Options struct {
	MinSharedTags int
	IncludeTags   []string
	MaxEdges      int
}

// Edge represents an undirected edge between two posts.
type Edge struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	SharedTags []string `json:"shared_tags"`
	Weight     int      `json:"weight"`
}

// Graph bundles posts, their edges, and a snapshot of the tag index used.
type Graph struct {
	Posts    []*models.Post      `json:"posts"`
	Edges    []Edge              `json:"edges"`
	TagIndex map[string][]string `json:"tag_index"`
}

// Builder provides incremental graph construction with caching.
type Builder struct {
	mu         sync.RWMutex
	edgeMap    map[string]*Edge
	opts       Options
	graphReady bool
}

// NewBuilder creates a new graph builder.
func NewBuilder(opts Options) *Builder {
	if opts.MinSharedTags < 1 {
		opts.MinSharedTags = 1
	}
	return &Builder{
		edgeMap: make(map[string]*Edge),
		opts:    opts,
	}
}

// Build constructs a complete graph from all posts.
func (b *Builder) Build(posts map[string]*models.Post, tagIndex *tags.Index, opts Options) *Graph {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.graphReady && reflect.DeepEqual(b.opts, opts) {
		return b.buildSnapshot(posts, tagIndex)
	}

	b.opts = opts
	if b.opts.MinSharedTags < 1 {
		b.opts.MinSharedTags = 1
	}
	b.edgeMap = make(map[string]*Edge)

	filter := map[string]struct{}{}
	for _, t := range b.opts.IncludeTags {
		filter[strings.TrimSpace(t)] = struct{}{}
	}

	for _, post := range posts {
		for _, tag := range post.Meta.Tags {
			if len(filter) > 0 {
				if _, ok := filter[tag]; !ok {
					continue
				}
			}
			relatedSlugs := tagIndex.GetRelatedSlugs([]string{tag}, post.Meta.Slug)
			for other := range relatedSlugs {
				if other == post.Meta.Slug {
					continue
				}
				slugA, slugB := post.Meta.Slug, other
				if slugA > slugB {
					slugA, slugB = slugB, slugA
				}
				key := slugA + "|" + slugB
				edge, exists := b.edgeMap[key]
				if !exists {
					edge = &Edge{From: slugA, To: slugB}
					b.edgeMap[key] = edge
				}
				hasTag := false
				for _, existing := range edge.SharedTags {
					if existing == tag {
						hasTag = true
						break
					}
				}
				if !hasTag {
					edge.SharedTags = append(edge.SharedTags, tag)
				}
			}
		}
	}

	b.graphReady = true
	return b.buildSnapshot(posts, tagIndex)
}

func (b *Builder) buildSnapshot(posts map[string]*models.Post, tagIndex *tags.Index) *Graph {
	edges := make([]Edge, 0, len(b.edgeMap))
	for _, e := range b.edgeMap {
		if len(e.SharedTags) < b.opts.MinSharedTags {
			continue
		}
		copyTags := append([]string(nil), e.SharedTags...)
		sort.Strings(copyTags)
		edge := Edge{From: e.From, To: e.To, SharedTags: copyTags, Weight: len(copyTags)}
		edges = append(edges, edge)
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Weight != edges[j].Weight {
			return edges[i].Weight > edges[j].Weight
		}
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})

	if b.opts.MaxEdges > 0 && len(edges) > b.opts.MaxEdges {
		edges = edges[:b.opts.MaxEdges]
	}

	postList := make([]*models.Post, 0, len(posts))
	for _, p := range posts {
		postList = append(postList, p)
	}

	return &Graph{
		Posts:    postList,
		Edges:    edges,
		TagIndex: tagIndex.Snapshot(),
	}
}

// Invalidate marks the graph as needing rebuild.
func (b *Builder) Invalidate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.graphReady = false
}
