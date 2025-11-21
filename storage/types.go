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
	Name         string    `yaml:"name"`
	Slug         string    `yaml:"slug"` // derived from filename; frontmatter value ignored on upload
	Tags         []string  `yaml:"tags"`
	Aliases      []string  `yaml:"aliases"`
	Lead         string    `yaml:"lead"`      // short summary (can substitute description)
	CreatedRaw   string    `yaml:"created"`   // raw created date (YYYY-MM-DD) from frontmatter
	Created      time.Time `yaml:"-"`         // parsed created date (strict date only)
	UpdatedRaw   string    `yaml:"updated"`   // raw timestamp string from frontmatter (flexible formats)
	Updated      time.Time `yaml:"-"`         // parsed canonical time (set during validation / parse)
	PublishedRaw string    `yaml:"published"` // raw published value (string/bool); parsed in validation
	Published    bool      `yaml:"-"`         // parsed boolean
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
		if a.Meta.Updated.After(b.Meta.Updated) {
			return -1 // a is newer, comes first
		} else if a.Meta.Updated.Before(b.Meta.Updated) {
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
