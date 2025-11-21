package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/soockee/cybersocke.com/components"
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
		writeHTTPError(w, r, h.Log, ErrMethodNotAllowed) // still disallowed
	case http.MethodGet:
		if err := h.Get(w, r); err != nil {
			writeHTTPError(w, r, h.Log, err)
		}
	default:
		writeHTTPError(w, r, h.Log, ErrMethodNotAllowed)
	}
}

func (h *LoginHandler) Get(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	h.View(w, r, components.LoginViewProps{
		FirebaseInsensitiveAPIKey: os.Getenv("FIREBASE_INSENSITIVE_API_KEY"),
		FirebaseAuthDomain:        os.Getenv("FIREBASE_AUTH_DOMAIN"),
	})
	return nil
}

func (h *LoginHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return ErrMethodNotAllowed
}

func (h *LoginHandler) View(w http.ResponseWriter, r *http.Request, props components.LoginViewProps) {
	components.Login(props).Render(r.Context(), w)
}
