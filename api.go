package main

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/handlers"
	"github.com/soockee/cybersocke.com/middleware"
	"github.com/soockee/cybersocke.com/services"
	"github.com/soockee/cybersocke.com/storage"
)

type apiFunc func(w http.ResponseWriter, r *http.Request) error

type ApiError struct {
	Error string
}

type ApiServer struct {
	embedStore   storage.Storage
	gcsStore     storage.Storage
	sessionStore *sessions.CookieStore
	cfg          *config.Config

	domainName string
	logger     *slog.Logger
	assets     embed.FS
	ctx        context.Context
}

func NewApiServer(embed storage.Storage, gcs storage.Storage, logger *slog.Logger, assets embed.FS, cfg *config.Config) *ApiServer {
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   !cfg.LocalDev,
		MaxAge:   300,
	}
	server := &ApiServer{
		embedStore:   embed,
		gcsStore:     gcs,
		sessionStore: store,
		cfg:          cfg,

		domainName: "cybersocke.com",
		logger:     logger,
		assets:     assets,
		ctx:        context.Background(),
	}
	return server
}

func (s *ApiServer) Run() {
	router := s.InitRoutes()

	httpServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         ":8080",
		Handler:      router,
		ErrorLog:     slog.NewLogLogger(s.logger.Handler(), slog.LevelDebug),
	}

	if err := httpServer.ListenAndServe(); err != nil {
		s.logger.Error("Failed to start HTTP server", slog.Any("err", err))
		os.Exit(1)
	}
}

func (s *ApiServer) InitRoutes() *mux.Router {
	rootRouter := mux.NewRouter()

	authService, err := services.NewAuthService(s.ctx, s.cfg.FirebaseCredentialsBase64, s.cfg.GCPProjectName)
	if err != nil {
		s.logger.Error("Failed to initialize AuthService", slog.Any("err", err))
		os.Exit(1)
	}
	postService := services.NewPostService(s.gcsStore, authService)
	tagService := services.NewTagService()

	// Attempt concrete GCS store assertion for graph features
	gcsConcrete, _ := s.gcsStore.(*storage.GCSStore)

	rootRouter.Use(
		middleware.WithLogging(s.logger),
		middleware.WithDebugContext(), // required for dev
		middleware.WithCORS(),
		middleware.WithSession(s.sessionStore),
	)

	// Unprotected routes
	rootRouter.HandleFunc("/auth", makeHTTPHandleFunc(s.logger, handlers.NewLoginHandler(s.logger).ServeHTTP))
	rootRouter.HandleFunc("/auth/google/callback", makeHTTPHandleFunc(s.logger, handlers.NewAuthCallbackHandler(s.logger).ServeHTTP))
	rootRouter.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", s.embedStore.GetAssets()))

	// Public GETs
	rootRouter.HandleFunc("/posts/{id}", makeHTTPHandleFunc(s.logger, handlers.NewPostHandler(postService, s.logger).ServeHTTP)).Methods("GET")
	rootRouter.HandleFunc("/posts/{id}/fragment", makeHTTPHandleFunc(s.logger, handlers.NewPostHandler(postService, s.logger).ServeHTTP)).Methods("GET")
	rootRouter.HandleFunc("/", makeHTTPHandleFunc(s.logger, handlers.NewHomeHandler(postService, tagService, s.logger).ServeHTTP))
	// Adjacency JSON (neighbors with shared tags)
	rootRouter.HandleFunc("/posts/{id}/adjacency", makeHTTPHandleFunc(s.logger, handlers.NewAdjacencyHandler(postService, tagService, s.logger).ServeHTTP)).Methods("GET")
	// Batch post fragments endpoint (renamed from /theme/fragments)
	rootRouter.HandleFunc("/posts/fragments", makeHTTPHandleFunc(s.logger, handlers.NewPostFragmentsHandler(postService, s.logger).ServeHTTP)).Methods("GET")
	// Tag-specific fragments (posts filtered by single tag)
	rootRouter.HandleFunc("/tags/{tag}/posts", makeHTTPHandleFunc(s.logger, handlers.NewTagPostsHandler(postService, s.logger).ServeHTTP)).Methods("GET")
	if gcsConcrete != nil {
		graphService := services.NewGraphService(gcsConcrete, tagService)
		rootRouter.HandleFunc("/graph", makeHTTPHandleFunc(s.logger, handlers.NewGraphHandler(s.logger, graphService, postService).ServeHTTP)).Methods("GET")
		rootRouter.HandleFunc("/graph.json", makeHTTPHandleFunc(s.logger, handlers.NewGraphHandler(s.logger, graphService, postService).ServeHTTP)).Methods("GET")
	}

	// Subrouter for CSRF-protected routes (e.g., writes from authenticated users)
	secure := rootRouter.PathPrefix("/").Subrouter()
	secure.Use(
		middleware.WithCSRF(s.cfg.CSRFSecret, !s.cfg.LocalDev),
		middleware.WithAuthentication(authService, s.sessionStore),
	)
	secure.HandleFunc("/error", makeHTTPHandleFunc(s.logger, handlers.NewErrorHandler(s.logger).ServeHTTP))
	// Admin legacy navigator (protected)
	secure.HandleFunc("/admin", makeHTTPHandleFunc(s.logger, handlers.NewAdminHandler(postService, authService, s.logger).ServeHTTP))

	// roler-protected POST route for creating posts
	role := secure.PathPrefix("/").Subrouter()
	role.Use(middleware.WithRole("user"))
	role.HandleFunc("/posts", makeHTTPHandleFunc(s.logger, handlers.NewPostHandler(postService, s.logger).ServeHTTP)).Methods(http.MethodPost)

	return rootRouter
}

func makeHTTPHandleFunc(logger *slog.Logger, f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Maintain session (e.g., for auth / csrf) but do not expose error details to client
		s := middleware.GetSession(r)
		_ = s.Save(r, w) // ignore error
		if err := f(w, r); err != nil {
			msg := err.Error()
			// Classify status without leaking internal message
			status := http.StatusBadRequest
			if strings.Contains(msg, "unauthorized") {
				status = http.StatusUnauthorized
			} else if strings.Contains(msg, "write object") || strings.Contains(msg, "close writer") {
				status = http.StatusInternalServerError
			}
			// Log full error server-side
			if logger != nil {
				logger.Error("request error", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", status), slog.Any("err", err))
			}
			// Standard status response only; optional minimal body text
			w.WriteHeader(status)
			// Provide a generic message for non-HTML clients (no internal details)
			switch status {
			case http.StatusUnauthorized:
				_, _ = w.Write([]byte("unauthorized"))
			case http.StatusInternalServerError:
				_, _ = w.Write([]byte("internal server error"))
			default:
				_, _ = w.Write([]byte("bad request"))
			}
		}
	}
}
