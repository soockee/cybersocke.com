package handlers

import (
	"net/http"
	"strings"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/session"
)

// IsAuthed reports whether the authentication middleware stored a verified token in context.
func IsAuthed(r *http.Request) bool {
	// Token placed in context only after successful verification & revocation check by middleware.
	return r.Context().Value(session.IdTokenKey) != nil
}

// NavUser builds a lightweight profile for navbar display from the verified ID token, if present.
func NavUser(r *http.Request) *components.NavUser {
	if r == nil {
		return nil
	}
	tok, ok := session.TokenFromContext(r.Context())
	if !ok || tok == nil {
		return nil
	}
	claims := tok.Claims
	name := claimString(claims, "name")
	if name == "" {
		name = claimString(claims, "displayName")
	}
	if name == "" && tok.UID != "" {
		name = tok.UID
	}
	email := strings.ToLower(claimString(claims, "email"))
	return &components.NavUser{
		UID:        tok.UID,
		Name:       name,
		Email:      email,
		PictureURL: claimString(claims, "picture"),
	}
}

func claimString(claims map[string]interface{}, key string) string {
	if claims == nil {
		return ""
	}
	v, ok := claims[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
