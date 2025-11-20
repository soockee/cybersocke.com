package services

import (
	"sort"
	"strings"

	"github.com/soockee/cybersocke.com/storage"
)

type TagService struct{}

type TagSummary struct {
	TagCounts map[string]int
	TagOrder  []string
	Suggested []string
}

func NewTagService() *TagService { return &TagService{} }

// ParseSelectedTags parses a raw comma-separated tag query string.
func (ts *TagService) ParseSelectedTags(raw string) []string {
	selected := []string{}
	if strings.TrimSpace(raw) == "" {
		return selected
	}
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			selected = append(selected, p)
		}
	}
	return selected
}

// BuildSummary computes tag frequency, ordering, and suggested co-occurring tags
// based on the provided posts and currently selected tag filters.
func (ts *TagService) BuildSummary(posts map[string]*storage.Post, selected []string) TagSummary {
	counts := map[string]int{}
	for _, p := range posts {
		for _, t := range p.Meta.Tags {
			if t == "" {
				continue
			}
			counts[t]++
		}
	}
	order := make([]string, 0, len(counts))
	for t := range counts {
		order = append(order, t)
	}
	sort.Slice(order, func(i, j int) bool {
		if counts[order[i]] == counts[order[j]] {
			return order[i] < order[j]
		}
		return counts[order[i]] > counts[order[j]]
	})
	// Suggested tags: tags appearing with ALL selected
	suggestedCounts := map[string]int{}
	selectedSet := map[string]struct{}{}
	for _, s := range selected {
		selectedSet[s] = struct{}{}
	}
	if len(selected) > 0 {
		for _, p := range posts {
			if !includesAll(p.Meta.Tags, selected) {
				continue
			}
			for _, t := range p.Meta.Tags {
				if _, isSel := selectedSet[t]; isSel {
					continue
				}
				suggestedCounts[t]++
			}
		}
	}
	suggested := make([]string, 0, len(suggestedCounts))
	for t := range suggestedCounts {
		suggested = append(suggested, t)
	}
	sort.Slice(suggested, func(i, j int) bool {
		if suggestedCounts[suggested[i]] == suggestedCounts[suggested[j]] {
			return suggested[i] < suggested[j]
		}
		return suggestedCounts[suggested[i]] > suggestedCounts[suggested[j]]
	})
	if len(suggested) > 15 {
		suggested = suggested[:15]
	}
	return TagSummary{TagCounts: counts, TagOrder: order, Suggested: suggested}
}

func includesAll(tags []string, selected []string) bool {
	if len(selected) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, t := range tags {
		set[t] = struct{}{}
	}
	for _, s := range selected {
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}
