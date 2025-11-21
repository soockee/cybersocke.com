package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/soockee/cybersocke.com/internal/httpx"

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

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return httpx.ErrMethodNotAllowed
	}
}

func (h *LoginHandler) Get(w http.ResponseWriter, r *http.Request) error {
	h.View(w, r, components.LoginViewProps{
		FirebaseInsensitiveAPIKey: os.Getenv("FIREBASE_INSENSITIVE_API_KEY"),
		FirebaseAuthDomain:        os.Getenv("FIREBASE_AUTH_DOMAIN"),
	})
	return nil
}

func (h *LoginHandler) Post(w http.ResponseWriter, r *http.Request) error {
	return httpx.ErrMethodNotAllowed
}

func (h *LoginHandler) View(w http.ResponseWriter, r *http.Request, props components.LoginViewProps) {
	components.Login(props).Render(r.Context(), w)
}
