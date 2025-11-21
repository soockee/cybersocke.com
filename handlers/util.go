package handlers

import (
	"net/http"

	"github.com/soockee/cybersocke.com/session"
)

// isAuthed returns true if the authentication middleware has stored a verified token in context.
func isAuthed(r *http.Request) bool {
	// Token placed in context only after successful verification & revocation check by middleware.
	return r.Context().Value(session.IdTokenKey) != nil
}
