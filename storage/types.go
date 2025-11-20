package storage

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"
)

type Storage interface {
	GetPost(slug string, ctx context.Context) (*Post, error)
	GetPosts(ctx context.Context) (map[string]*Post, error)
	GetPostsByTags(ctx context.Context, tags []string, matchAll bool) ([]*Post, error)
	GetRelatedPosts(ctx context.Context, slug string, limit int) ([]*Post, error)
	GetAbout() []byte
	GetAssets() http.Handler

	CreatePost(data []byte, originalFilename string, ctx context.Context) error
}

type PostMeta struct {
	Name            string    `yaml:"name"`
	Slug            string    `yaml:"slug"` // derived from filename; frontmatter value ignored on upload
	Tags            []string  `yaml:"tags"`
	Aliases         []string  `yaml:"aliases"`
	Lead            string    `yaml:"lead"` // short summary (can substitute description)
	Visual          string    `yaml:"visual"`
	Created         any       `yaml:"created"`  // may be scalar string or map placeholder
	Modified        any       `yaml:"modified"` // may be scalar string or map placeholder
	TemplateType    string    `yaml:"template_type"`
	TemplateVersion string    `yaml:"template_version"`
	License         string    `yaml:"license"`
	UpdatedRaw      string    `yaml:"updated"`
	Date            time.Time `yaml:"date"` // legacy field; falls back to updated/created if absent
	Description     string    `yaml:"description"`
}

type Post struct {
	Meta    PostMeta
	Content []byte
}

func SortPostMap(posts map[string]*Post) []*Post {
	it := maps.Values(posts)
	s := []*Post{}
	for post := range it {
		s = append(s, post)
	}
	return SortPostsByDate(s)
}

func SortPostsByDate(posts []*Post) []*Post {
	// Sort in descending order (newest first)
	slices.SortFunc(posts, func(a, b *Post) int {
		if a.Meta.Date.After(b.Meta.Date) {
			return -1 // a is newer, comes first
		} else if a.Meta.Date.Before(b.Meta.Date) {
			return 1 // b is newer, comes first
		}
		return 0 // Dates are equal
	})
	return posts
}

// DeriveDisplayName builds a human-friendly Title Case name from a slug that includes `.md`.
// Example: "my-first-post.md" -> "My First Post".
// Hyphens become spaces; each segment capitalized (first rune upper, rest unchanged).
func DeriveDisplayName(slug string) string {
	if slug == "" {
		return ""
	}
	base := strings.TrimSuffix(slug, ".md")
	parts := strings.Split(base, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
