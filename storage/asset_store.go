package storage

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// AssetsStore provides read-only access to embedded static assets.
// It no longer implements full post storage; dynamic content operations are
// handled by the primary GCS-backed store. Only asset serving and a lightweight
// About page helper remain. All mutation or post-centric methods now return
// explicit errors to signal the read-only nature.
type AssetsStore struct {
	assets embed.FS
	fs     http.Handler
}

func NewAssetsStore(publicDir string, assets embed.FS) (*AssetsStore, error) {
	public, err := fs.Sub(assets, publicDir)
	if err != nil {
		return nil, err
	}
	baseFS := http.FileServer(http.FS(public))
	// Wrap to enforce correct JS MIME type (some environments default to text/plain)
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		// normalize path extension
		ext := strings.ToLower(path.Ext(p))
		if ext == ".js" {
			// Always set explicit JS MIME to avoid strict MIME rejection
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		baseFS.ServeHTTP(w, r)
	})

	return &AssetsStore{assets: assets, fs: wrapped}, nil
}

func (s *AssetsStore) GetAssets() http.Handler {
	return s.fs
}
