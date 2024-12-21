package storage

import (
	"embed"
	"strings"

	"github.com/soockee/cybersocke.com/parser/frontmatter"
)

type EmbedStore struct {
	assets    embed.FS
	blogPosts map[string]BlogPost
}

func NewEmbedStore(postDir string, assets embed.FS) (*EmbedStore, error) {
	blogPosts := map[string]BlogPost{}

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
			blogPosts[postMeta.Slug] = BlogPost{
				Meta:    postMeta,
				Content: content,
			}
		}
	}
	return &EmbedStore{
		assets:    assets,
		blogPosts: blogPosts,
	}, nil
}

func (s *EmbedStore) GetPost(id string) BlogPost {
	return s.blogPosts[id]
}

func (s *EmbedStore) GetPosts() map[string]BlogPost {
	return s.blogPosts
}
