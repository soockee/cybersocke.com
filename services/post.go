package services

import (
	"context"

	"github.com/soockee/cybersocke.com/storage"
)

type PostService struct {
	authService *AuthService
	store       storage.Storage
}

func NewPostService(store storage.Storage, authService *AuthService) *PostService {
	return &PostService{
		authService: authService,
		store:       store,
	}
}

func (s *PostService) GetPost(slug string, ctx context.Context) (*storage.Post, error) {
	post, err := s.store.GetPost(slug, ctx)
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (s *PostService) GetPosts(ctx context.Context) (map[string]*storage.Post, error) {
	return s.store.GetPosts(ctx)
}

func (s *PostService) SearchPost(slug string, ctx context.Context) []string {
	return []string{}
}

func (s *PostService) CreatePost(data []byte, originalFilename string, ctx context.Context) error {
	return s.store.CreatePost(data, originalFilename, ctx)
}
