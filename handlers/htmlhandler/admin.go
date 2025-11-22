package htmlhandler

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
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

func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if err := h.Get(w, r); err != nil {
			handlers.WriteHTTPError(w, r, h.Log, err)
		}
	default:
		handlers.WriteHTTPError(w, r, h.Log, nil)
	}
}

// GetAdminDashboard godoc
// @Summary Admin dashboard
// @Description Renders the admin console for managing posts and uploads (requires authentication + CSRF token).
// @Tags admin
// @Produce text/html
// @Success 200 {string} string "HTML page"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin [get]
func (h *AdminHandler) Get(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	posts, err := h.postService.GetPosts(ctx)
	if err != nil {
		return err
	}
	// Retrieve real CSRF token provided by gorilla/csrf middleware.
	csrfToken := csrf.Token(r)
	// Determine authentication from context (verified id token presence).
	authed := handlers.IsAuthed(r)
	props := components.AdminViewProps{Posts: posts, CSRFToken: csrfToken, Authed: authed, ThemeTags: services.CollectThemeTags(posts), User: handlers.NavUser(r)}
	if err := htmlresp.Render(w, http.StatusOK, ctx, components.Admin(props)); err != nil {
		return err
	}
	return nil
}
