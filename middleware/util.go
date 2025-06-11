package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/soockee/cybersocke.com/session"
)

func GetSession(r *http.Request) *sessions.Session {
	session, ok := r.Context().Value(session.SessionKey).(*sessions.Session)
	if !ok {
		return nil
	}
	return session
}
