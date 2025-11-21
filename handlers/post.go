package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/soockee/cybersocke.com/components"
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
		return errors.New("method not allowed")
	}
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) error {
	idStr := mux.Vars(r)["id"]
	post, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return err
	}
	cleaned := services.StripDataview(post.Content)
	md := services.RenderMD(cleaned)
	// Build adjacency via service (include all tags, minShared=1, limit=12)
	neighbors, err := services.ComputeAdjacency(h.postService, post.Meta.Slug, map[string]struct{}{}, 1, 12, r.Context())
	if err != nil {
		return err
	}
	entries := make([]components.AdjacencyEntry, 0, len(neighbors))
	for _, n := range neighbors {
		entries = append(entries, components.AdjacencyEntry{Slug: n.Slug, Name: n.Name, Weight: n.Weight, SharedTags: n.SharedTags, Date: n.Date})
	}
	// Derive created time (frontmatter validation stores time.Time in Created)
	createdTs, _ := post.Meta.Created.(time.Time)
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
	file, header, err := r.FormFile("file")
	if err != nil {
		// Propagate error so wrapper can set status code
		return fmt.Errorf("reading form file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("reading file content: %w", err)
	}

	// Debug: print size only (avoid dumping content)
	h.Log.Info("upload received name=%s size=%d bytes\n", header.Filename, len(content))

	// Sanitize original filename and ensure .md extension for slug derivation in storage layer
	original := filepath.Base(header.Filename)
	if err := h.postService.CreatePost(content, original, r.Context()); err != nil {
		return err
	}
	// Derive slug for client confirmation (mirrors storage slug derivation logic)
	slug := storage.SanitizeFilename(original)
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
	idStr := mux.Vars(r)["id"]
	post, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return err
	}
	related, _ := h.postService.GetRelatedPosts(post.Meta.Slug, 12, r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return components.PostFragment(components.PostFragmentProps{Post: post, Related: related}).Render(r.Context(), w)
}
