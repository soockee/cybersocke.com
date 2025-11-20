package storage

// TagGraphOptions controls graph construction.
type TagGraphOptions struct {
	MinSharedTags int      // minimum shared tags required to create an edge (default 1)
	IncludeTags   []string // optional whitelist; if non-empty only these tags considered for edges
	MaxEdges      int      // optional cap; 0 = unlimited
}

// GraphEdge represents an undirected edge between two posts.
// From is always lexicographically <= To for canonicalization.
type GraphEdge struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	SharedTags []string `json:"shared_tags"`
	Weight     int      `json:"weight"` // number of shared tags
}

// TagGraph bundles posts, their edges, and a snapshot of the tag index used.
type TagGraph struct {
	Posts    []*Post             `json:"posts"`
	Edges    []GraphEdge         `json:"edges"`
	TagIndex map[string][]string `json:"tag_index"`
}
