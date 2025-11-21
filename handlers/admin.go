package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/internal/httpx"
	"github.com/soockee/cybersocke.com/services"
)

// AdminHandler serves the admin navigator interface (post list + upload box)
type AdminHandler struct {
	Log         *slog.Logger
	postService *services.PostService
	authService *services.AuthService
}

func NewAdminHandler(posts *services.PostService, auth *services.AuthService, log *slog.Logger) *AdminHandler {
	return &AdminHandler{Log: log, postService: posts, authService: auth}
}

func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return httpx.ErrMethodNotAllowed
	}
}

func (h *AdminHandler) Get(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	posts, err := h.postService.GetPosts(ctx)
	if err != nil {
		return httpx.Classify(err)
	}
	// Retrieve real CSRF token provided by gorilla/csrf middleware.
	csrfToken := csrf.Token(r)
	// Determine authentication from context (verified id token presence).
	authed := isAuthed(r)
	props := components.AdminViewProps{Posts: posts, CSRFToken: csrfToken, Authed: authed, ThemeTags: services.CollectThemeTags(posts)}
	components.Admin(props).Render(ctx, w)
	return nil
}
