package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
)

type HomeHandler struct {
	Log         *slog.Logger
	postService *services.PostService
	authService *services.AuthService
	csrfService *services.CSFRService
}

func NewHomeHandler(post *services.PostService, auth *services.AuthService, csrf *services.CSFRService, log *slog.Logger) *HomeHandler {
	return &HomeHandler{
		Log:         log,
		postService: post,
		authService: auth,
		csrfService: csrf,
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
	posts, err := h.postService.GetPosts(r.Context())
	if err != nil {
		return err
	}

	// default values
	authed := false
	var csrfToken string
	if token, ok := middleware.GetSession(r).Values["id_token"].(string); ok {
		if _, err := h.authService.Verify(token, r.Context()); err == nil {
			authed = true
			csrfToken = csrf.Token(r)
		}
	}

	h.View(w, r, components.HomeViewProps{
		Posts:     posts,
		CSRFToken: csrfToken,
		Authed:    authed,
	})
	return nil
}

func (h *HomeHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}

func (h *HomeHandler) View(w http.ResponseWriter, r *http.Request, props components.HomeViewProps) {
	components.Home(props).Render(r.Context(), w)
}
