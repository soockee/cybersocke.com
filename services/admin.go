package services

import "github.com/soockee/cybersocke.com/storage"

// CollectThemeTags builds a distinct tag slice for admin filtering and navigation.
// Domain-level helper; kept in services to allow reuse beyond handlers.
func CollectThemeTags(posts map[string]*storage.Post) []string {
	uniq := make(map[string]struct{})
	for _, p := range posts {
		for _, t := range p.Meta.Tags {
			uniq[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(uniq))
	for t := range uniq {
		out = append(out, t)
	}
	return out
}
