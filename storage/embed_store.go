package storage

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"net/http"
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

	files, err := assets.ReadDir(postDir)
	if err != nil {
		return nil, err
	}

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
			// Derive date from updated field on read if zero
			if postMeta.Date.IsZero() && strings.TrimSpace(postMeta.UpdatedRaw) != "" {
				if ts := parseTimestamp(postMeta.UpdatedRaw); !ts.IsZero() {
					postMeta.Date = ts
				}
			}
			posts[postMeta.Slug] = Post{Meta: postMeta, Content: content}
		}
	}

	public, err := fs.Sub(assets, publicDir)
	if err != nil {
		return nil, err
	}
	fs := http.FileServer(http.FS(public))

	return &EmbedStore{
		assets: assets,
		posts:  posts,
		fs:     fs,
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
