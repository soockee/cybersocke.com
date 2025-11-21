package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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

func (h *AuthCallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if err := h.Post(w, r); err != nil {
			writeHTTPError(w, r, h.Log, err)
		}
	case http.MethodGet:
		writeHTTPError(w, r, h.Log, fmt.Errorf("GET not allowed")) // GET no longer supported explicitly
	default:
		writeHTTPError(w, r, h.Log, fmt.Errorf("GET not allowed"))
	}
}

func (h *AuthCallbackHandler) Post(w http.ResponseWriter, r *http.Request) error {
	// Decode JSON payload (expects {"idToken":"<token>"}). Unknown fields rejected.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req SessionRequest
	if err := dec.Decode(&req); err != nil {
		return BadRequest("invalid request", err)
	}
	if req.IDToken == "" {
		return BadRequest("missing idToken", nil)
	}

	s := middleware.GetSession(r)
	if s == nil {
		return Internal(errors.New("session missing"))
	}
	// Persist ID token into session cookie; actual save occurs in session middleware.
	s.Values["id_token"] = req.IDToken

	// Explicit success response to ensure status code is logged and proxy receives a valid HTTP response.
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *AuthCallbackHandler) Get(w http.ResponseWriter, r *http.Request) error {
	return ErrMethodNotAllowed
}
