package htmlhandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
)

type AuthCallbackHandler struct {
	Log     *slog.Logger
	Service *services.AuthService
}

type SessionRequest struct {
	IDToken string `json:"idToken"`
}

func NewAuthCallbackHandler(log *slog.Logger) *AuthCallbackHandler {
	return &AuthCallbackHandler{
		Log: log,
	}
}

func (h *AuthCallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if err := h.Post(w, r); err != nil {
			handlers.WriteHTTPError(w, r, h.Log, err)
		}
	case http.MethodGet:
		handlers.WriteHTTPError(w, r, h.Log, fmt.Errorf("GET not allowed")) // GET no longer supported explicitly
	default:
		handlers.WriteHTTPError(w, r, h.Log, fmt.Errorf("GET not allowed"))
	}
}

// PostAuthCallback godoc
// @Summary Store Firebase ID token
// @Description Accepts the ID token from the client and persists it into the session cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body SessionRequest true "Session payload"
// @Success 204 {string} string "No Content"
// @Failure 400 {object} map[string]string "Invalid payload"
// @Failure 401 {object} map[string]string "Missing session"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/google/callback [post]
func (h *AuthCallbackHandler) Post(w http.ResponseWriter, r *http.Request) error {
	// Decode JSON payload (expects {"idToken":"<token>"}). Unknown fields rejected.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req SessionRequest
	if err := dec.Decode(&req); err != nil {
		return handlers.BadRequest("invalid request", err)
	}
	if req.IDToken == "" {
		return handlers.BadRequest("missing idToken", nil)
	}

	s := middleware.GetSession(r)
	if s == nil {
		return handlers.Internal(errors.New("session missing"))
	}
	// Persist ID token into session cookie; actual save occurs in session middleware.
	s.Values["id_token"] = req.IDToken

	// Explicit success response to ensure status code is logged and proxy receives a valid HTTP response.
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *AuthCallbackHandler) Get(w http.ResponseWriter, r *http.Request) error {
	return handlers.ErrMethodNotAllowed
}
