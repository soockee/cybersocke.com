package storage

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/soockee/cybersocke.com/parser/frontmatter"
)

type EmbedStore struct {
	assets embed.FS
	posts  map[string]Post
	fs     http.Handler
}

func NewEmbedStore(postDir, publicDir string, assets embed.FS) (*EmbedStore, error) {
	posts := map[string]Post{}

	// files, err := assets.ReadDir(postDir)
	// if err != nil {
	// 	return nil, err
	// }
	// empty placeholder
	files := []fs.DirEntry{}

	for _, file := range files {
		if !file.IsDir() {
			postMeta := PostMeta{}
			path := strings.Join([]string{postDir, file.Name()}, "/")
			f, err := assets.ReadFile(path)
			if err != nil {
				return nil, err
			}
			content, err := frontmatter.Parse(strings.NewReader(string(f)), &postMeta)
			if err != nil {
				return nil, err
			}
			// Derive slug & name if missing
			if strings.TrimSpace(postMeta.Slug) == "" {
				postMeta.Slug = file.Name()
			}
			if strings.TrimSpace(postMeta.Name) == "" {
				postMeta.Name = DeriveDisplayName(postMeta.Slug)
			}
			// Parse flexible updated timestamp if provided.
			if postMeta.Updated.IsZero() && strings.TrimSpace(postMeta.UpdatedRaw) != "" {
				postMeta.Updated = parseTimestamp(postMeta.UpdatedRaw)
			}
			// Parse flexible published boolean (string or bool) if provided.
			rawPub := strings.ToLower(strings.TrimSpace(postMeta.PublishedRaw))
			switch rawPub {
			case "", "false", "no", "0", "off":
				postMeta.Published = false
			case "true", "yes", "1", "on":
				postMeta.Published = true
			default:
				postMeta.Published = false // ignore invalid
			}
			posts[postMeta.Slug] = Post{Meta: postMeta, Content: content}
		}
	}

	public, err := fs.Sub(assets, publicDir)
	if err != nil {
		return nil, err
	}
	baseFS := http.FileServer(http.FS(public))
	// Wrap to enforce correct JS MIME type (some environments default to text/plain)
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		// normalize path extension
		ext := strings.ToLower(path.Ext(p))
		if ext == ".js" {
			// Always set explicit JS MIME to avoid strict MIME rejection
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		baseFS.ServeHTTP(w, r)
	})

	return &EmbedStore{
		assets: assets,
		posts:  posts,
		fs:     wrapped,
	}, nil
}

// GetPost returns a deep copy of the stored post to prevent external modification
func (s *EmbedStore) GetPost(id string, ctx context.Context) (*Post, error) {
	orig, exists := s.posts[id]
	if !exists {
		return nil, errors.New("post not found")
	}
	if orig.Content == nil {
		return nil, errors.New("post content is empty")
	}

	copyContent := make([]byte, len(orig.Content))
	copy(copyContent, orig.Content)

	p := &Post{
		Meta:    orig.Meta,
		Content: copyContent,
	}
	return p, nil
}

// GetPosts returns copies of all posts
func (s *EmbedStore) GetPosts(ctx context.Context) (map[string]*Post, error) {
	result := make(map[string]*Post, len(s.posts))
	for id, orig := range s.posts {
		copyContent := make([]byte, len(orig.Content))
		copy(copyContent, orig.Content)
		result[id] = &Post{
			Meta:    orig.Meta,
			Content: copyContent,
		}
	}
	return result, nil
}

func (s *EmbedStore) GetAbout() []byte {
	f, err := s.assets.ReadFile("assets/content/about/about.md")
	if err != nil {
		return []byte{}
	}
	postMeta := PostMeta{}
	content, err := frontmatter.Parse(strings.NewReader(string(f)), &postMeta)
	if err != nil {
		return []byte{}
	}
	return content
}

func (s *EmbedStore) GetAssets() http.Handler {
	return s.fs
}

// CreatePost implements the Storage interface but is not supported for the embedded store.
// originalFilename is ignored. Returns an error to signal this backend is read-only.
func (s *EmbedStore) CreatePost(_ []byte, _ string, _ context.Context) error {
	return errors.New("create post not supported for embed store (read-only)")
}

// GetPostsByTags implements tag filtering for embedded posts.
// If tags is empty returns error. matchAll controls intersection semantics.
func (s *EmbedStore) GetPostsByTags(ctx context.Context, tags []string, matchAll bool) ([]*Post, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags provided")
	}
	// normalize & dedupe
	uniq := []string{}
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
	result := []*Post{}
	for _, p := range s.posts {
		if matchAll {
			matches := true
			for _, t := range uniq {
				if !hasTag(p.Meta.Tags, t) {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		} else {
			any := false
			for _, t := range uniq {
				if hasTag(p.Meta.Tags, t) {
					any = true
					break
				}
			}
			if !any {
				continue
			}
		}
		// copy content to comply with interface expectations (avoid mutation)
		copyContent := make([]byte, len(p.Content))
		copy(copyContent, p.Content)
		cp := &Post{Meta: p.Meta, Content: copyContent}
		result = append(result, cp)
	}
	return SortPostsByDate(result), nil
}

// GetRelatedPosts returns posts sharing at least one tag with slug, ranked by shared count desc then date desc then slug asc.
func (s *EmbedStore) GetRelatedPosts(ctx context.Context, slug string, limit int) ([]*Post, error) {
	current, ok := s.posts[slug]
	if !ok {
		return nil, errors.New("post not found")
	}
	sharedCounts := map[string]int{}
	for _, tag := range current.Meta.Tags {
		for otherSlug, p := range s.posts {
			if otherSlug == slug {
				continue
			}
			if hasTag(p.Meta.Tags, tag) {
				sharedCounts[otherSlug]++
			}
		}
	}
	related := []*Post{}
	for otherSlug := range sharedCounts {
		p := s.posts[otherSlug]
		copyContent := make([]byte, len(p.Content))
		copy(copyContent, p.Content)
		related = append(related, &Post{Meta: p.Meta, Content: copyContent})
	}
	sort.Slice(related, func(i, j int) bool {
		ci := sharedCounts[related[i].Meta.Slug]
		cj := sharedCounts[related[j].Meta.Slug]
		if ci != cj {
			return ci > cj
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

// hasTag checks membership
func hasTag(tags []string, t string) bool {
	for _, x := range tags {
		if x == t {
			return true
		}
	}
	return false
}
