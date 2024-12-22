package services

import (
	"github.com/soockee/cybersocke.com/storage"
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

func (s *PostService) GetPosts() map[string]storage.Post {
	return s.store.GetPosts()
}

func (s *PostService) SearchPost(id string) []string {
	return []string{}
}
