package storage

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/soockee/cybersocke.com/storage/graph"
	"github.com/soockee/cybersocke.com/storage/models"
	"github.com/soockee/cybersocke.com/storage/tags"
)

func buildPost(slug string, updated string, tgs []string) *models.Post {
	ts, _ := time.Parse("2006-01-02", updated)
	return &models.Post{Meta: models.PostMeta{Slug: slug, Name: models.DeriveDisplayName(slug), UpdatedRaw: updated, Updated: ts, Tags: tgs, Published: true}, Content: []byte("content")}
}

func seedStore() *GCSStore {
	s := &GCSStore{
		logger:       slog.Default(),
		postCache:    make(map[string]*models.Post),
		tagIndex:     tags.NewIndex(),
		graphBuilder: graph.NewBuilder(graph.Options{}),
	}
	posts := []*models.Post{
		buildPost("alpha.md", "2024-01-01", []string{"type/note", "theme/kubernetes", "source/book"}),
		buildPost("beta.md", "2024-02-01", []string{"type/note", "theme/kubernetes", "theme/cost-optimization", "source/article"}),
		buildPost("gamma.md", "2024-03-01", []string{"type/note", "theme/cloud-architecture", "source/book"}),
		buildPost("delta.md", "2024-04-01", []string{"type/note", "theme/kubernetes", "theme/cloud-architecture", "source/paper"}),
	}
	for _, p := range posts {
		s.postCache[p.Meta.Slug] = p
		s.tagIndex.Add(p.Meta.Slug, p.Meta.Tags)
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
	any, err := s.GetPostsByTags(context.Background(), []string{"theme/kubernetes", "theme/cloud-architecture"}, false)
	if err != nil {
		t.Fatalf("ANY query error: %v", err)
	}
	if len(any) != 4 {
		t.Fatalf("ANY query size=%d want 4", len(any))
	}
	all, err := s.GetPostsByTags(context.Background(), []string{"theme/kubernetes", "theme/cloud-architecture"}, true)
	if err != nil {
		t.Fatalf("ALL query error: %v", err)
	}
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
	if len(rel) != 3 {
		t.Fatalf("expected 3 related posts got %d", len(rel))
	}
}

func TestBuildGraph(t *testing.T) {
	s := seedStore()
	g, err := s.BuildGraph(context.Background(), graph.Options{MinSharedTags: 1})
	if err != nil {
		t.Fatalf("BuildGraph error: %v", err)
	}
	if len(g.Posts) != 4 {
		t.Fatalf("expected 4 posts got %d", len(g.Posts))
	}
	if len(g.Edges) == 0 {
		t.Fatalf("expected edges >0")
	}
	for _, e := range g.Edges {
		if e.Weight < 1 {
			t.Fatalf("edge %s-%s weight <1", e.From, e.To)
		}
	}
}

func TestIncrementalGraphUpdate(t *testing.T) {
	s := seedStore()
	initial, err := s.BuildGraph(context.Background(), graph.Options{MinSharedTags: 1})
	if err != nil {
		t.Fatalf("initial BuildGraph error: %v", err)
	}
	edgeCount := len(initial.Edges)
	newPost := buildPost("epsilon.md", "2024-05-01", []string{"type/note", "theme/kubernetes", "source/article"})
	s.mu.Lock()
	s.postCache[newPost.Meta.Slug] = newPost
	s.tagIndex.Add(newPost.Meta.Slug, newPost.Meta.Tags)
	s.graphBuilder.Invalidate()
	s.mu.Unlock()
	updated, err := s.BuildGraph(context.Background(), graph.Options{MinSharedTags: 1})
	if err != nil {
		t.Fatalf("updated BuildGraph error: %v", err)
	}
	if len(updated.Posts) != 5 {
		t.Fatalf("expected 5 posts, got %d", len(updated.Posts))
	}
	if len(updated.Edges) <= edgeCount {
		t.Fatalf("expected edge count increase > %d got %d", edgeCount, len(updated.Edges))
	}
}
