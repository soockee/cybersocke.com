package storage

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"log/slog"

	"cloud.google.com/go/storage"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/parser/frontmatter"
	"github.com/soockee/cybersocke.com/session"
	"github.com/soockee/cybersocke.com/storage/graph"
	"github.com/soockee/cybersocke.com/storage/models"
	"github.com/soockee/cybersocke.com/storage/tags"
	"github.com/soockee/cybersocke.com/storage/validation"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCSStore struct {
	logger     *slog.Logger
	bucketName string
	client     *storage.Client

	mu           sync.RWMutex
	tagIndex     *tags.Index
	postCache    map[string]*models.Post
	graphBuilder *graph.Builder
}

// NewGCSStore creates a GCS backed store using a base64 encoded service account key.
// Hetzner (non-GCP) deployment requires explicit JSON credentials instead of ADC / OIDC.
// The credentialsBase64 parameter MUST contain the base64 encoded JSON service account key.
func NewGCSStore(ctx context.Context, logger *slog.Logger, bucketName string, credentialsBase64 string) (*GCSStore, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name must be provided")
	}
	if credentialsBase64 == "" {
		return nil, fmt.Errorf("service account key (base64) must be provided")
	}

	if logger == nil {
		logger = slog.Default()
	}

	credJSON, err := base64.StdEncoding.DecodeString(credentialsBase64)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 credentials: %w", err)
	}

	client, err := storage.NewClient(
		ctx,
		option.WithCredentialsJSON(credJSON),
		option.WithUserAgent("cybersocke.com/storage-gcs"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating storage client with JSON key: %w", err)
	}

	store := &GCSStore{
		logger:       logger,
		bucketName:   bucketName,
		client:       client,
		tagIndex:     tags.NewIndex(),
		postCache:    make(map[string]*models.Post),
		graphBuilder: graph.NewBuilder(graph.Options{}),
	}

	store.logger = store.logger.With("component", "gcsStore", "bucket", bucketName, "auth_mode", "base64_service_account")

	// Load contract (remote-first) before preloading posts so validation succeeds.
	if _, src, err := config.LoadPostContractRemote(ctx, store); err == nil {
		store.logger.Info("post contract loaded", "source", src, "version", config.Contract.Version, "hash", config.Contract.Hash())
	} else {
		store.logger.Warn("post contract load failed", "err", err)
	}

	if err := store.preloadCache(ctx); err != nil {
		return nil, fmt.Errorf("preloading cache: %w", err)
	}
	tagSnapshot := store.tagIndex.Snapshot()
	store.logger.Info("gcs preload complete", slog.Int("posts_cached", len(store.postCache)), slog.Int("distinct_tags", len(tagSnapshot)))

	return store, nil
}

// NewGCSStoreADC creates a GCS backed store using Application Default Credentials (ADC).
// This is intended for environments where a workload identity / federated identity or
// user credentials are already provisioned (e.g., GitHub Actions Workload Identity Federation,
// Cloud Run, GCE, or a developer machine with `gcloud auth application-default login`).
// Unlike NewGCSStore, this variant does NOT accept a service account key and instead
// relies on ambient credentials resolution.
func NewGCSStoreADC(ctx context.Context, logger *slog.Logger, bucketName string) (*GCSStore, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name must be provided")
	}
	if logger == nil {
		logger = slog.Default()
	}
	client, err := storage.NewClient(
		ctx,
		option.WithUserAgent("cybersocke.com/storage-gcs-adc"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating storage client with ADC: %w", err)
	}
	store := &GCSStore{
		logger:       logger.With("component", "gcsStore", "bucket", bucketName, "auth_mode", "adc"),
		bucketName:   bucketName,
		client:       client,
		tagIndex:     tags.NewIndex(),
		postCache:    make(map[string]*models.Post),
		graphBuilder: graph.NewBuilder(graph.Options{}),
	}

	// Load contract (remote-first) before preloading posts so validation succeeds.
	if _, src, err := config.LoadPostContractRemote(ctx, store); err == nil {
		store.logger.Info("post contract loaded", "source", src, "version", config.Contract.Version, "hash", config.Contract.Hash())
	} else {
		store.logger.Warn("post contract load failed", "err", err)
	}

	if err := store.preloadCache(ctx); err != nil {
		return nil, fmt.Errorf("preloading cache: %w", err)
	}
	tagSnapshot := store.tagIndex.Snapshot()
	store.logger.Info("gcs preload complete", slog.Int("posts_cached", len(store.postCache)), slog.Int("distinct_tags", len(tagSnapshot)))
	return store, nil
}

