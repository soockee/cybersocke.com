package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

type HomeHandler struct {
	Log         *slog.Logger
	postService *services.PostService
	authService *services.AuthService
	tagService  *services.TagService
}

func NewHomeHandler(post *services.PostService, auth *services.AuthService, tags *services.TagService, log *slog.Logger) *HomeHandler {
	return &HomeHandler{
		Log:         log,
		postService: post,
		authService: auth,
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
	raw := r.URL.Query().Get("tags")
	selected := h.tagService.ParseSelectedTags(raw)
	var posts map[string]*storage.Post
	var err error
	if len(selected) > 0 {
		posts, err = h.postService.GetPostsByTags(selected, false, ctx)
	} else {
		posts, err = h.postService.GetPosts(ctx)
	}
	if err != nil {
		return err
	}
	summary := h.tagService.BuildSummary(posts, selected)

	// default values
	authed := false
	csrfToken := csrf.Token(r)

	session := middleware.GetSession(r)
	if token, ok := session.Values["id_token"].(string); ok {
		if token, err := h.authService.Verify(token, r.Context()); err == nil {
			slog.Info("", slog.Any("claims", token.Claims))
			authed = true
		}
	}

	h.View(w, r, components.HomeViewProps{
		Posts:        posts,
		CSRFToken:    csrfToken,
		Authed:       authed,
		TagCounts:    summary.TagCounts,
		TagOrder:     summary.TagOrder,
		SelectedTags: selected,
		Suggested:    summary.Suggested,
	})
	return nil
}

func (h *HomeHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}

func (h *HomeHandler) View(w http.ResponseWriter, r *http.Request, props components.HomeViewProps) {
	components.Home(props).Render(r.Context(), w)
}
