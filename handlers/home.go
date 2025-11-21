package handlers

import (
	"bytes"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/internal/httpx"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

type HomeHandler struct {
	Log         *slog.Logger
	postService *services.PostService
	tagService  *services.TagService
}

func NewHomeHandler(post *services.PostService, tags *services.TagService, log *slog.Logger) *HomeHandler {
	return &HomeHandler{
		Log:         log,
		postService: post,
		tagService:  tags,
	}
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return httpx.ErrMethodNotAllowed
	}
}

func (h *HomeHandler) Get(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	selected := h.tagService.ParseSelectedTags(r.URL.Query().Get("tags"))
	posts, err := h.postService.GetPosts(ctx)
	if err != nil {
		return httpx.Classify(err)
	}
	authed := isAuthed(r)
	// Tag-centric view
	if len(selected) > 0 {
		domainEntries := services.ComputeTagAdjacency(posts, selected, 50)
		entries := make([]components.AdjacencyEntry, 0, len(domainEntries))
		for _, e := range domainEntries {
			entries = append(entries, components.AdjacencyEntry{Slug: e.Slug, Name: e.Name, Weight: e.Weight, SharedTags: e.SharedTags, Date: e.Date})
		}
		var empty bytes.Buffer
		title := strings.Join(selected, ", ")
		// Tag-centric view uses an empty post shell; leave times zero and published false.
		props := components.PostViewProps{
			Content: empty,
			Title:   title,
			Slug:    "",
			Tags:    selected,
			Related: []*storage.Post{},
		}
		components.ExplorerLayout(components.ExplorerLayoutProps{Post: props, Adjacency: entries, Tags: selected, Authed: authed}).Render(ctx, w)
		return nil
	}
	// Default starting post view
	starting := h.postService.ChooseStartingPost(posts)
	if starting == nil {
		return httpx.NotFound("no posts available")
	}
	md := services.RenderMD(starting.Content)
	neighbors, err := services.ComputeAdjacency(h.postService, starting.Meta.Slug, nil, 1, 12, ctx)
	if err != nil {
		return httpx.Classify(err)
	}
	entries := make([]components.AdjacencyEntry, 0, len(neighbors))
	for _, n := range neighbors {
		entries = append(entries, components.AdjacencyEntry{Slug: n.Slug, Name: n.Name, Weight: n.Weight, SharedTags: n.SharedTags, Date: n.Date})
	}
	// Created already parsed during validation / load
	createdTs := starting.Meta.Created
	// Build tag families similar to PostHandler
	families := map[string][]string{}
	for _, t := range starting.Meta.Tags {
		parts := strings.SplitN(t, "/", 2)
		if len(parts) != 2 {
			continue
		}
		families[parts[0]] = append(families[parts[0]], parts[1])
	}
	for k := range families {
		sort.Strings(families[k])
	}
	props := components.PostViewProps{
		Content:     md,
		Title:       starting.Meta.Name,
		Slug:        starting.Meta.Slug,
		Tags:        starting.Meta.Tags,
		Related:     []*storage.Post{},
		Lead:        starting.Meta.Lead,
		Created:     createdTs,
		Updated:     starting.Meta.Updated,
		Published:   starting.Meta.Published,
		Aliases:     starting.Meta.Aliases,
		TagFamilies: families,
	}
	components.ExplorerLayout(components.ExplorerLayoutProps{Post: props, Adjacency: entries, Tags: []string{}, Authed: authed}).Render(ctx, w)
	return nil
}

func (h *HomeHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return httpx.ErrMethodNotAllowed
}

// buildTagAdjacency constructs adjacency entries for posts intersecting with any of the selected tags.