// GetPost retrieves a single post by its filename (slug including .md, without the posts/ prefix)
func (s *GCSStore) GetPost(slug string, ctx context.Context) (*models.Post, error) {
	// Expect slug to include .md per spec
	if !strings.HasSuffix(slug, ".md") {
		slug = slug + ".md"
	}
	// Fast path: parsed cache
	s.mu.RLock()
	if p, ok := s.postCache[slug]; ok {
		s.mu.RUnlock()
		return p, nil
	}
	s.mu.RUnlock()

	objName := "posts/" + slug
	raw, err2 := s.readObject(ctx, objName)
	if err2 != nil {
		return nil, err2
	}
	postPtr, err := parsePost(raw)
	if err != nil {
		return nil, err
	}
	postPtr.Meta.Slug = slug
	if strings.TrimSpace(postPtr.Meta.Name) == "" {
		postPtr.Meta.Name = models.DeriveDisplayName(slug)
	}
	s.mu.Lock()
	s.postCache[slug] = postPtr
	s.mu.Unlock()
	return postPtr, nil
}

// GetPosts returns all posts as pointers parsed from cache or GCS
func (s *GCSStore) GetPosts(ctx context.Context) (map[string]*models.Post, error) {
	result := make(map[string]*models.Post)
	q := &storage.Query{Prefix: "posts/"}
	it := s.client.Bucket(s.bucketName).Objects(ctx, q)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing objects: %w", err)
		}
		// Skip non markdown files
		if !strings.HasSuffix(attrs.Name, ".md") {
			continue
		}
		// Derive slug (filename with extension) by trimming prefix
		filename := strings.TrimPrefix(attrs.Name, "posts/")
		postPtr, err := s.GetPost(filename, ctx)
		if err != nil {
			return nil, err
		}
		result[filename] = postPtr
	}
	return result, nil
}

func (s *GCSStore) GetAssets() http.Handler {
	return nil
}

