package storage

import (
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
			posts[postMeta.Slug] = Post{
				Meta:    postMeta,
				Content: content,
			}
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

func (s *EmbedStore) GetPost(id string) Post {
	return s.posts[id]
}

func (s *EmbedStore) GetPosts() map[string]Post {
	return s.posts
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

func (s *EmbedStore) GetFS() http.Handler {
	return s.fs
}

func (s *EmbedStore) CreatePost(post Post) error {
	return errors.New("method not implemented")
}
