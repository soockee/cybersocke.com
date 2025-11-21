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
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

// APIError retained for potential future structured error responses.
type APIError struct{ Error string }

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
	register := func(pattern string, h http.Handler, extra ...middlewareFunc) {
		stack := append(global, extra...)
		final := chain(h, stack...)
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
func (s *APIServer) registerPublic(register func(string, http.Handler, ...middlewareFunc)) {
	login := handlers.NewLoginHandler(s.logger)
	callback := handlers.NewAuthCallbackHandler(s.logger)
	post := handlers.NewPostHandler(s.postService, s.logger)
	home := handlers.NewHomeHandler(s.postService, s.tagService, s.logger)
	fragments := handlers.NewPostFragmentsHandler(s.postService, s.logger)
	tagPosts := handlers.NewTagPostsHandler(s.postService, s.logger)
	graph := handlers.NewGraphHandler(s.logger, s.graphService, s.postService)

	register("GET /auth", login)
	// Callback: GET for redirect completion; POST carries ID token JSON.
	register("GET /auth/google/callback", callback)
	register("POST /auth/google/callback", callback)
	// Assets subtree using wildcard capture.
	register("GET /assets/{rest...}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/assets/", s.embedStore.GetAssets()).ServeHTTP(w, r)
	}))
	register("GET /posts/{id}", post)
	register("GET /posts/{id}/fragment", post)
	register("GET /", home)
	register("GET /posts/fragments", fragments)
	register("GET /tags/{tag}/posts", tagPosts)
	register("GET /graph", graph)
}

// apiRoutes returns JSON API endpoints (versionless initial design).
// registerAPI attaches JSON API endpoints.
func (s *APIServer) registerAPI(register func(string, http.Handler, ...middlewareFunc)) {
	if s.graphService != nil {
		register("GET /api/graph", handlers.NewGraphAPIHandler(s.logger, s.graphService))
	}
	register("GET /api/posts/{id}/adjacency", handlers.NewAdjacencyHandler(s.postService, s.tagService, s.logger))
}

// secureRoutes adds authenticated endpoints (CSRF protected).
func (s *APIServer) registerSecure(register func(string, http.Handler, ...middlewareFunc)) {
	secure := []middlewareFunc{
		middleware.WithCSRF(s.cfg.CSRFSecret, !s.cfg.LocalDev),
		middleware.WithAuthentication(s.authService, s.sessionStore, s.logger),
	}
	register("GET /admin", handlers.NewAdminHandler(s.postService, s.authService, s.logger), secure...)
}

// roleRoutes attaches role-gated write operations.
func (s *APIServer) registerRole(register func(string, http.Handler, ...middlewareFunc)) {
	secure := []middlewareFunc{
		middleware.WithCSRF(s.cfg.CSRFSecret, !s.cfg.LocalDev),
		middleware.WithAuthentication(s.authService, s.sessionStore, s.logger),
	}
	role := []middlewareFunc{
		middleware.WithRole("user", s.logger),
	}
	register("POST /posts", handlers.NewPostHandler(s.postService, s.logger), append(secure, role...)...)
}

// makeHTTPHandleFunc removed; handlers now implement http.Handler directly with internal error handling.
