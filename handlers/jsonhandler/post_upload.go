package jsonhandler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/respond/jsonresp"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage/validation"
)

// PostUploadHandler handles authenticated post uploads and responds with JSON metadata.
type PostUploadHandler struct {
	Log         *slog.Logger
	postService *services.PostService
}

func NewPostUploadHandler(postService *services.PostService, log *slog.Logger) *PostUploadHandler {
	return &PostUploadHandler{
		Log:         log,
		postService: postService,
	}
}

func (h *PostUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handlers.WriteHTTPError(w, r, h.Log, handlers.ErrMethodNotAllowed)
		return
	}

	if err := h.upload(w, r); err != nil {
		handlers.WriteHTTPError(w, r, h.Log, err)
	}
}

func (h *PostUploadHandler) upload(w http.ResponseWriter, r *http.Request) error {
	start := time.Now()
	logger := h.Log.With(
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
	)

	file, header, err := r.FormFile("file")
	if err != nil {
		logger.Info("upload form file missing", slog.Any("err", err))
		return handlers.BadRequest("invalid upload", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		logger.Info("upload read failed", slog.Any("err", err))
		return handlers.BadRequest("failed to read file", err)
	}
	if len(content) == 0 {
		logger.Info("upload empty content")
		return handlers.BadRequest("empty file", errors.New("empty file"))
	}

	original := filepath.Base(header.Filename)
	slug := validation.SanitizeFilename(original)
	logger.Debug("upload parsed", slog.String("filename", original), slog.Int("size", len(content)), slog.String("slug", slug))

	if err := h.postService.CreatePost(content, original, r.Context()); err != nil {
		logger.Error("create post failed", slog.String("slug", slug), slog.Any("err", err))
		return err
	}

	logger.Info("upload stored", slog.String("slug", slug), slog.Duration("took", time.Since(start)))

	return jsonresp.Write(w, http.StatusCreated, struct {
		Slug string `json:"slug"`
	}{Slug: slug})
}
