package storage

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// helper to build a post
func buildPost(slug string, updated string, tags []string) *Post {
	ts, _ := time.Parse("2006-01-02", updated)
	return &Post{Meta: PostMeta{Slug: slug, Name: DeriveDisplayName(slug), UpdatedRaw: updated, Updated: ts, Tags: tags, Published: true}, Content: []byte("content")}
}

func seedStore() *GCSStore {
	s := &GCSStore{tagIndex: make(map[string]map[string]struct{}), postCache: make(map[string]*Post), logger: slog.Default()}
	posts := []*Post{
		buildPost("alpha.md", "2024-01-01", []string{"type/note", "theme/kubernetes", "source/book"}),
		buildPost("beta.md", "2024-02-01", []string{"type/note", "theme/kubernetes", "theme/cost-optimization", "source/article"}),
		buildPost("gamma.md", "2024-03-01", []string{"type/note", "theme/cloud-architecture", "source/book"}),
		buildPost("delta.md", "2024-04-01", []string{"type/note", "theme/kubernetes", "theme/cloud-architecture", "source/paper"}),
	}
	for _, p := range posts {
		s.postCache[p.Meta.Slug] = p
		indexTagsLocked(s.tagIndex, p.Meta.Slug, p.Meta.Tags)
	}
	return s
}

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"My File.md":             "my-file.md",
		"My File":                "my-file.md",
		"Some_Path/Deep/File.md": "file.md",
		"UPPER_case Name":        "upper-case-name.md",
		"__spaces  multiple__":   "spaces-multiple.md",
	}
	for in, expect := range cases {
		got := SanitizeFilename(in)
		if got != expect {
			t.Fatalf("sanitizeFilename(%q) = %q; want %q", in, got, expect)
		}
	}
}

func TestGetPostsByTagsAnyAll(t *testing.T) {
	s := seedStore()
	// ANY query
	any, err := s.GetPostsByTags(context.Background(), []string{"theme/kubernetes", "theme/cloud-architecture"}, false)
	if err != nil {
		t.Fatalf("ANY query error: %v", err)
	}
	// kubernetes: alpha, beta, delta; cloud-architecture: gamma, delta => union = alpha, beta, gamma, delta (4)
	if len(any) != 4 {
		t.Fatalf("ANY query size=%d want 4", len(any))
	}
	// ALL query
	all, err := s.GetPostsByTags(context.Background(), []string{"theme/kubernetes", "theme/cloud-architecture"}, true)
	if err != nil {
		t.Fatalf("ALL query error: %v", err)
	}
	// Only delta has both
	if len(all) != 1 || all[0].Meta.Slug != "delta.md" {
		t.Fatalf("ALL query unexpected result size=%d first=%v", len(all), func() string {
			if len(all) > 0 {
				return all[0].Meta.Slug
			}
			return ""
		}())
	}
}

func TestGetRelatedPostsRanking(t *testing.T) {
	s := seedStore()
	rel, err := s.GetRelatedPosts(context.Background(), "delta.md", 0)
	if err != nil {
		t.Fatalf("GetRelatedPosts error: %v", err)
	}
	// delta shares tags:
	// with alpha: theme/kubernetes
	// with beta: theme/kubernetes
	// with gamma: theme/cloud-architecture
	if len(rel) != 3 {
		t.Fatalf("expected 3 related posts got %d", len(rel))
	}
}

func TestBuildTagGraph(t *testing.T) {
	s := seedStore()
	g, err := s.BuildTagGraph(context.Background(), TagGraphOptions{MinSharedTags: 1})
	if err != nil {
		t.Fatalf("BuildTagGraph error: %v", err)
	}
	if len(g.Posts) != 4 {
		t.Fatalf("expected 4 posts got %d", len(g.Posts))
	}
	if len(g.Edges) == 0 {
		t.Fatalf("expected edges >0")
	}
	// Ensure weights >=1
	for _, e := range g.Edges {
		if e.Weight < 1 {
			t.Fatalf("edge %s-%s weight <1", e.From, e.To)
		}
	}
}

func TestIncrementalGraphUpdate(t *testing.T) {
	s := seedStore()
	opts := TagGraphOptions{MinSharedTags: 1}
	initial, err := s.BuildTagGraph(context.Background(), opts)
	if err != nil {
		t.Fatalf("initial build error: %v", err)
	}
	edgeCount := len(initial.Edges)
	// Add new post with shared tag
	newPost := buildPost("epsilon.md", "2024-05-01", []string{"type/note", "theme/kubernetes", "source/article"})
	s.mu.Lock()
	s.postCache[newPost.Meta.Slug] = newPost
	indexTagsLocked(s.tagIndex, newPost.Meta.Slug, newPost.Meta.Tags)
	incrementalAddPostToGraphLocked(s.edgeMap, newPost, s.tagIndex, opts.MinSharedTags, opts.IncludeTags)
	s.mu.Unlock()
	// Snapshot again (should reuse existing edgeMap and include new edges without full rebuild)
	updated, err := s.BuildTagGraph(context.Background(), opts)
	if err != nil {
		t.Fatalf("updated build error: %v", err)
	}
	if len(updated.Posts) != 5 {
		t.Fatalf("expected 5 posts, got %d", len(updated.Posts))
	}
	if len(updated.Edges) <= edgeCount {
		t.Fatalf("expected edge count increase > %d got %d", edgeCount, len(updated.Edges))
	}
}