func (s *GCSStore) CreatePost(content []byte, originalFilename string, ctx context.Context) error {
	// Require authenticated Firebase user (middleware should have injected token)
	firebaseTok, _ := ctx.Value(session.IdTokenKey).(*firebaseauth.Token)
	if firebaseTok == nil { // presence implies prior successful verification
		return fmt.Errorf("unauthorized: firebase token missing")
	}


	derivedSlug := SanitizeFilename(originalFilename)
	postMeta := PostMeta{}
	// Parse frontmatter to populate metadata and obtain the markdown body without frontmatter.
	body, err := frontmatter.Parse(strings.NewReader(string(content)), &postMeta)
	if err != nil {
		return err
	}
	// Normalize parsed dates & published flag (mirror parsePost logic for consistency with preload).
	if postMeta.Created.IsZero() && strings.TrimSpace(postMeta.CreatedRaw) != "" {
		postMeta.Created = parseDate(postMeta.CreatedRaw)
	}
	if postMeta.Updated.IsZero() && strings.TrimSpace(postMeta.UpdatedRaw) != "" {
		postMeta.Updated = parseTimestamp(postMeta.UpdatedRaw)
	}
	rawPub := strings.ToLower(strings.TrimSpace(postMeta.PublishedRaw))
	switch rawPub {
	case "", "false":
		postMeta.Published = false
	case "true":
		postMeta.Published = true
	default:
		postMeta.Published = false // error surfaced in ValidateMeta
	}
	postMeta.Slug = derivedSlug
	if err := validation.ValidateMeta(&postMeta, config.Contract, originalFilename); err != nil {
		return err
	}
	if err := validation.ValidateTags(&postMeta, config.Contract); err != nil {
		return err
	}

	obj := s.client.Bucket(s.bucketName).Object("posts/" + postMeta.Slug).NewWriter(ctx)
	obj.ContentType = "text/markdown"
	obj.Metadata = map[string]string{"uploaded_by": firebaseTok.UID}

	if _, err := obj.Write(content); err != nil {
		obj.Close()
		return fmt.Errorf("write object: %w", err)
	}
	if err := obj.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}


	post := models.Post{Meta: postMeta, Content: body}

	s.mu.Lock()
	s.postCache[postMeta.Slug] = &post
	s.tagIndex.Add(postMeta.Slug, postMeta.Tags)
	// Invalidate graph - will be rebuilt on next request
	s.graphBuilder.Invalidate()
	s.mu.Unlock()
	s.logger.Info("post created", slog.String("slug", postMeta.Slug), slog.Int("tag_count", len(postMeta.Tags)))
	return nil
}

// UpdatePost overwrites an existing post's raw markdown (frontmatter + body) and updates caches.
func (s *GCSStore) UpdatePost(slug string, data []byte, ctx context.Context) error {
	firebaseTok, _ := ctx.Value(session.IdTokenKey).(*firebaseauth.Token)
	if firebaseTok == nil {
		return fmt.Errorf("unauthorized: firebase token missing")
	}
	actor := firebaseTok.UID
	if actor == "" {
		actor = "firebase_user"
	}
	return s.updatePost(slug, data, ctx, actor)
}

// UpdatePostSystem overwrites a post using elevated system credentials (no Firebase token required).
// It is intended for trusted automation such as contract migrations running with bucket-level IAM access.
func (s *GCSStore) UpdatePostSystem(slug string, data []byte, ctx context.Context) error {
	return s.updatePost(slug, data, ctx, "contract_migrate")
}

func (s *GCSStore) updatePost(slug string, data []byte, ctx context.Context, actor string) error {
	if !strings.HasSuffix(slug, ".md") {
		slug = slug + ".md"
	}
	postPtr, err := parsePost(data)
	if err != nil {
		return err
	}
	postPtr.Meta.Slug = slug
	if strings.TrimSpace(postPtr.Meta.Name) == "" {
		postPtr.Meta.Name = models.DeriveDisplayName(slug)
	}
	if err := validation.ValidateMeta(&postPtr.Meta, config.Contract, ""); err != nil {
		return err
	}
	if err := validation.ValidateTags(&postPtr.Meta, config.Contract); err != nil {
		return err
	}
	obj := s.client.Bucket(s.bucketName).Object("posts/" + slug).NewWriter(ctx)
	obj.ContentType = "text/markdown"
	if actor != "" {
		obj.Metadata = map[string]string{"updated_by": actor}
	}
	if _, err := obj.Write(data); err != nil {
		obj.Close()
		return fmt.Errorf("write object: %w", err)
	}
	if err := obj.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}
	s.mu.Lock()
	oldPost, had := s.postCache[slug]
	if had {
		s.tagIndex.Remove(slug, oldPost.Meta.Tags)
	}
	s.postCache[slug] = postPtr
	s.tagIndex.Add(slug, postPtr.Meta.Tags)
	s.graphBuilder.Invalidate()
	s.mu.Unlock()
	s.logger.Info("post updated", slog.String("slug", slug), slog.Int("tag_count", len(postPtr.Meta.Tags)), slog.String("actor", actor))
	return nil
}

