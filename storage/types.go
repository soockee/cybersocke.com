package storage

import (
	"context"
	"net/http"

	"github.com/soockee/cybersocke.com/storage/models"
)

// NOTE: Directly depend on models package; legacy re-exports removed.

// ContentStore defines pure persistence operations for dynamic post content only
// (no tag indexing, validation, graph logic, or static asset concerns). Implementations
// are responsible for raw CRUD and optional caching of posts. Static assets and
// about page content are now handled by a distinct AssetStore.
type ContentStore interface {
	// GetPost retrieves a single post by slug.
	GetPost(slug string, ctx context.Context) (*models.Post, error)
	// GetPosts returns all posts as a map keyed by slug.
	GetPosts(ctx context.Context) (map[string]*models.Post, error)
	// GetRaw returns the raw markdown bytes (frontmatter + body) for editing.
	GetRaw(slug string, ctx context.Context) ([]byte, error)
	// CreatePost persists a new post from raw markdown bytes.
	CreatePost(data []byte, originalFilename string, ctx context.Context) error
	// UpdatePost replaces the entire raw markdown content of an existing post.
	UpdatePost(slug string, data []byte, ctx context.Context) error
	// DeletePost removes a post completely.
	DeletePost(slug string, ctx context.Context) error
}

// AssetStore provides read-only access to static public assets and optional
// auxiliary content pages (e.g. About). Mutation operations are intentionally
// excluded; dynamic content belongs in a ContentStore implementation.
type AssetStore interface {
	// GetAssets returns an HTTP handler that serves static files.
	GetAssets() http.Handler
}

// SchemaStore defines operations for reading/writing post contract schema.
// Separated from ContentStore to allow independent schema management.
type SchemaStore interface {
	// GetSchema retrieves the current post contract schema (YAML).
	GetSchema(ctx context.Context) ([]byte, error)
	// PutSchema writes/updates the post contract schema.
	PutSchema(ctx context.Context, data []byte) error
}

// PostQueryStore defines tag/graph-aware read-only querying operations separated from
// the core ContentStore CRUD. Implementations may maintain in-memory indexes or graphs.
// This separation allows simpler testing of CRUD vs derived queries independently.
type PostQueryStore interface {
	// GetPostsByTags returns posts matching ANY or ALL provided tags.
	GetPostsByTags(ctx context.Context, tags []string, matchAll bool) ([]*models.Post, error)
	// GetRelatedPosts returns posts related to a slug ranked by shared tags.
	GetRelatedPosts(ctx context.Context, slug string, limit int) ([]*models.Post, error)
}

// Compile-time assertions (kept near interface declarations for clarity).
// Concrete types are expected to satisfy these; failure surfaces during build.
var (
	_ ContentStore   = (*GCSStore)(nil)
	_ PostQueryStore = (*GCSStore)(nil)
	_ AssetStore     = (*AssetsStore)(nil)
	_ SchemaStore    = (*GCSStore)(nil)
)
