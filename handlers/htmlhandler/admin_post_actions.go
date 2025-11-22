package htmlhandler

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/config"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
	"github.com/soockee/cybersocke.com/respond/jsonresp"
	"github.com/soockee/cybersocke.com/services"
)

type AdminEditHandler struct {
	log         *slog.Logger
	postService *services.PostService
}

func NewAdminEditHandler(postSvc *services.PostService, log *slog.Logger) *AdminEditHandler {
	return &AdminEditHandler{log: log, postService: postSvc}
}

func (h *AdminEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if err := h.Get(w, r); err != nil {
			handlers.WriteHTTPError(w, r, h.log, err)
		}
		return
	}
	if r.Method == http.MethodPost {
		// Update
		if err := h.Post(w, r); err != nil {
			handlers.WriteHTTPError(w, r, h.log, err)
		}
		return
	}
	handlers.WriteHTTPError(w, r, h.log, nil)
}

// GetAdminEditPage godoc
// @Summary Edit post form
// @Description Renders the HTML editor for a post along with schema and CSRF token.
// @Tags admin
// @Produce text/html
// @Param slug path string true "Post slug"
// @Success 200 {string} string "HTML page"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {string} string "Post not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/posts/{slug}/edit [get]
func (h *AdminEditHandler) Get(w http.ResponseWriter, r *http.Request) error {
	slug := strings.TrimPrefix(r.URL.Path, "/admin/posts/")
	slug = strings.TrimSuffix(slug, "/edit")
	post, err := h.postService.GetPost(slug, r.Context())
	if err != nil {
		return err
	}
	if post == nil {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	raw, err := h.postService.GetRaw(slug, r.Context())
	if err != nil {
		return err
	}
	post.Content = raw // ensure we have full raw (frontmatter+body) for editing
	schemaJSON := "{}"
	if config.Contract != nil {
		if raw, err := config.Contract.EditorJSON(); err == nil {
			schemaJSON = raw
		}
	}
	props := components.AdminEditProps{Post: post, Slug: slug, CSRFToken: csrf.Token(r), SchemaJSON: schemaJSON, User: handlers.NavUser(r)}
	if err := htmlresp.Render(w, http.StatusOK, r.Context(), components.AdminEdit(props)); err != nil {
		return err
	}
	return nil
}

// PostAdminEdit godoc
// @Summary Update a post
// @Description Accepts the raw markdown payload and updates the post, redirecting back to /admin on success.
// @Tags admin
// @Accept text/plain
// @Produce text/html
// @Param slug path string true "Post slug"
// @Param body body string true "Raw markdown + frontmatter"
// @Success 303 {string} string "Redirect to /admin"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/posts/{slug}/edit [post]
func (h *AdminEditHandler) Post(w http.ResponseWriter, r *http.Request) error {
	slug := strings.TrimPrefix(r.URL.Path, "/admin/posts/")
	slug = strings.TrimSuffix(slug, "/edit")
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return handlers.Internal(err)
	}
	if err := h.postService.UpdatePost(slug, data, r.Context()); err != nil {
		h.log.Error("update post failed", slog.String("slug", slug), slog.Any("err", err))
		return handlers.Internal(err)
	}
	h.log.Info("post updated successfully", slog.String("slug", slug))
	w.Header().Set("Location", "/admin")
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

type AdminDeleteHandler struct {
	log         *slog.Logger
	postService *services.PostService
}

func NewAdminDeleteHandler(postSvc *services.PostService, log *slog.Logger) *AdminDeleteHandler {
	return &AdminDeleteHandler{log: log, postService: postSvc}
}

// PostAdminDelete godoc
// @Summary Delete a post
// @Description Deletes the specified post and returns a JSON confirmation.
// @Tags admin
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param slug path string true "Post slug"
// @Success 200 {object} map[string]string "Deletion confirmation"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Post not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/posts/{slug}/delete [post]
func (h *AdminDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handlers.WriteHTTPError(w, r, h.log, nil)
		return
	}
	slug := strings.TrimPrefix(r.URL.Path, "/admin/posts/")
	slug = strings.TrimSuffix(slug, "/delete")
	if err := h.postService.DeletePost(slug, r.Context()); err != nil {
		handlers.WriteHTTPError(w, r, h.log, err)
		return
	}
	jsonresp.Error(w, http.StatusOK, "ok")
}
