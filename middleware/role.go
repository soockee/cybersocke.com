package middleware

import (
	"log/slog"
	"net/http"
	"strings"

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
func WithRole(required string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok, _ := r.Context().Value(session.IdTokenKey).(*firebaseauth.Token)
			if tok == nil { // authentication middleware should have populated this
				if logger != nil {
					logger.Info("role check missing token", slog.String("required", required), slog.String("path", r.URL.Path), slog.String("method", r.Method))
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if !hasRole(tok, required) {
				if logger != nil {
					claimsSummary := summarizeRoleClaims(tok)
					logger.Info("role check forbidden", slog.String("required", required), slog.String("uid", tok.UID), slog.String("claims", claimsSummary), slog.String("path", r.URL.Path), slog.String("method", r.Method))
				}
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if logger != nil {
				logger.Debug("role check passed", slog.String("required", required), slog.String("uid", tok.UID), slog.String("path", r.URL.Path), slog.String("method", r.Method))
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
	c := tok.Claims
	roleStr, _ := c["role"].(string)
	rolesSlice, _ := c["roles"].([]any)
	claimTrue := func(name string) bool {
		if b, ok := c[name].(bool); ok && b {
			return true
		}
		return false
	}
	rolePresent := func(name string) bool {
		if roleStr == name {
			return true
		}
		for _, v := range rolesSlice {
			if s, ok := v.(string); ok && s == name {
				return true
			}
		}
		return false
	}
	admin := rolePresent("admin") || claimTrue("admin")
	writer := rolePresent("writer") || claimTrue("writer") || admin // admin supersets writer
	user := rolePresent("user") || claimTrue("user") || writer      // writer/admin supersets user
	switch want {
	case "admin":
		return admin
	case "writer":
		return writer
	case "user":
		return user
	default:
		return rolePresent(want) || claimTrue(want)
	}
}

// summarizeRoleClaims builds a compact string listing role-related claims for logging.
func summarizeRoleClaims(tok *firebaseauth.Token) string {
	if tok == nil {
		return "<nil>"
	}
	c := tok.Claims
	parts := []string{}
	appendIf := func(k string) {
		if v, ok := c[k]; ok {
			parts = append(parts, k+"="+toString(v))
		}
	}
	appendIf("role")
	appendIf("roles")
	appendIf("admin")
	appendIf("writer")
	appendIf("user")
	if len(parts) == 0 {
		return "<none>"
	}
	return strings.Join(parts, ";")
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return strings.Join(out, ",")
	default:
		return "?"
	}
}
