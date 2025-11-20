package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	aboutService := services.NewAboutService(s.embedStore)

	rootRouter.Use(
		middleware.WithLogging(s.logger),
		// middleware.WithDebugContext(), // required for dev
		middleware.WithCORS(),
		middleware.WithSession(s.sessionStore),
	)

	// Unprotected routes
	rootRouter.HandleFunc("/auth", makeHTTPHandleFunc(handlers.NewLoginHandler(s.logger).ServeHTTP))
	rootRouter.HandleFunc("/auth/google/callback", makeHTTPHandleFunc(handlers.NewAuthCallbackHandler(s.logger).ServeHTTP))
	rootRouter.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", s.embedStore.GetAssets()))

	// Public GETs
	rootRouter.HandleFunc("/about", makeHTTPHandleFunc(handlers.NewAboutHandler(aboutService, s.logger).ServeHTTP))
	rootRouter.HandleFunc("/posts/{id}", makeHTTPHandleFunc(handlers.NewPostHandler(postService, s.logger).ServeHTTP)).Methods("GET")

	// Subrouter for CSRF-protected routes (e.g., writes from authenticated users)
	protected := rootRouter.PathPrefix("/").Subrouter()
	protected.Use(
		middleware.WithCSRF(s.cfg.CSRFSecret, !s.cfg.LocalDev),
	)
	protected.HandleFunc("/", makeHTTPHandleFunc(handlers.NewHomeHandler(postService, authService, s.logger).ServeHTTP))

	authenticated := protected.PathPrefix("/").Subrouter()
	authenticated.Use(
		middleware.WithAuthentication(authService, s.sessionStore),
	)
	authenticated.HandleFunc("/error", makeHTTPHandleFunc(handlers.NewErrorHandler(s.logger).ServeHTTP))
	// Writer-protected POST route for creating posts
	writer := authenticated.PathPrefix("/").Subrouter()
	writer.Use(middleware.WithRole("user"))
	writer.HandleFunc("/posts", makeHTTPHandleFunc(handlers.NewPostHandler(postService, s.logger).ServeHTTP)).Methods(http.MethodPost)

	return rootRouter
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := middleware.GetSession(r)
		s.Save(r, w)
		if err := f(w, r); err != nil {
			// Save error in session, flash, or temporary store
			msg := err.Error()
			errs := s.Values["errors"]
			if e, ok := errs.([]string); ok {
				s.Values["errors"] = append(e, time.Now().String()+msg)
			} else {
				s.Values["errors"] = []string{time.Now().String() + msg}
			}
			if err := s.Save(r, w); err != nil {
				fmt.Printf("Failed to save session: %v\n", err)
			}
		}
	}
}
