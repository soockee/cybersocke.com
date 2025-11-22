package tags

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/soockee/cybersocke.com/storage/models"
)

// Index provides tag-based indexing and querying for posts.
// Thread-safe with RWMutex.
type Index struct {
	mu    sync.RWMutex
	index map[string]map[string]struct{} // tag -> set of slugs
}

// NewIndex creates a new empty tag index.
func NewIndex() *Index {
	return &Index{
		index: make(map[string]map[string]struct{}),
	}
}

// Add indexes all tags for a given post slug.
func (idx *Index) Add(slug string, tags []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		set, ok := idx.index[t]
		if !ok {
			set = make(map[string]struct{})
			idx.index[t] = set
		}
		set[slug] = struct{}{}
	}
}

// Remove removes all tag references for a given post slug.
func (idx *Index) Remove(slug string, tags []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if set, ok := idx.index[t]; ok {
			delete(set, slug)
			if len(set) == 0 {
				delete(idx.index, t)
			}
		}
	}
}

// QueryAny returns slugs matching ANY of the provided tags (union).
func (idx *Index) QueryAny(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	resultSlugs := map[string]struct{}{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		set, ok := idx.index[t]
		if !ok {
			continue
		}
		for slug := range set {
			resultSlugs[slug] = struct{}{}
		}
	}

	slugs := make([]string, 0, len(resultSlugs))
	for slug := range resultSlugs {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

// QueryAll returns slugs matching ALL of the provided tags (intersection).
func (idx *Index) QueryAll(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	uniq := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		uniq = append(uniq, t)
	}
	if len(uniq) == 0 {
		return []string{}
	}

	firstSet, ok := idx.index[uniq[0]]
	if !ok {
		return []string{}
	}
	resultSlugs := map[string]struct{}{}
	for slug := range firstSet {
		resultSlugs[slug] = struct{}{}
	}

	for _, t := range uniq[1:] {
		set, ok := idx.index[t]
		if !ok {
			return []string{}
		}
		for slug := range resultSlugs {
			if _, present := set[slug]; !present {
				delete(resultSlugs, slug)
			}
		}
	}

	slugs := make([]string, 0, len(resultSlugs))
	for slug := range resultSlugs {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

// GetRelatedSlugs returns slugs sharing at least one tag with the given tags.
func (idx *Index) GetRelatedSlugs(postTags []string, excludeSlug string) map[string]int {
	if len(postTags) == 0 {
		return map[string]int{}
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	sharedCounts := map[string]int{}
	for _, tag := range postTags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		set, ok := idx.index[tag]
		if !ok {
			continue
		}
		for slug := range set {
			if slug == excludeSlug {
				continue
			}
			sharedCounts[slug]++
		}
	}
	return sharedCounts
}

// Rebuild clears and rebuilds the index from a map of posts.
func (idx *Index) Rebuild(posts map[string]*models.Post) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.index = make(map[string]map[string]struct{})
	for slug, p := range posts {
		for _, t := range p.Meta.Tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			set, ok := idx.index[t]
			if !ok {
				set = make(map[string]struct{})
				idx.index[t] = set
			}
			set[slug] = struct{}{}
		}
	}
}

// Snapshot returns a read-only copy of the current index state.
func (idx *Index) Snapshot() map[string][]string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := make(map[string][]string, len(idx.index))
	for tag, set := range idx.index {
		slugs := make([]string, 0, len(set))
		for slug := range set {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
		out[tag] = slugs
	}
	return out
}

// GetPostsByTags queries the index and returns full Post objects.
func GetPostsByTags(idx *Index, postCache map[string]*models.Post, tags []string, matchAll bool) ([]*models.Post, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags provided")
	}

	var slugs []string
	if matchAll {
		slugs = idx.QueryAll(tags)
	} else {
		slugs = idx.QueryAny(tags)
	}

	posts := make([]*models.Post, 0, len(slugs))
	for _, slug := range slugs {
		if p, ok := postCache[slug]; ok {
			posts = append(posts, p)
		}
	}

	sort.Slice(posts, func(i, j int) bool {
		if posts[i].Meta.Updated.Equal(posts[j].Meta.Updated) {
			return posts[i].Meta.Slug < posts[j].Meta.Slug
		}
		return posts[i].Meta.Updated.After(posts[j].Meta.Updated)
	})

	return posts, nil
}

// GetRelatedPosts returns posts sharing at least one tag with the given post.
func GetRelatedPosts(idx *Index, postCache map[string]*models.Post, slug string, limit int) ([]*models.Post, error) {
	post, ok := postCache[slug]
	if !ok {
		return nil, errors.New("post not found")
	}

	sharedCounts := idx.GetRelatedSlugs(post.Meta.Tags, slug)

	related := make([]*models.Post, 0, len(sharedCounts))
	for otherSlug := range sharedCounts {
		if p, ok := postCache[otherSlug]; ok {
			related = append(related, p)
		}
	}

	sort.Slice(related, func(i, j int) bool {
		ci := sharedCounts[related[i].Meta.Slug]
		cj := sharedCounts[related[j].Meta.Slug]
		if ci != cj {
			return ci > cj
		}
		if related[i].Meta.Updated.Equal(related[j].Meta.Updated) {
			return related[i].Meta.Slug < related[j].Meta.Slug
		}
		return related[i].Meta.Updated.After(related[j].Meta.Updated)
	})

	if limit > 0 && len(related) > limit {
		related = related[:limit]
	}

	return related, nil
}