// DeletePost removes a post permanently from GCS and caches.
func (s *GCSStore) DeletePost(slug string, ctx context.Context) error {
	firebaseTok, _ := ctx.Value(session.IdTokenKey).(*firebaseauth.Token)
	if firebaseTok == nil {
		return fmt.Errorf("unauthorized: firebase token missing")
	}
	if !strings.HasSuffix(slug, ".md") {
		slug = slug + ".md"
	}
	s.mu.Lock()
	postPtr, ok := s.postCache[slug]
	s.mu.Unlock()
	// Delete object first
	obj := s.client.Bucket(s.bucketName).Object("posts/" + slug)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	if ok {
		s.mu.Lock()
		s.tagIndex.Remove(slug, postPtr.Meta.Tags)
		delete(s.postCache, slug)
		// Graph dirty
		s.graphBuilder.Invalidate()
		s.mu.Unlock()
	}
	s.logger.Info("post deleted", slog.String("slug", slug), slog.String("by", firebaseTok.UID))
	return nil
}

// GetRaw returns raw frontmatter + markdown for editing.
func (s *GCSStore) GetRaw(slug string, ctx context.Context) ([]byte, error) {
	if !strings.HasSuffix(slug, ".md") {
		slug = slug + ".md"
	}
	return s.readObject(ctx, "posts/"+slug)
}

// Federated impersonation functions removed.

// preloadCache lists all objects in the bucket and stores their content in cache
func (s *GCSStore) preloadCache(ctx context.Context) error {
	q := &storage.Query{Prefix: "posts/"}
	it := s.client.Bucket(s.bucketName).Objects(ctx, q)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("listing objects: %w", err)
		}
		if !strings.HasSuffix(attrs.Name, ".md") {
			continue
		}
		filename := strings.TrimPrefix(attrs.Name, "posts/")
		obj := s.client.Bucket(s.bucketName).Object(attrs.Name)
		rc, err := obj.NewReader(ctx)
		if err != nil {
			return fmt.Errorf("reading object %s: %w", attrs.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("reading data %s: %w", attrs.Name, err)
		}
		// parse and index directly; no separate raw cache
		postPtr, err := parsePost(data)
		if err != nil {
			return fmt.Errorf("parsing post %s: %w", attrs.Name, err)
		}
		postPtr.Meta.Slug = filename
		if strings.TrimSpace(postPtr.Meta.Name) == "" {
			postPtr.Meta.Name = models.DeriveDisplayName(filename)
		}
		// validate metadata
		if err := validation.ValidateMeta(&postPtr.Meta, config.Contract, ""); err != nil {
			s.logger.Warn("preload validation failed", "slug", postPtr.Meta.Slug, "error", err)
			continue // skip inserting into cache and tag index
		}
		if err := validation.ValidateTags(&postPtr.Meta, config.Contract); err != nil {
			s.logger.Warn("preload tag validation failed", "slug", postPtr.Meta.Slug, "error", err)
			continue
		}
		// index tags
		s.mu.Lock()
		s.postCache[filename] = postPtr
		s.tagIndex.Add(filename, postPtr.Meta.Tags)
		s.mu.Unlock()
	}
	return nil
}

// readObject reads raw bytes from GCS
func (s *GCSStore) readObject(ctx context.Context, name string) ([]byte, error) {
	rc, err := s.client.Bucket(s.bucketName).Object(name).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("opening object %s: %w", name, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("reading data %s: %w", name, err)
	}
	return data, nil
}

// GetSchema reads the canonical schema file from GCS (schema/post_contract.yaml).
// Returns storage.ErrObjectNotExist wrapped error if schema does not exist.
func (s *GCSStore) GetSchema(ctx context.Context) ([]byte, error) {
	return s.readObject(ctx, "schema/post_contract.yaml")
}

