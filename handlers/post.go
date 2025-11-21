package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"errors"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/internal/httpx"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

type PostHandler struct {
	Log         *slog.Logger
	postService *services.PostService
}

func NewPostHandler(postService *services.PostService, log *slog.Logger) *PostHandler {
	return &PostHandler{
		Log:         log,
		postService: postService,
	}
}

func (h *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		// Support fragment sub-path: /posts/{id}/fragment
		if strings.HasSuffix(r.URL.Path, "/fragment") {
			return h.Fragment(w, r)
		}
		return h.Get(w, r)
	default:
		return httpx.ErrMethodNotAllowed
	}
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	post, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return httpx.Classify(err)
	}
	cleaned := services.StripDataview(post.Content)
	md := services.RenderMD(cleaned)
	// Build adjacency via service (include all tags, minShared=1, limit=12)
	neighbors, err := services.ComputeAdjacency(h.postService, post.Meta.Slug, map[string]struct{}{}, 1, 12, r.Context())
	if err != nil {
		return httpx.Classify(err)
	}
	entries := make([]components.AdjacencyEntry, 0, len(neighbors))
	for _, n := range neighbors {
		entries = append(entries, components.AdjacencyEntry{Slug: n.Slug, Name: n.Name, Weight: n.Weight, SharedTags: n.SharedTags, Date: n.Date})
	}
	// Created already parsed during validation / load
	createdTs := post.Meta.Created
	// Group tags by family for refined display
	families := map[string][]string{}
	for _, t := range post.Meta.Tags {
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
		Title:       post.Meta.Name,
		Slug:        post.Meta.Slug,
		Tags:        post.Meta.Tags,
		Related:     []*storage.Post{},
		Lead:        post.Meta.Lead,
		Created:     createdTs,
		Updated:     post.Meta.Updated,
		Published:   post.Meta.Published,
		Aliases:     post.Meta.Aliases,
		TagFamilies: families,
	}
	authed := isAuthed(r)
	components.ExplorerLayout(components.ExplorerLayoutProps{Post: props, Adjacency: entries, Authed: authed}).Render(r.Context(), w)
	return nil
}

func (h *PostHandler) Post(w http.ResponseWriter, r *http.Request) error {
	start := time.Now()
	logger := h.Log.With(
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
	)
	// Parse file
	file, header, err := r.FormFile("file")
	if err != nil {
		logger.Info("upload form file missing", slog.Any("err", err))
		return httpx.BadRequest("invalid upload", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		logger.Info("upload read failed", slog.Any("err", err))
		return httpx.BadRequest("failed to read file", err)
	}
	if len(content) == 0 {
		logger.Info("upload empty content")
		return httpx.BadRequest("empty file", errors.New("empty file"))
	}

	original := filepath.Base(header.Filename)
	slug := storage.SanitizeFilename(original)
	logger.Debug("upload parsed", slog.String("filename", original), slog.Int("size", len(content)), slog.String("slug", slug))

	if err := h.postService.CreatePost(content, original, r.Context()); err != nil {
		logger.Error("create post failed", slog.String("slug", slug), slog.Any("err", err))
		return httpx.Classify(err)
	}

	logger.Info("upload stored", slog.String("slug", slug), slog.Duration("took", time.Since(start)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"slug":%q}`, slug)))
	return nil
}

func (h *PostHandler) View(w http.ResponseWriter, r *http.Request, props components.PostViewProps) {
	// Legacy view fallback (still available if needed elsewhere)
	components.Post(props).Render(r.Context(), w)
}

// Fragment returns a lightweight HTML snippet (without layout) for overlay navigation.
func (h *PostHandler) Fragment(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	post, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return httpx.Classify(err)
	}
	related, _ := h.postService.GetRelatedPosts(post.Meta.Slug, 12, r.Context()) // ignore classification for related fetch errors
	// Render markdown for fragment (same as full post view)
	cleaned := services.StripDataview(post.Content)
	md := services.RenderMD(cleaned)
	// Build tag families
	families := map[string][]string{}
	for _, t := range post.Meta.Tags {
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
		Title:       post.Meta.Name,
		Slug:        post.Meta.Slug,
		Tags:        post.Meta.Tags,
		Related:     related,
		Lead:        post.Meta.Lead,
		Created:     post.Meta.Created,
		Updated:     post.Meta.Updated,
		Published:   post.Meta.Published,
		Aliases:     post.Meta.Aliases,
		TagFamilies: families,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return components.PostFragment(props).Render(r.Context(), w)
}
