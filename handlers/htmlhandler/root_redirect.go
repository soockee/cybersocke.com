package htmlhandler

import (
	"log/slog"
	"net/http"

	handlers "github.com/soockee/cybersocke.com/handlers"
)

type RootRedirectHandler struct {
	target string
	log    *slog.Logger
}

func NewRootRedirectHandler(target string, log *slog.Logger) *RootRedirectHandler {
	return &RootRedirectHandler{target: target, log: log}
}

// RootRedirect godoc
// @Summary Root redirect to posts grid
// @Description Redirects visitors hitting the root path to the /posts listing.
// @Tags public
// @Produce text/html
// @Success 302 {string} string "Redirect to /posts"
// @Failure 405 {object} map[string]string "Method not allowed"
// @Router / [get]
func (h *RootRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.log, handlers.ErrMethodNotAllowed)
		return
	}
	http.Redirect(w, r, h.target, http.StatusFound)
}
