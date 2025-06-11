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

	domainName string
	logger     *slog.Logger
	assets     embed.FS
	ctx        context.Context
}

func NewApiServer(embed storage.Storage, gcs storage.Storage, logger *slog.Logger, assets embed.FS) *ApiServer {
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // false for local dev
		MaxAge:   300,
	}
	server := &ApiServer{
		embedStore:   embed,
		gcsStore:     gcs,
		sessionStore: store,

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

	authService, err := services.NewAuthService(s.ctx)
	if err != nil {
		s.logger.Error("Failed to initialize AuthService", slog.Any("err", err))
		os.Exit(1)
	}
	postService := services.NewPostService(s.gcsStore, authService)
	aboutService := services.NewAboutService(s.embedStore)
	csfrService := services.NewCSFRService(s.ctx)

	rootRouter.Use(
		middleware.WithLogging(s.logger),
		middleware.WithDebugContext(),
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
		middleware.WithCSRF(),
	)
	protected.HandleFunc("/", makeHTTPHandleFunc(handlers.NewHomeHandler(postService, authService, csfrService, s.logger).ServeHTTP))

	authenticated := protected.PathPrefix("/").Subrouter()
	authenticated.Use(
		middleware.WithAuthentication(authService, s.sessionStore),
	)
	authenticated.HandleFunc("/error", makeHTTPHandleFunc(handlers.NewErrorHandler(s.logger).ServeHTTP))
	authenticated.HandleFunc("/posts", makeHTTPHandleFunc(handlers.NewPostHandler(postService, s.logger).ServeHTTP))

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
