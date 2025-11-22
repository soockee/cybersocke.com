package htmlhandler

import (
	"log/slog"
	"net/http"

	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/middleware"
)

// LogoutHandler clears the stored ID token and expires the session cookie.
type LogoutHandler struct {
	Log *slog.Logger
}

func NewLogoutHandler(log *slog.Logger) *LogoutHandler {
	return &LogoutHandler{Log: log}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}
	if err := h.Get(w, r); err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
	}
}

// GetLogout godoc
// @Summary Logout current user
// @Description Clears the session cookie and redirects to the public homepage.
// @Tags auth
// @Produce text/html
// @Success 303 {string} string "Redirect to /"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/logout [get]
func (h *LogoutHandler) Get(w http.ResponseWriter, r *http.Request) error {
	sess := middleware.GetSession(r)
	if sess != nil {
		for k := range sess.Values {
			delete(sess.Values, k)
		}
		sess.Options.MaxAge = -1
		if h.Log != nil {
			h.Log.Info("user logged out", slog.String("path", r.URL.Path))
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
