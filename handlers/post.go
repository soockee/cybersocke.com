package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
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
	processed := applySimpleDataview(post.Meta, post.Content)
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

// hasWriterRole determines if the token has permission to write posts.
// Accepts role patterns: role==writer/admin, roles slice contains writer/admin, writer=true or admin=true.
// Role enforcement moved to middleware.WithRole("writer") for POST /posts.

// applySimpleDataview performs minimal inline dataview substitution:
// Replaces occurrences of `= this.<field>` inside backticks with the frontmatter value.
// Only supports direct field lookup; no expressions.
func applySimpleDataview(meta storage.PostMeta, content []byte) []byte {
	lines := bytes.Split(content, []byte("\n"))
	replacer := func(expr string) string {
		field := strings.TrimPrefix(expr, "= this.")
		field = strings.TrimSpace(field)
		switch field {
		case "lead":
			return meta.Lead
		case "description":
			return meta.Description
		case "license":
			return meta.License
		case "template_type":
			return meta.TemplateType
		case "template_version":
			return meta.TemplateVersion
		default:
			return "" // unknown field -> empty
		}
	}
	for i, ln := range lines {
		if bytes.Contains(ln, []byte("`= this.")) {
			segments := bytes.Split(ln, []byte("`"))
			for j := 0; j < len(segments); j++ {
				seg := string(segments[j])
				if strings.HasPrefix(seg, "= this.") {
					val := replacer(seg)
					segments[j] = []byte(val)
				}
			}
			lines[i] = bytes.Join(segments, []byte("`"))
		}
	}
	return bytes.Join(lines, []byte("\n"))
}
