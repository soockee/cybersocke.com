package htmlhandler

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
	"github.com/soockee/cybersocke.com/services"
)

type LoginHandler struct {
	Log     *slog.Logger
	Service *services.AuthService
}

func NewLoginHandler(log *slog.Logger) *LoginHandler {
	return &LoginHandler{
		Log: log,
	}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed) // still disallowed
	case http.MethodGet:
		if err := h.Get(w, r); err != nil {
			handlers.WriteHTTPError(w, r, h.Log, err)
		}
	default:
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
	}
}

// GetLoginPage godoc
// @Summary Login page
// @Description Renders the Firebase-based login page for administrators.
// @Tags auth
// @Produce text/html
// @Success 200 {string} string "HTML page"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth [get]
func (h *LoginHandler) Get(w http.ResponseWriter, r *http.Request) error {
	return htmlresp.Render(w, http.StatusOK, r.Context(), components.Login(components.LoginViewProps{
		FirebaseInsensitiveAPIKey: os.Getenv("FIREBASE_INSENSITIVE_API_KEY"),
		FirebaseAuthDomain:        os.Getenv("FIREBASE_AUTH_DOMAIN"),
	}))
}

func (h *LoginHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return handlers.ErrMethodNotAllowed
}

func (h *LoginHandler) View(w http.ResponseWriter, r *http.Request, props components.LoginViewProps) {
	_ = htmlresp.Render(w, http.StatusOK, r.Context(), components.Login(props))
}
