package htmlresp

import (
	"context"
	"io"
	"net/http"
)

// Renderer describes types (e.g., templ components) that can render into an io.Writer.
type Renderer interface {
	Render(context.Context, io.Writer) error
}

// Render writes HTML headers and status, then renders the component.
func Render(w http.ResponseWriter, status int, ctx context.Context, r Renderer) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	return r.Render(ctx, w)
}
