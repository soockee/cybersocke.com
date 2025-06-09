package middleware

import (
	"net/http"
	"strings"

	"github.com/gorilla/handlers"
)

func WithCORS() func(http.Handler) http.Handler {
	return handlers.CORS(
		handlers.AllowCredentials(),
		handlers.AllowedOriginValidator(
			func(origin string) bool {
				return strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://cybersocke.com")
			},
		),
		handlers.AllowedHeaders([]string{"X-CSRF-Token", "Content-Type", "Authorization", "Accept", "Origin", "Cache-Control", "X-Requested-With"}),
		handlers.ExposedHeaders([]string{"X-CSRF-Token"}),
	)
}
