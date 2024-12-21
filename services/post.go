package services

import (
	"bytes"

	"github.com/soockee/cybersocke.com/storage"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type PostService struct {
	store storage.Storage
}

func NewPostService(store storage.Storage) *PostService {
	return &PostService{
		store: store,
	}
}

func (s *PostService) GetPost(id string) []byte {
	post := s.store.GetPost(id)
	return post.Content
}

func (s *PostService) GetPosts() map[string]storage.BlogPost {
	return s.store.GetPosts()
}

func (s *PostService) SearchPost(id string) []string {
	return []string{}
}

func (s *PostService) RenderMD(source []byte) bytes.Buffer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		panic(err)
	}
	return buf
}
