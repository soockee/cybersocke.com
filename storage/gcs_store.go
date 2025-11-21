package storage

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	firebaseauth "firebase.google.com/go/v4/auth"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/soockee/cybersocke.com/parser/frontmatter"
	"github.com/soockee/cybersocke.com/session"
)

type GCSStore struct {
	logger     *slog.Logger
	bucketName string
	client     *storage.Client

	mu        sync.RWMutex
	tagIndex  map[string]map[string]struct{}
	postCache map[string]*Post

	edgeMap      map[string]*GraphEdge
	graphOptions TagGraphOptions
	graphReady   bool
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
		logger:     logger,
		bucketName: bucketName,
		client:     client,
		tagIndex:   make(map[string]map[string]struct{}),
		postCache:  make(map[string]*Post),
		edgeMap:    make(map[string]*GraphEdge),
	}

	store.logger = store.logger.With("component", "gcsStore", "bucket", bucketName, "auth_mode", "base64_service_account")

	if err := store.preloadCache(ctx); err != nil {
		return nil, fmt.Errorf("preloading cache: %w", err)
	}
	store.logger.Info("gcs preload complete", slog.Int("posts_cached", len(store.postCache)), slog.Int("distinct_tags", len(store.tagIndex)))

	return store, nil
}

// GetPost retrieves a single post by its filename (slug including .md, without the posts/ prefix)
func (s *GCSStore) GetPost(slug string, ctx context.Context) (*Post, error) {
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
		postPtr.Meta.Name = DeriveDisplayName(slug)
	}
	s.mu.Lock()
	s.postCache[slug] = postPtr
	s.mu.Unlock()
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
		postPtr, err := s.GetPost(filename, ctx)
		if err != nil {
			return nil, err
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
	firebaseTok, _ := ctx.Value(session.IdTokenKey).(*firebaseauth.Token)
	if firebaseTok == nil { // presence implies prior successful verification
		return fmt.Errorf("unauthorized: firebase token missing")
	}

	// Derive slug from original filename (ignore any frontmatter slug)
	derivedSlug := SanitizeFilename(originalFilename)
	postMeta := PostMeta{}
	if _, err := frontmatter.Parse(strings.NewReader(string(content)), &postMeta); err != nil {
		return err
	}
	postMeta.Slug = derivedSlug
	if err := postMeta.Validate(); err != nil {
		return err
	}
	if err := ValidateTags(&postMeta); err != nil {
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

	// Update in-memory caches so new post is immediately queryable
	post := Post{Meta: postMeta, Content: content}
	s.mu.Lock()
	s.postCache[postMeta.Slug] = &post
	indexTagsLocked(s.tagIndex, postMeta.Slug, postMeta.Tags)
	// Incrementally update graph if already built with current options
	if s.graphReady {
		incrementalAddPostToGraphLocked(s.edgeMap, &post, s.tagIndex, s.graphOptions.MinSharedTags, s.graphOptions.IncludeTags)
	}
	s.mu.Unlock()
	s.logger.Info("post created", slog.String("slug", postMeta.Slug), slog.Int("tag_count", len(postMeta.Tags)))
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
		// parse and index directly; no separate raw cache
		postPtr, err := parsePost(data)
		if err != nil {
			return fmt.Errorf("parsing post %s: %w", attrs.Name, err)
		}
		postPtr.Meta.Slug = filename
		if strings.TrimSpace(postPtr.Meta.Name) == "" {
			postPtr.Meta.Name = DeriveDisplayName(filename)
		}
		// index tags
		s.mu.Lock()
		s.postCache[filename] = postPtr
		indexTagsLocked(s.tagIndex, filename, postPtr.Meta.Tags)
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

// readWithExtension removed (unused); callers should compose name+extension directly with readObject.

// parsePost converts raw frontmatter+content bytes into a Post
func parsePost(raw []byte) (*Post, error) {
	var meta PostMeta
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
	// Parse published flexible boolean (string or bool) into canonical bool.
	rawPub := strings.ToLower(strings.TrimSpace(meta.PublishedRaw))
	switch rawPub {
	case "", "false", "no", "0", "off":
		meta.Published = false
	case "true", "yes", "1", "on":
		meta.Published = true
	default:
		// treat invalid as false; do not error during passive parsing
		meta.Published = false
	}
	return &Post{Meta: meta, Content: body}, nil
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
func (s *GCSStore) GetPostsByTags(ctx context.Context, tags []string, matchAll bool) ([]*Post, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags provided")
	}
	// Normalize and deduplicate input tags
	uniq := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		uniq = append(uniq, t)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	resultSlugs := map[string]struct{}{}
	if matchAll {
		// Initialize with first tag set
		if len(uniq) == 0 {
			return []*Post{}, nil
		}
		firstSet, ok := s.tagIndex[uniq[0]]
		if !ok {
			return []*Post{}, nil
		}
		for slug := range firstSet {
			resultSlugs[slug] = struct{}{}
		}
		for _, t := range uniq[1:] {
			set, ok := s.tagIndex[t]
			if !ok {
				// Intersection with empty set -> empty result
				return []*Post{}, nil
			}
			for slug := range resultSlugs {
				if _, present := set[slug]; !present {
					delete(resultSlugs, slug)
				}
			}
		}
	} else {
		for _, t := range uniq {
			set, ok := s.tagIndex[t]
			if !ok {
				continue
			}
			for slug := range set {
				resultSlugs[slug] = struct{}{}
			}
		}
	}
	posts := make([]*Post, 0, len(resultSlugs))
	for slug := range resultSlugs {
		if p, ok := s.postCache[slug]; ok {
			posts = append(posts, p)
		} else {
			// Fallback to GetPost (outside lock) – but we still hold RLock so skip network, rely on existing cache only
		}
	}
	// Sort by date desc (newest first) then slug asc for stability
	sort.Slice(posts, func(i, j int) bool {
		if posts[i].Meta.Updated.Equal(posts[j].Meta.Updated) {
			return posts[i].Meta.Slug < posts[j].Meta.Slug
		}
		return posts[i].Meta.Updated.After(posts[j].Meta.Updated)
	})
	return posts, nil
}

// GetRelatedPosts returns posts that share at least one tag with the given slug, ranked by
// number of shared tags desc, then by date desc, then by slug asc. limit <=0 means no cap.
func (s *GCSStore) GetRelatedPosts(ctx context.Context, slug string, limit int) ([]*Post, error) {
	post, err := s.GetPost(slug, ctx)
	if err != nil {
		return nil, err
	}
	// Build frequency map
	s.mu.RLock()
	sharedCounts := map[string]int{}
	for _, tag := range post.Meta.Tags {
		set, ok := s.tagIndex[tag]
		if !ok {
			continue
		}
		for other := range set {
			if other == post.Meta.Slug {
				continue
			}
			sharedCounts[other]++
		}
	}
	// Collect posts
	related := make([]*Post, 0, len(sharedCounts))
	for otherSlug, count := range sharedCounts {
		p, ok := s.postCache[otherSlug]
		if !ok {
			continue
		}
		// attach count via temporary struct? We'll sort using map lookup
		pCopy := p
		_ = count
		related = append(related, pCopy)
	}
	s.mu.RUnlock()
	// Sort using ranking criteria
	sort.Slice(related, func(i, j int) bool {
		ci := sharedCounts[related[i].Meta.Slug]
		cj := sharedCounts[related[j].Meta.Slug]
		if ci != cj {
			return ci > cj // more shared tags first
		}
		if !related[i].Meta.Updated.Equal(related[j].Meta.Updated) {
			return related[i].Meta.Updated.After(related[j].Meta.Updated)
		}
		return related[i].Meta.Slug < related[j].Meta.Slug
	})
	if limit > 0 && len(related) > limit {
		related = related[:limit]
	}
	return related, nil
}

// BuildTagGraph constructs a graph of posts connected by shared tags.
func (s *GCSStore) BuildTagGraph(ctx context.Context, opts TagGraphOptions) (*TagGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if opts.MinSharedTags < 1 {
		opts.MinSharedTags = 1
	}
	// If graph already built with identical options, return cached projection
	if s.graphReady && s.graphOptions.MinSharedTags == opts.MinSharedTags && slicesEqual(s.graphOptions.IncludeTags, opts.IncludeTags) && s.graphOptions.MaxEdges == opts.MaxEdges {
		return buildGraphSnapshotFromEdgeMap(s.postCache, s.edgeMap, s.tagIndex, opts), nil
	}
	// Rebuild (first time or different options)
	s.edgeMap = make(map[string]*GraphEdge)
	filter := map[string]struct{}{}
	for _, t := range opts.IncludeTags {
		filter[strings.TrimSpace(t)] = struct{}{}
	}
	for tag, set := range s.tagIndex {
		if len(filter) > 0 {
			if _, ok := filter[tag]; !ok {
				continue
			}
		}
		slugs := make([]string, 0, len(set))
		for slug := range set {
			slugs = append(slugs, slug)
		}
		for i := 0; i < len(slugs); i++ {
			for j := i + 1; j < len(slugs); j++ {
				a, b := slugs[i], slugs[j]
				if a > b {
					a, b = b, a
				}
				key := a + "|" + b
				edge, exists := s.edgeMap[key]
				if !exists {
					edge = &GraphEdge{From: a, To: b}
					s.edgeMap[key] = edge
				}
				edge.SharedTags = append(edge.SharedTags, tag)
			}
		}
	}
	s.graphOptions = opts
	s.graphReady = true
	return buildGraphSnapshotFromEdgeMap(s.postCache, s.edgeMap, s.tagIndex, opts), nil
}

// RebuildTagIndex rebuilds tag index from the current parsed post cache.
func (s *GCSStore) RebuildTagIndex(ctx context.Context) error { // ctx reserved for future parallelization
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tagIndex = make(map[string]map[string]struct{})
	for slug, p := range s.postCache {
		indexTagsLocked(s.tagIndex, slug, p.Meta.Tags)
	}
	return nil
}

// ValidateTags enforces basic tag architecture cardinalities & family prefixes.
func ValidateTags(meta *PostMeta) error {
	// Allowed families restricted to a minimal curated set.
	// Only these families are permitted: type, role, structure, source, theme, target.
	families := map[string]struct{}{"type": {}, "role": {}, "structure": {}, "source": {}, "theme": {}, "target": {}}
	counts := map[string]int{}
	unique := map[string]struct{}{}
	filtered := make([]string, 0, len(meta.Tags))
	for _, raw := range meta.Tags {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		if _, dup := unique[t]; dup {
			continue
		}
		unique[t] = struct{}{}
		parts := strings.SplitN(t, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("tag %q must be family/value", t)
		}
		if _, ok := families[parts[0]]; !ok {
			return fmt.Errorf("unknown tag family: %s", parts[0])
		}
		counts[parts[0]]++
		filtered = append(filtered, t)
	}
	// Cardinality rules (subset MVP)
	if counts["source"] > 1 {
		return errors.New("multiple source/* tags not allowed")
	}
	if counts["structure"] > 1 {
		return errors.New("multiple structure/* tags not allowed")
	}
	if c := counts["type"]; c < 1 || c > 2 {
		return fmt.Errorf("type/* tags must be 1-2 (got %d)", c)
	}
	// role family optional; at most 3 distinct roles to avoid overclassification
	if c := counts["role"]; c > 3 {
		return fmt.Errorf("role/* tags must be 0-3 (got %d)", c)
	}
	if c := counts["theme"]; c < 1 || c > 5 {
		return fmt.Errorf("theme/* tags must be 1-5 (got %d)", c)
	}
	meta.Tags = filtered // normalized
	return nil
}

// Helper: index tags into tagIndex (must hold write lock).
func indexTagsLocked(idx map[string]map[string]struct{}, slug string, tags []string) {
	for _, t := range tags {
		if strings.TrimSpace(t) == "" {
			continue
		}
		set, ok := idx[t]
		if !ok {
			set = make(map[string]struct{})
			idx[t] = set
		}
		set[slug] = struct{}{}
	}
}

// snapshotTagIndex creates a copy for safe external use.
func snapshotTagIndex(src map[string]map[string]struct{}) map[string][]string {
	out := make(map[string][]string, len(src))
	for tag, set := range src {
		slugs := make([]string, 0, len(set))
		for slug := range set {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
		out[tag] = slugs
	}
	return out
}

// incrementalAddPostToGraphLocked updates edgeMap for a newly added post.
// Caller must hold write lock.
func incrementalAddPostToGraphLocked(edgeMap map[string]*GraphEdge, post *Post, tagIndex map[string]map[string]struct{}, minShared int, include []string) {
	if minShared < 1 {
		minShared = 1
	}
	filter := map[string]struct{}{}
	for _, t := range include {
		filter[strings.TrimSpace(t)] = struct{}{}
	}
	for _, tag := range post.Meta.Tags {
		if len(filter) > 0 {
			if _, ok := filter[tag]; !ok {
				continue
			}
		}
		set := tagIndex[tag]
		for other := range set {
			if other == post.Meta.Slug {
				continue
			}
			a, b := post.Meta.Slug, other
			if a > b {
				a, b = b, a
			}
			key := a + "|" + b
			edge, exists := edgeMap[key]
			if !exists {
				edge = &GraphEdge{From: a, To: b}
				edgeMap[key] = edge
			}
			edge.SharedTags = append(edge.SharedTags, tag)
		}
	}
	// We defer weight assignment & filtering to snapshot function to keep incremental path minimal.
}

// buildGraphSnapshotFromEdgeMap converts edgeMap into TagGraph honoring options.
func buildGraphSnapshotFromEdgeMap(postCache map[string]*Post, edgeMap map[string]*GraphEdge, tagIndex map[string]map[string]struct{}, opts TagGraphOptions) *TagGraph {
	edges := make([]GraphEdge, 0, len(edgeMap))
	for _, e := range edgeMap {
		if len(e.SharedTags) < opts.MinSharedTags {
			continue
		}
		// ensure deterministic order of SharedTags & weight
		copyTags := append([]string(nil), e.SharedTags...)
		sort.Strings(copyTags)
		edge := GraphEdge{From: e.From, To: e.To, SharedTags: copyTags, Weight: len(copyTags)}
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
	if opts.MaxEdges > 0 && len(edges) > opts.MaxEdges {
		edges = edges[:opts.MaxEdges]
	}
	posts := make([]*Post, 0, len(postCache))
	for _, p := range postCache {
		posts = append(posts, p)
	}
	return &TagGraph{Posts: posts, Edges: edges, TagIndex: snapshotTagIndex(tagIndex)}
}

// slicesEqual compares two string slices ignoring order; used for option re-use check.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// compare ordered copies
	aCopy := append([]string(nil), a...)
	bCopy := append([]string(nil), b...)
	sort.Strings(aCopy)
	sort.Strings(bCopy)
	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}
