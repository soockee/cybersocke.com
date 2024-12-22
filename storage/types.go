package storage

import (
	"maps"
	"net/http"
	"slices"
	"time"
)

type Storage interface {
	GetPost(id string) Post
	GetPosts() map[string]Post
	GetAbout() []byte
	GetFS() http.Handler
}

type PostMeta struct {
	Name        string    `yaml:"name"`
	Slug        string    `yaml:"slug"`
	Tags        []string  `yaml:"tags"`
	Date        time.Time `yaml:"date"`
	Description string    `yaml:"description"`
}

type Post struct {
	Meta    PostMeta
	Content []byte
}

func SortPostMap(posts map[string]Post) []Post {
	it := maps.Values(posts)
	s := []Post{}
	for post := range it {
		s = append(s, post)
	}
	return SortPostsByDate(s)
}

func SortPostsByDate(posts []Post) []Post {
	// Sort in descending order (newest first)
	slices.SortFunc(posts, func(a, b Post) int {
		if a.Meta.Date.After(b.Meta.Date) {
			return -1 // a is newer, comes first
		} else if a.Meta.Date.Before(b.Meta.Date) {
			return 1 // b is newer, comes first
		}
		return 0 // Dates are equal
	})
	return posts
}
