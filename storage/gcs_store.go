package storage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/allegro/bigcache/v3"
	"github.com/soockee/cybersocke.com/parser/frontmatter"
	"github.com/soockee/cybersocke.com/session"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCSStore struct {
	bucketName string
	client     *storage.Client
	logger     *slog.Logger
	cache      *bigcache.BigCache
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

	cache, err := bigcache.New(ctx, bigcache.DefaultConfig(1*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("initializing cache: %w", err)
	}

	store := &GCSStore{
		logger:     logger,
		bucketName: bucketName,
		client:     client,
		cache:      cache,
	}

	store.logger = store.logger.With("component", "gcsStore", "bucket", bucketName, "auth_mode", "base64_service_account")

	if err := store.preloadCache(ctx); err != nil {
		return nil, fmt.Errorf("preloading cache: %w", err)
	}

	return store, nil
}

// GetPost retrieves a single post by its filename (slug including .md, without the posts/ prefix)
func (s *GCSStore) GetPost(slug string, ctx context.Context) (*Post, error) {
	// Expect slug to include .md per spec
	if !strings.HasSuffix(slug, ".md") {
		slug = slug + ".md"
	}
	var raw []byte
	if data, err := s.cache.Get(slug); err == nil {
		s.logger.Info("Cache hit for post", slog.String("slug", slug))
		raw = data
	} else {
		objName := "posts/" + slug
		var err2 error
		raw, err2 = s.readObject(ctx, objName)
		if err2 != nil {
			return nil, err2
		}
		s.cache.Set(slug, raw)
	}
	postPtr, err := parsePost(raw)
	if err != nil {
		return nil, err
	}
	postPtr.Meta.Slug = slug
	if strings.TrimSpace(postPtr.Meta.Name) == "" {
		postPtr.Meta.Name = DeriveDisplayName(slug)
	}
	return postPtr, nil
}

// GetPosts returns all posts as pointers parsed from cache or GCS
func (s *GCSStore) GetPosts(ctx context.Context) (map[string]*Post, error) {
	result := make(map[string]*Post)
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
		var raw []byte
		if data, err := s.cache.Get(filename); err == nil {
			s.logger.Info("Cache hit for post", slog.String("slug", filename))
			raw = data
		} else {
			raw, err = s.readObject(ctx, attrs.Name)
			if err != nil {
				return nil, err
			}
			s.cache.Set(filename, raw)
		}
		postPtr, err := parsePost(raw)
		if err != nil {
			return nil, err
		}
		postPtr.Meta.Slug = filename
		if strings.TrimSpace(postPtr.Meta.Name) == "" {
			postPtr.Meta.Name = DeriveDisplayName(filename)
		}
		result[filename] = postPtr
	}
	return result, nil
}

func (s *GCSStore) GetAbout() []byte {
	return []byte{}
}

func (s *GCSStore) GetAssets() http.Handler {
	return nil
}

func (s *GCSStore) CreatePost(content []byte, originalFilename string, ctx context.Context) error {
	// Require authenticated Firebase user (middleware should have injected token)
	firebaseTok, ok := ctx.Value(session.IdTokenKey).(*firebaseauth.Token)
	if !ok || firebaseTok == nil {
		return fmt.Errorf("unauthorized: firebase token missing")
	}

	// Derive slug from original filename (ignore any frontmatter slug)
	derivedSlug := sanitizeFilename(originalFilename)
	postMeta := PostMeta{}
	if _, err := frontmatter.Parse(strings.NewReader(string(content)), &postMeta); err != nil {
		return err
	}
	postMeta.Slug = derivedSlug
	if err := postMeta.Validate(); err != nil {
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
	return nil
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
		if err := s.cache.Set(filename, data); err != nil {
			return fmt.Errorf("caching content %s: %w", attrs.Name, err)
		}
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

func (s *GCSStore) readWithExtension(ctx context.Context, name, extension string) ([]byte, error) {
	data, err := s.readObject(ctx, name+extension)
	if err != nil {
		return nil, fmt.Errorf("reading object with extension %s: %w", extension, err)
	}
	return data, nil
}

// parsePost converts raw frontmatter+content bytes into a Post
func parsePost(raw []byte) (*Post, error) {
	var meta PostMeta
	body, err := frontmatter.Parse(strings.NewReader(string(raw)), &meta)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}
	// Derive Date from updated if not already set (Validate not called on read path)
	if meta.Date.IsZero() && strings.TrimSpace(meta.UpdatedRaw) != "" {
		if ts := parseTimestamp(meta.UpdatedRaw); !ts.IsZero() {
			meta.Date = ts
		}
	}
	return &Post{Meta: meta, Content: body}, nil
}

// sanitizeFilename converts an arbitrary filename to a lowercase kebab-case slug with .md extension.
func sanitizeFilename(name string) string {
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
