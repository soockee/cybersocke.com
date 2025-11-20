package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
)

type PostHandler struct {
	Log         *slog.Logger
	postService *services.PostService
}

func NewPostHandler(postService *services.PostService, log *slog.Logger) *PostHandler {
	return &PostHandler{
		Log:         log,
		postService: postService,
	}
}

func (h *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodPost:
		return h.Post(w, r)
	case http.MethodGet:
		return h.Get(w, r)
	default:
		return errors.New("method not allowed")
	}
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) error {
	idStr := mux.Vars(r)["id"]
	post, err := h.postService.GetPost(idStr, r.Context())
	if err != nil {
		return err
	}
	processed := stripDataview(post.Content)
	md := services.RenderMD(processed)
	h.View(w, r, components.PostViewProps{Content: md})
	return nil
}

func (h *PostHandler) Post(w http.ResponseWriter, r *http.Request) error {
	file, header, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Debug: print size only (avoid dumping content)
	fmt.Printf("upload received: name=%s size=%d bytes\n", header.Filename, len(content))

	// Sanitize original filename and ensure .md extension for slug derivation in storage layer
	original := filepath.Base(header.Filename)
	if err := h.postService.CreatePost(content, original, r.Context()); err != nil {
		return err
	}
	return nil
}

func (h *PostHandler) View(w http.ResponseWriter, r *http.Request, props components.PostViewProps) {
	components.Post(props).Render(r.Context(), w)
}

// stripDataview removes or comments out Obsidian dataview / callout constructs so they don't render.
// Rules:
// 1. Lines starting with "> [!" (callouts like > [!Summary]) are removed.
// 2. Inline code segments of the form `= this.<field>` are stripped entirely.
// 3. The rest of the content is returned unchanged.
func stripDataview(content []byte) []byte {
	lines := bytes.Split(content, []byte("\n"))
	for i, ln := range lines {
		trimmed := bytes.TrimSpace(ln)
		if bytes.HasPrefix(trimmed, []byte("> [!")) {
			// Remove callout line
			lines[i] = []byte("")
			continue
		}
		// Remove inline dataview expressions enclosed in backticks
		for bytes.Contains(lines[i], []byte("`= this.")) {
			start := bytes.Index(lines[i], []byte("`= this."))
			if start == -1 {
				break
			}
			rest := lines[i][start+1:] // skip initial backtick
			end := bytes.IndexByte(rest, '`')
			if end == -1 {
				// No closing backtick: remove from start to end of line
				lines[i] = lines[i][:start]
				break
			}
			// Remove the whole `= this....` segment including both backticks
			lines[i] = append(lines[i][:start], rest[end+1:]...)
		}
	}
	return bytes.Join(lines, []byte("\n"))
}
