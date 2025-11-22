package htmlhandler

import (
	"log/slog"
	"net/http"

	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage/models"
)

type PostsGridHandler struct {
	Log         *slog.Logger
	postService *services.PostService
}

func NewPostsGridHandler(postService *services.PostService, log *slog.Logger) *PostsGridHandler {
	return &PostsGridHandler{
		Log:         log,
		postService: postService,
	}
}

// GetPostsGrid godoc
// @Summary Posts overview grid
// @Description Renders the HTML grid listing of published posts ordered by most recently updated.
// @Tags public
// @Produce text/html
// @Success 200 {string} string "HTML page"
// @Failure 405 {object} map[string]string "Method not allowed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /posts [get]
func (h *PostsGridHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}

	// Get all posts
	posts, err := h.postService.GetPosts(r.Context())
	if err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
		return
	}

	// Filter published posts only
	publishedPosts := make([]*models.Post, 0, len(posts))
	for _, post := range posts {
		if post.Meta.Published {
			publishedPosts = append(publishedPosts, post)
		}
	}

	// Sort by updated descending
	sortedPosts := models.SortPostsByDate(publishedPosts)

	// Get first page for initial render
	pageSize := 20
	firstPagePosts := sortedPosts
	if len(sortedPosts) > pageSize {
		firstPagePosts = sortedPosts[:pageSize]
	}

	// Convert to card props
	cards := make([]components.PostCardProps, len(firstPagePosts))
	for i, post := range firstPagePosts {
		cards[i] = components.PostCardProps{
			Title:       post.Meta.Name,
			Description: post.Meta.Lead,
			Slug:        post.Meta.Slug,
			Date:        post.Meta.Updated,
			Tags:        post.Meta.Tags,
		}
	}

	// Render template with auth-aware nav
	props := components.PostsGridProps{Posts: cards, Authed: handlers.IsAuthed(r), User: handlers.NavUser(r)}
	_ = htmlresp.Render(w, http.StatusOK, r.Context(), components.PostsGrid(props))
}
