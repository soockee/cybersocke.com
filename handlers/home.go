package handlers

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/soockee/cybersocke.com/components"
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
		return errors.New("method not allowed")
	}
}

func (h *HomeHandler) Get(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	selected := h.tagService.ParseSelectedTags(r.URL.Query().Get("tags"))
	posts, err := h.postService.GetPosts(ctx)
	if err != nil {
		return err
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
		props := components.PostViewProps{Content: empty, Title: title, Slug: "", Tags: selected, Related: []*storage.Post{}}
		components.ExplorerLayout(components.ExplorerLayoutProps{Post: props, Adjacency: entries, Tags: selected, Authed: authed}).Render(ctx, w)
		return nil
	}
	// Default starting post view
	starting := h.postService.ChooseStartingPost(posts)
	if starting == nil {
		return errors.New("no posts available")
	}
	md := services.RenderMD(starting.Content)
	neighbors, err := services.ComputeAdjacency(h.postService, starting.Meta.Slug, nil, 1, 12, ctx)
	if err != nil {
		return err
	}
	entries := make([]components.AdjacencyEntry, 0, len(neighbors))
	for _, n := range neighbors {
		entries = append(entries, components.AdjacencyEntry{Slug: n.Slug, Name: n.Name, Weight: n.Weight, SharedTags: n.SharedTags, Date: n.Date})
	}
	props := components.PostViewProps{Content: md, Title: starting.Meta.Name, Slug: starting.Meta.Slug, Tags: starting.Meta.Tags, Related: []*storage.Post{}}
	components.ExplorerLayout(components.ExplorerLayoutProps{Post: props, Adjacency: entries, Tags: []string{}, Authed: authed}).Render(ctx, w)
	return nil
}

func (h *HomeHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}

// buildTagAdjacency constructs adjacency entries for posts intersecting with any of the selected tags.
