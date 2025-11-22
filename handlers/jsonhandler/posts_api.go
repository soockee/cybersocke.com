package jsonhandler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/jsonresp"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage/models"
)

const pageSize = 20

type PostsAPIHandler struct {
	Log         *slog.Logger
	postService *services.PostService
}

type PostsAPIResponse struct {
	Posts   []PostSummary `json:"posts"`
	Page    int           `json:"page"`
	HasMore bool          `json:"hasMore"`
}

type PostSummary struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Lead    string `json:"lead"`
	Updated string `json:"updated"`
}

func NewPostsAPIHandler(postService *services.PostService, log *slog.Logger) *PostsAPIHandler {
	return &PostsAPIHandler{
		Log:         log,
		postService: postService,
	}
}

// GetPosts godoc
// @Summary List published posts
// @Description Returns a paginated list of published posts, ordered by updated date descending.
// @Tags posts
// @Produce json
// @Param page query int false "Page number (1-based, default 1)"
// @Success 200 {object} PostsAPIResponse
// @Failure 400 {object} map[string]string "Bad request or invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/posts [get]
func (h *PostsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}

	// Parse page parameter
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}

	// Get all posts from service
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

	// Sort by updated descending (newest first)
	sortedPosts := models.SortPostsByDate(publishedPosts)

	// Calculate pagination
	total := len(sortedPosts)
	start := (page - 1) * pageSize
	end := start + pageSize

	// Handle edge cases
	if start >= total {
		// Beyond total, return empty
		if err := jsonresp.Write(w, http.StatusOK, PostsAPIResponse{Posts: []PostSummary{}, Page: page, HasMore: false}); err != nil {
			handlers.WriteHTTPError(w, r, h.Log, handlers.Internal(err))
		}
		return
	}

	if end > total {
		end = total
	}

	// Slice to get current page
	pagePosts := sortedPosts[start:end]
	hasMore := end < total

	// Convert to summaries
	summaries := make([]PostSummary, len(pagePosts))
	for i, post := range pagePosts {
		summaries[i] = PostSummary{
			Slug:    post.Meta.Slug,
			Title:   post.Meta.Name,
			Lead:    post.Meta.Lead,
			Updated: post.Meta.Updated.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Return JSON response
	if err := jsonresp.Write(w, http.StatusOK, PostsAPIResponse{Posts: summaries, Page: page, HasMore: hasMore}); err != nil {
		handlers.WriteHTTPError(w, r, h.Log, handlers.Internal(err))
	}
}
