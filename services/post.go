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

func (s *PostService) GetPost(id string) []byte {
	post := s.store.GetPost(id)
	return post.Content
}

func (s *PostService) GetPosts() map[string]storage.Post {
	return s.store.GetPosts()
}

func (s *PostService) SearchPost(id string) []string {
	return []string{}
}

func (s *PostService) CreatePost(data []byte, ctx context.Context) error {
	return s.store.CreatePost(data, ctx)
}
