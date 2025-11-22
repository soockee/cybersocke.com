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
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/soockee/cybersocke.com/docs"

	htmlhandler "github.com/soockee/cybersocke.com/handlers/htmlhandler"
	jsonhandler "github.com/soockee/cybersocke.com/handlers/jsonhandler"
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

// APIServer hosts all HTTP routes and their dependent services.
// It is constructed once and its services are reused across handlers.
// @title Cybersocke API
// @version 1.0
// @description HTTP JSON API for cybersocke.com posts, tags, and graphs.
// @host cybersocke.com
// @BasePath /api
// @schemes https http
type APIServer struct {
	assetStore   storage.AssetStore
	contentStore storage.ContentStore
	queryStore   storage.PostQueryStore
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
	graphService *services.GraphService // optional; nil if backing store supports graphs
}

// route represents a single endpoint registration.
// mw holds additional middlewares beyond the global stack.
// Middleware signature used locally for clarity.
type middlewareFunc func(http.Handler) http.Handler

// NewApiServer constructs the server and all required services.
// Returns an error instead of exiting so callers (main/tests) decide lifecycle.
// NewAPIServer constructs the server and all required services.
// Returns an error instead of exiting so callers (main/tests) decide lifecycle.
func NewAPIServer(asset storage.AssetStore, content storage.ContentStore, query storage.PostQueryStore, logger *slog.Logger, assets embed.FS, cfg *config.Config) (*APIServer, error) {
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   !cfg.LocalDev,
		MaxAge:   300,
	}
	server := &APIServer{
		assetStore:   asset,
		contentStore: content,
		queryStore:   query,
		sessionStore: store,
		cfg:          cfg,
		domainName:   "cybersocke.com",
		logger:       logger,
		assets:       assets,
		ctx:          context.Background(),
	}
	// Wire core services
	authSvc, err := services.NewAuthService(server.ctx, cfg.FirebaseCredentialsBase64, cfg.GCPProjectName)
	if err != nil {
		return nil, err
	}
	tagSvc := services.NewTagService()
	postSvc := services.NewPostService(content, query, authSvc)
	server.authService = authSvc
	server.tagService = tagSvc
	server.postService = postSvc
	// Optional graph service (only if storage implements GraphBuilder)
	if gb, ok := query.(services.GraphBuilder); ok {
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
	global = append(global, middleware.WithOptionalAuthentication(s.authService, s.logger))
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
	login := htmlhandler.NewLoginHandler(s.logger)
	callback := htmlhandler.NewAuthCallbackHandler(s.logger)
	logout := htmlhandler.NewLogoutHandler(s.logger)
	post := htmlhandler.NewPostHandler(s.postService, s.logger)
	fragments := htmlhandler.NewPostFragmentsHandler(s.postService, s.logger)
	tagPosts := htmlhandler.NewTagPostsHandler(s.postService, s.logger)
	graph := htmlhandler.NewGraphHandler(s.logger, s.graphService, s.postService)
	postsGrid := htmlhandler.NewPostsGridHandler(s.postService, s.logger)

	register("GET /auth", login)
	register("GET /auth/logout", logout)
	// Callback: GET for redirect completion; POST carries ID token JSON.
	register("GET /auth/google/callback", callback)
	register("POST /auth/google/callback", callback)
	// Assets subtree using wildcard capture.
	register("GET /assets/{rest...}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/assets/", s.assetStore.GetAssets()).ServeHTTP(w, r)
	}))
	register("GET /posts/{id}", post)
	register("GET /posts/{id}/fragment", post)
	register("GET /posts", postsGrid)
	register("GET /", htmlhandler.NewRootRedirectHandler("/posts", s.logger))
	register("GET /posts/fragments", fragments)
	register("GET /tags/{tag}/posts", tagPosts)
	register("GET /graph", graph)
}

// registerAPI attaches JSON API endpoints.
func (s *APIServer) registerAPI(register func(string, http.Handler, ...middlewareFunc)) {
	if s.graphService != nil {
		register("GET /api/graph", jsonhandler.NewGraphAPIHandler(s.logger, s.graphService))
	}
	register("GET /api/posts/{id}/adjacency", jsonhandler.NewAdjacencyHandler(s.postService, s.tagService, s.logger))
	register("GET /api/posts", jsonhandler.NewPostsAPIHandler(s.postService, s.logger))
}

// secureRoutes adds authenticated endpoints (CSRF protected).
func (s *APIServer) registerSecure(register func(string, http.Handler, ...middlewareFunc)) {
	secure := []middlewareFunc{
		middleware.WithCSRF(s.cfg.CSRFSecret, !s.cfg.LocalDev),
		middleware.WithAuthentication(s.authService, s.sessionStore, s.logger),
	}
	register("GET /swagger/{rest...}", httpSwagger.WrapHandler, secure...)
	register("GET /admin", htmlhandler.NewAdminHandler(s.postService, s.authService, s.logger), secure...)
	register("GET /admin/posts/{slug}/edit", htmlhandler.NewAdminEditHandler(s.postService, s.logger), secure...)
	register("POST /admin/posts/{slug}/edit", htmlhandler.NewAdminEditHandler(s.postService, s.logger), secure...)
	register("GET /profile", htmlhandler.NewProfileHandler(s.logger), secure...)
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
	register("POST /posts", jsonhandler.NewPostUploadHandler(s.postService, s.logger), append(secure, role...)...)
	register("POST /admin/posts/{slug}/delete", htmlhandler.NewAdminDeleteHandler(s.postService, s.logger), append(secure, role...)...)
}

// makeHTTPHandleFunc removed; handlers now implement http.Handler directly with internal error handling.
