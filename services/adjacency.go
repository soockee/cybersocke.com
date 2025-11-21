package services

import (
	"context"
	"sort"
	"time"

	"github.com/soockee/cybersocke.com/storage"
)

// AdjacentPost represents a neighboring post with shared tag weights.
// It is domain-level (no transport concerns) but includes JSON tags so handlers can marshal directly.
type AdjacentPost struct {
	Slug       string    `json:"slug"`
	Name       string    `json:"name"`
	Weight     int       `json:"weight"`
	SharedTags []string  `json:"shared_tags"`
	Date       time.Time `json:"date"`
}

// ComputeAdjacency returns weighted neighboring posts for a given post slug.
// includeSet filters which tags are considered (empty = all tags). minShared is the minimum
// number of shared tags required; limit <= 0 means unlimited.
func ComputeAdjacency(posts *PostService, slug string, includeSet map[string]struct{}, minShared, limit int, ctx context.Context) ([]AdjacentPost, error) {
	post, err := posts.GetPost(slug, ctx)
	if err != nil {
		return nil, err
	}
	// Broad related set (limit 0 = no cap) for accurate ranking
	related, err := posts.GetRelatedPosts(slug, 0, ctx)
	if err != nil {
		return nil, err
	}
	baseSet := make(map[string]struct{}, len(post.Meta.Tags))
	for _, t := range post.Meta.Tags {
		baseSet[t] = struct{}{}
	}
	neighbors := make([]AdjacentPost, 0, len(related))
	for _, rp := range related {
		shared := []string{}
		for _, t := range rp.Meta.Tags {
			if _, ok := baseSet[t]; !ok {
				continue
			}
			if len(includeSet) > 0 {
				if _, keep := includeSet[t]; !keep {
					continue
				}
			}
			shared = append(shared, t)
		}
		if len(shared) < minShared {
			continue
		}
		neighbors = append(neighbors, AdjacentPost{Slug: rp.Meta.Slug, Name: rp.Meta.Name, Weight: len(shared), SharedTags: shared, Date: rp.Meta.Updated})
	}
	// Rank: weight desc, date desc, slug asc
	sort.Slice(neighbors, func(i, j int) bool {
		if neighbors[i].Weight != neighbors[j].Weight {
			return neighbors[i].Weight > neighbors[j].Weight
		}
		if !neighbors[i].Date.Equal(neighbors[j].Date) {
			return neighbors[i].Date.After(neighbors[j].Date)
		}
		return neighbors[i].Slug < neighbors[j].Slug
	})
	if limit > 0 && len(neighbors) > limit {
		neighbors = neighbors[:limit]
	}
	return neighbors, nil
}

// ComputeTagAdjacency builds adjacency entries for a tag-centric view. It returns posts that match
// at least one selected tag. Weight = number of selected tags the post contains. Duplicates within
// a single post are ignored. Result is ranked weight desc, date desc, slug asc and capped by limit (>0).
func ComputeTagAdjacency(all map[string]*storage.Post, selected []string, limit int) []AdjacentPost {
	// Fast path: empty selection
	if len(selected) == 0 {
		return []AdjacentPost{}
	}
	selSet := make(map[string]struct{}, len(selected))
	for _, t := range selected {
		selSet[t] = struct{}{}
	}
	entries := make([]AdjacentPost, 0, len(all))
	for _, p := range all {
		seenLocal := map[string]struct{}{}
		matched := []string{}
		for _, t := range p.Meta.Tags {
			if _, ok := selSet[t]; !ok {
				continue
			}
			if _, dup := seenLocal[t]; dup { // ignore duplicates in one post
				continue
			}
			seenLocal[t] = struct{}{}
			matched = append(matched, t)
		}
		if len(matched) == 0 {
			continue
		}
		entries = append(entries, AdjacentPost{Slug: p.Meta.Slug, Name: p.Meta.Name, Weight: len(matched), SharedTags: matched, Date: p.Meta.Updated})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Weight != entries[j].Weight {
			return entries[i].Weight > entries[j].Weight
		}
		if !entries[i].Date.Equal(entries[j].Date) {
			return entries[i].Date.After(entries[j].Date)
		}
		return entries[i].Slug < entries[j].Slug
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}
