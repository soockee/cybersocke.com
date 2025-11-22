package htmlhandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/soockee/cybersocke.com/components"
	handlers "github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/htmlresp"
	"github.com/soockee/cybersocke.com/session"
)

// ProfileHandler renders a read-only view of the current session's ID token.
type ProfileHandler struct {
	Log *slog.Logger
}

func NewProfileHandler(log *slog.Logger) *ProfileHandler {
	return &ProfileHandler{Log: log}
}

func (h *ProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}
	if err := h.Get(w, r); err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
	}
}

// GetProfilePage godoc
// @Summary User profile page
// @Description Renders a read-only profile sourced from the verified session ID token.
// @Tags auth
// @Produce text/html
// @Success 200 {string} string "HTML page"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /profile [get]
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) error {
	tok, ok := session.TokenFromContext(r.Context())
	if !ok || tok == nil {
		return &handlers.HTTPError{Status: http.StatusUnauthorized, Message: "unauthorized"}
	}
	props := components.ProfileViewProps{
		Authed:  handlers.IsAuthed(r),
		User:    handlers.NavUser(r),
		Details: buildProfileDetails(tok),
	}
	return htmlresp.Render(w, http.StatusOK, r.Context(), components.Profile(props))
}

func buildProfileDetails(tok *auth.Token) components.ProfileDetails {
	claims := tok.Claims
	name := claimLookup(claims, "name", "displayName")
	return components.ProfileDetails{
		UID:        tok.UID,
		Name:       name,
		Email:      claimLookup(claims, "email"),
		PictureURL: claimLookup(claims, "picture"),
		Issuer:     tok.Issuer,
		Audience:   audienceSlice(tok.Audience),
		AuthTime:   timeFromUnix(tok.AuthTime),
		IssuedAt:   timeFromUnix(tok.IssuedAt),
		Expires:    timeFromUnix(tok.Expires),
		Claims:     extractClaims(claims),
	}
}

func audienceSlice(aud string) []string {
	if aud == "" {
		return nil
	}
	return []string{aud}
}

var reservedClaims = map[string]struct{}{
	"aud":            {},
	"iat":            {},
	"exp":            {},
	"iss":            {},
	"sub":            {},
	"user_id":        {},
	"uid":            {},
	"auth_time":      {},
	"firebase":       {},
	"email":          {},
	"email_verified": {},
	"name":           {},
	"displayName":    {},
	"picture":        {},
}

func extractClaims(claims map[string]interface{}) []components.ProfileClaim {
	if len(claims) == 0 {
		return nil
	}
	keys := make([]string, 0, len(claims))
	for k := range claims {
		if _, skip := reservedClaims[k]; skip {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]components.ProfileClaim, 0, len(keys))
	for _, k := range keys {
		result = append(result, components.ProfileClaim{Key: k, Value: claimValueString(claims[k])})
	}
	return result
}

func claimLookup(claims map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if claims == nil {
			break
		}
		if v, ok := claims[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func claimValueString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	case []string:
		return strings.Join(val, ", ")
	case []interface{}:
		return marshalValue(val)
	case map[string]interface{}:
		return marshalValue(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func marshalValue(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

func timeFromUnix(sec int64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}
