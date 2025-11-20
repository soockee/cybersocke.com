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

// GetPostsByTags returns posts matching ALL provided tags (if all=true) or ANY if all=false.
// If tags is empty, returns all posts.
func (s *PostService) GetPostsByTags(tags []string, all bool, ctx context.Context) (map[string]*storage.Post, error) {
	if len(tags) == 0 {
		return s.store.GetPosts(ctx)
	}
	posts, err := s.store.GetPostsByTags(ctx, tags, all)
	if err != nil {
		return nil, err
	}
	// Convert slice to map to align with existing HomeViewProps expectations.
	result := make(map[string]*storage.Post, len(posts))
	for _, p := range posts {
		result[p.Meta.Slug] = p
	}
	return result, nil
}

// GetRelatedPosts returns related posts for a given slug, ranked by shared tags.
func (s *PostService) GetRelatedPosts(slug string, limit int, ctx context.Context) ([]*storage.Post, error) {
	return s.store.GetRelatedPosts(ctx, slug, limit)
}

func (s *PostService) SearchPost(slug string, ctx context.Context) []string {
	return []string{}
}

func (s *PostService) CreatePost(data []byte, originalFilename string, ctx context.Context) error {
	return s.store.CreatePost(data, originalFilename, ctx)
}
