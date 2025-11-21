package middleware

import (
	"net/http"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/soockee/cybersocke.com/session"
)

// WithRole enforces that the authenticated Firebase user possesses the required role.
// Accepted patterns for roles:
//
//	role: "writer" OR "admin"
//	roles: ["writer", "admin"] slice
//	boolean claims: writer=true OR admin=true
//
// Admin implies writer access automatically.
func WithRole(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok, _ := r.Context().Value(session.IdTokenKey).(*firebaseauth.Token)
			if tok == nil { // middleware ensures token already verified
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if !hasRole(tok, required) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hasRole checks various claim patterns to determine role possession.
func hasRole(tok *firebaseauth.Token, want string) bool {
	if tok == nil {
		return false
	}
	claims := tok.Claims
	// Single role string
	if role, ok := claims["role"].(string); ok {
		if role == want { // admin supersets writer
			return true
		}
	}
	// roles slice
	if raw, ok := claims["roles"].([]any); ok {
		for _, v := range raw {
			if s, ok2 := v.(string); ok2 {
				if s == want {
					return true
				}
			}
		}
	}
	// Boolean style claims
	if b, ok := claims[want].(bool); ok && b {
		return true
	}
	return false
}
