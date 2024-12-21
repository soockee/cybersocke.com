package storage

import (
	"maps"
	"slices"
	"time"
)

type Storage interface {
	GetPost(id string) BlogPost
	GetPosts() map[string]BlogPost
}

type PostMeta struct {
	Name        string    `yaml:"name"`
	Slug        string    `yaml:"slug"`
	Tags        []string  `yaml:"tags"`
	Date        time.Time `yaml:"date"`
	Description string    `yaml:"description"`
}

type BlogPost struct {
	Meta    PostMeta
	Content []byte
}

func SortBlogPostMap(posts map[string]BlogPost) []BlogPost {
	it := maps.Values(posts)
	s := []BlogPost{}
	for post := range it {
		s = append(s, post)
	}
	return SortPostsByDate(s)
}

func SortPostsByDate(posts []BlogPost) []BlogPost {
	// Sort in descending order (newest first)
	slices.SortFunc(posts, func(a, b BlogPost) int {
		if a.Meta.Date.After(b.Meta.Date) {
			return -1 // a is newer, comes first
		} else if a.Meta.Date.Before(b.Meta.Date) {
			return 1 // b is newer, comes first
		}
		return 0 // Dates are equal
	})
	return posts
}
