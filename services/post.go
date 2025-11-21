package services

import (
	"context"
	"sort"

	firebaseauth "firebase.google.com/go/v4/auth"

	"github.com/soockee/cybersocke.com/session"
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
	if post == nil {
		return nil, nil
	}
	if !post.Meta.Published {
		// Allow access if authenticated (firebase token present in context)
		if tok, _ := ctx.Value(session.IdTokenKey).(*firebaseauth.Token); tok == nil {
			return nil, nil // treat unpublished as not found to unauthenticated callers
		}
	}
	return post, nil
}

func (s *PostService) GetPosts(ctx context.Context) (map[string]*storage.Post, error) {
	all, err := s.store.GetPosts(ctx)
	if err != nil {
		return nil, err
	}
	// Filter unpublished unless authenticated
	authed := ctx.Value(session.IdTokenKey) != nil
	if !authed {
		filtered := make(map[string]*storage.Post)
		for slug, p := range all {
			if p.Meta.Published {
				filtered[slug] = p
			}
		}
		return filtered, nil
	}
	return all, nil
}

// GetPostsByTag returns posts containing the tag ordered by date desc (slug asc tie-breaker).
// limit <= 0 means no cap. Returns an empty slice if tag is empty.
func (s *PostService) GetPostsByTag(tag string, limit int, ctx context.Context) ([]*storage.Post, error) {
	if tag == "" {
		return []*storage.Post{}, nil
	}
	all, err := s.store.GetPosts(ctx)
	if err != nil {
		return nil, err
	}
	authed := ctx.Value(session.IdTokenKey) != nil
	out := make([]*storage.Post, 0)
	for _, p := range all {
		if !authed && !p.Meta.Published {
			continue
		}
		for _, t := range p.Meta.Tags {
			if t == tag {
				out = append(out, p)
				break
			}
		}
	}
	if len(out) == 0 {
		return out, nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Meta.Updated.Equal(out[j].Meta.Updated) {
			return out[i].Meta.Slug < out[j].Meta.Slug
		}
		return out[i].Meta.Updated.After(out[j].Meta.Updated)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// GetPostsByTags returns posts matching ALL provided tags (if all=true) or ANY if all=false.
// If tags is empty, returns all posts.
func (s *PostService) GetPostsByTags(tags []string, all bool, ctx context.Context) (map[string]*storage.Post, error) {
	if len(tags) == 0 {
		return s.GetPosts(ctx)
	}
	posts, err := s.store.GetPostsByTags(ctx, tags, all)
	if err != nil {
		return nil, err
	}
	// Convert slice to map to align with existing HomeViewProps expectations.
	result := make(map[string]*storage.Post, len(posts))
	authed := ctx.Value(session.IdTokenKey) != nil
	for _, p := range posts {
		if !authed && !p.Meta.Published {
			continue
		}
		result[p.Meta.Slug] = p
	}
	return result, nil
}

// GetRelatedPosts returns related posts for a given slug, ranked by shared tags.
func (s *PostService) GetRelatedPosts(slug string, limit int, ctx context.Context) ([]*storage.Post, error) {
	posts, err := s.store.GetRelatedPosts(ctx, slug, limit)
	if err != nil {
		return nil, err
	}
	authed := ctx.Value(session.IdTokenKey) != nil
	if authed {
		return posts, nil
	}
	filtered := make([]*storage.Post, 0, len(posts))
	for _, p := range posts {
		if p.Meta.Published {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

func (s *PostService) SearchPost(slug string, ctx context.Context) []string {
	return []string{}
}

func (s *PostService) CreatePost(data []byte, originalFilename string, ctx context.Context) error {
	return s.store.CreatePost(data, originalFilename, ctx)
}

// ChooseStartingPost selects the newest post (date desc; slug asc tie-breaker) from a map.
// Returns nil if map is empty.
func (s *PostService) ChooseStartingPost(posts map[string]*storage.Post) *storage.Post {
	if len(posts) == 0 {
		return nil
	}
	candidates := make([]*storage.Post, 0, len(posts))
	for _, p := range posts {
		candidates = append(candidates, p)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Meta.Updated.Equal(candidates[j].Meta.Updated) {
			return candidates[i].Meta.Slug < candidates[j].Meta.Slug
		}
		return candidates[i].Meta.Updated.After(candidates[j].Meta.Updated)
	})
	return candidates[0]
}