// PutSchema writes schema data to the canonical location (schema/post_contract.yaml).
// Overwrites any existing schema; use object versioning for history.
func (s *GCSStore) PutSchema(ctx context.Context, data []byte) error {
	obj := s.client.Bucket(s.bucketName).Object("schema/post_contract.yaml").NewWriter(ctx)
	obj.ContentType = "application/x-yaml"
	obj.Metadata = map[string]string{"purpose": "post_contract_canonical"}
	if _, err := obj.Write(data); err != nil {
		obj.Close()
		return fmt.Errorf("write schema object: %w", err)
	}
	if err := obj.Close(); err != nil {
		return fmt.Errorf("close schema writer: %w", err)
	}
	s.logger.Info("schema persisted", slog.String("path", "schema/post_contract.yaml"), slog.Int("bytes", len(data)))
	return nil
}

// parsePost converts raw frontmatter+content bytes into a Post
func parsePost(raw []byte) (*models.Post, error) {
	var meta models.PostMeta
	body, err := frontmatter.Parse(strings.NewReader(string(raw)), &meta)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}
	// Parse strict created date if provided.
	if meta.Created.IsZero() && strings.TrimSpace(meta.CreatedRaw) != "" {
		meta.Created = parseDate(meta.CreatedRaw)
	}
	// Parse flexible updated timestamp into canonical time if provided.
	if meta.Updated.IsZero() && strings.TrimSpace(meta.UpdatedRaw) != "" {
		meta.Updated = parseTimestamp(meta.UpdatedRaw)
	}
	// Parse published strict boolean value.
	rawPub := strings.ToLower(strings.TrimSpace(meta.PublishedRaw))
	switch rawPub {
	case "", "false":
		meta.Published = false
	case "true":
		meta.Published = true
	default:
		meta.Published = false // ValidateMeta will surface error
	}
	return &models.Post{Meta: meta, Content: body}, nil
}

// SanitizeFilename converts an arbitrary filename to a lowercase kebab-case slug with .md extension.
// Exported to allow handlers to provide immediate feedback (e.g., predicted slug) after upload.
func SanitizeFilename(name string) string {
	name = strings.ToLower(name)
	// Remove any path components
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}
	base := name
	// Remove extension for transformation
	withoutExt := strings.TrimSuffix(base, ".md")
	replacer := strings.NewReplacer(" ", "-", "_", "-")
	withoutExt = replacer.Replace(withoutExt)
	// Remove invalid characters
	valid := make([]rune, 0, len(withoutExt))
	for _, r := range withoutExt {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			valid = append(valid, r)
		}
	}
	clean := string(valid)
	// Collapse multiple hyphens
	clean = regexp.MustCompile(`-+`).ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "post"
	}
	return clean + ".md"
}

// GetPostsByTags returns posts matching ANY or ALL of the provided tags.
// If tags slice is empty an error is returned.
func (s *GCSStore) GetPostsByTags(ctx context.Context, tagList []string, matchAll bool) ([]*models.Post, error) {
	if len(tagList) == 0 {
		return nil, errors.New("no tags provided")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return tags.GetPostsByTags(s.tagIndex, s.postCache, tagList, matchAll)
}

// GetRelatedPosts returns posts that share at least one tag with the given slug, ranked by
// number of shared tags desc, then by date desc, then by slug asc. limit <=0 means no cap.
func (s *GCSStore) GetRelatedPosts(ctx context.Context, slug string, limit int) ([]*models.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return tags.GetRelatedPosts(s.tagIndex, s.postCache, slug, limit)
}

// BuildGraph constructs a graph.Graph (new API) of posts connected by shared tags.
func (s *GCSStore) BuildGraph(ctx context.Context, opts graph.Options) (*graph.Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.graphBuilder.Build(s.postCache, s.tagIndex, opts)
	return g, nil
}

// RebuildTagIndex rebuilds tag index from the current parsed post cache.
func (s *GCSStore) RebuildTagIndex(ctx context.Context) error { // ctx reserved for future parallelization
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tagIndex.Rebuild(s.postCache)
	return nil
}
