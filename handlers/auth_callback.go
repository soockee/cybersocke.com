package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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

func (h *AuthCallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return errors.New("method not allowed")
	}
}

func (h *AuthCallbackHandler) Post(w http.ResponseWriter, r *http.Request) error {
	var req SessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return err
	}

	session := middleware.GetSession(r)
	session.Values["id_token"] = req.IDToken
	session.Save(r, w)
	return nil
}

func (h *AuthCallbackHandler) Get(w http.ResponseWriter, r *http.Request) error {
	return errors.New("method not allowed")
}
