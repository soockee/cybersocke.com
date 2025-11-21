package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/sessions"

	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/internal/httpx"
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

type apiFunc func(w http.ResponseWriter, r *http.Request) error

type ApiError struct {
	Error string
}

// APIServer hosts all HTTP routes and their dependent services.
// It is constructed once and its services are reused across handlers.
type APIServer struct {
	embedStore   storage.Storage
	gcsStore     storage.Storage
	sessionStore *sessions.CookieStore
	cfg          *config.Config

	domainName string
	logger     *slog.Logger
	assets     embed.FS
	ctx        context.Context

	// Services (wired once in constructor; handlers reuse)
	authService  *services.AuthService
	postService  *services.PostService
	tagService   *services.TagService
	graphService *services.GraphService // optional; nil if backing store doesn't support graphs
}

// route represents a single endpoint registration.
// mw holds additional middlewares beyond the global stack.
// Middleware signature used locally for clarity.
type middlewareFunc func(http.Handler) http.Handler

// NewApiServer constructs the server and all required services.
// Returns an error instead of exiting so callers (main/tests) decide lifecycle.
// NewAPIServer constructs the server and all required services.
// Returns an error instead of exiting so callers (main/tests) decide lifecycle.
func NewAPIServer(embed storage.Storage, gcs storage.Storage, logger *slog.Logger, assets embed.FS, cfg *config.Config) (*APIServer, error) {
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   !cfg.LocalDev,
		MaxAge:   300,
	}
	server := &APIServer{
		embedStore:   embed,
		gcsStore:     gcs,
		sessionStore: store,
		cfg:          cfg,

		domainName: "cybersocke.com",
		logger:     logger,
		assets:     assets,
		ctx:        context.Background(),
	}
	// Wire core services
	authSvc, err := services.NewAuthService(server.ctx, cfg.FirebaseCredentialsBase64, cfg.GCPProjectName)
	if err != nil {
		return nil, err
	}
	tagSvc := services.NewTagService()
	postSvc := services.NewPostService(gcs, authSvc)
	server.authService = authSvc
	server.tagService = tagSvc
	server.postService = postSvc
	// Optional graph service (only if storage implements GraphBuilder)
	if gb, ok := gcs.(services.GraphBuilder); ok {
		server.graphService = services.NewGraphService(gb, tagSvc)
	}
	return server, nil
}

func (s *APIServer) Run() error {
	mux, err := s.InitRoutes()
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         ":8080",
		Handler:      mux,
		ErrorLog:     slog.NewLogLogger(s.logger.Handler(), slog.LevelDebug),
	}
	if err := httpServer.ListenAndServe(); err != nil {
		s.logger.Error("Failed to start HTTP server", slog.Any("err", err))
		return err
	}
	return nil
}

// InitRoutes assembles the router. No os.Exit side-effects; errors are propagated.
func (s *APIServer) InitRoutes() (*http.ServeMux, error) {
	if s.authService == nil || s.postService == nil || s.tagService == nil {
		return nil, errors.New("services not initialized")
	}
	mux := http.NewServeMux()

	// Helper to chain middlewares (shared)
	chain := func(h http.Handler, mws ...middlewareFunc) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}

	global := []middlewareFunc{
		middleware.WithLogging(s.logger),
		middleware.WithCORS(),
		middleware.WithSession(s.sessionStore, s.logger),
	}
	if s.cfg.LocalDev {
		global = append(global, middleware.WithDebugContext())
	}

	// Lightweight helper to register a pattern with handler + optional extra middleware.
	register := func(pattern string, h apiFunc, extra ...middlewareFunc) {
		stack := append(global, extra...)
		final := chain(makeHTTPHandleFunc(s.logger, h), stack...)
		mux.Handle(pattern, final)
	}

	// Explicit grouped registrations for readability.
	s.registerPublic(register)
	s.registerAPI(register)
	s.registerSecure(register)
	s.registerRole(register)

	return mux, nil
}

// registerPublic attaches all unauthenticated & public endpoints.
func (s *APIServer) registerPublic(register func(string, apiFunc, ...middlewareFunc)) {
	login := handlers.NewLoginHandler(s.logger)
	callback := handlers.NewAuthCallbackHandler(s.logger)
	post := handlers.NewPostHandler(s.postService, s.logger)
	home := handlers.NewHomeHandler(s.postService, s.tagService, s.logger)
	fragments := handlers.NewPostFragmentsHandler(s.postService, s.logger)
	tagPosts := handlers.NewTagPostsHandler(s.postService, s.logger)
	graph := handlers.NewGraphHandler(s.logger, s.graphService, s.postService)

	register("GET /auth", login.ServeHTTP)
	// Callback: GET for redirect completion; POST carries ID token JSON.
	register("GET /auth/google/callback", callback.ServeHTTP)
	register("POST /auth/google/callback", callback.ServeHTTP)
	// Assets subtree using wildcard capture.
	register("GET /assets/{rest...}", func(w http.ResponseWriter, r *http.Request) error {
		http.StripPrefix("/assets/", s.embedStore.GetAssets()).ServeHTTP(w, r)
		return nil
	})
	register("GET /posts/{id}", post.ServeHTTP)
	register("GET /posts/{id}/fragment", post.ServeHTTP)
	register("GET /", home.ServeHTTP)
	register("GET /posts/fragments", fragments.ServeHTTP)
	register("GET /tags/{tag}/posts", tagPosts.ServeHTTP)
	register("GET /graph", graph.ServeHTTP)
}

// apiRoutes returns JSON API endpoints (versionless initial design).
// registerAPI attaches JSON API endpoints.
func (s *APIServer) registerAPI(register func(string, apiFunc, ...middlewareFunc)) {
	if s.graphService != nil {
		register("GET /api/graph", handlers.NewGraphAPIHandler(s.logger, s.graphService).ServeHTTP)
	}
	register("GET /api/posts/{id}/adjacency", handlers.NewAdjacencyHandler(s.postService, s.tagService, s.logger).ServeHTTP)
}

// secureRoutes adds authenticated endpoints (CSRF protected).
func (s *APIServer) registerSecure(register func(string, apiFunc, ...middlewareFunc)) {
	secure := []middlewareFunc{
		middleware.WithCSRF(s.cfg.CSRFSecret, !s.cfg.LocalDev),
		middleware.WithAuthentication(s.authService, s.sessionStore, s.logger),
	}
	register("GET /admin", handlers.NewAdminHandler(s.postService, s.authService, s.logger).ServeHTTP, secure...)
}

// roleRoutes attaches role-gated write operations.
func (s *APIServer) registerRole(register func(string, apiFunc, ...middlewareFunc)) {
	secure := []middlewareFunc{
		middleware.WithCSRF(s.cfg.CSRFSecret, !s.cfg.LocalDev),
		middleware.WithAuthentication(s.authService, s.sessionStore, s.logger),
	}
	role := []middlewareFunc{
		middleware.WithRole("user", s.logger),
	}
	register("POST /posts", handlers.NewPostHandler(s.postService, s.logger).ServeHTTP, append(secure, role...)...)
}

func makeHTTPHandleFunc(logger *slog.Logger, f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			he := httpx.Classify(err)
			if logger != nil {
				// Log full details including cause if present.
				if he.Cause != nil {
					logger.Error("request error", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", he.Status), slog.Any("err", he.Cause))
				} else {
					logger.Error("request error", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", he.Status), slog.Any("err", err))
				}
			}
			w.WriteHeader(he.Status)
			_, _ = w.Write([]byte(he.Message))
		}
	}
}
