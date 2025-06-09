package main

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"github.com/soockee/cybersocke.com/components"
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
	store      storage.Storage
	domainName string
	logger     *slog.Logger
	assets     embed.FS
	ctx        context.Context
}

func NewApiServer(store storage.Storage, logger *slog.Logger, assets embed.FS) *ApiServer {
	server := &ApiServer{
		store:      store,
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

	postService := services.NewPostService(s.store)
	aboutService := services.NewAboutService(s.store)
	csfrService := services.NewCSFRService(s.ctx)
	authService, err := services.NewAuthService(s.ctx)
	if err != nil {
		s.logger.Error("Failed to initialize AuthService", slog.Any("err", err))
		os.Exit(1)
	}

	rootRouter.Use(
		middleware.WithLogging(s.logger),
		middleware.WithDebugContext(),
		middleware.WithCORS(),
	)

	// Unprotected routes
	rootRouter.HandleFunc("/auth", makeHTTPHandleFunc(handlers.NewLoginHandler(s.logger).ServeHTTP))
	rootRouter.HandleFunc("/auth/google/callback", makeHTTPHandleFunc(handlers.NewAuthCallbackHandler(s.logger).ServeHTTP))
	rootRouter.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", s.store.GetFS()))

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
		middleware.WithAuthentication(authService),
	)
	authenticated.HandleFunc("/posts", makeHTTPHandleFunc(handlers.NewPostHandler(postService, s.logger).ServeHTTP)).Methods("POST")

	return rootRouter
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			components.Error(err.Error()).Render(r.Context(), w)
		}
	}
}

func authenticate(next http.HandlerFunc, authService *services.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get session cookie
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Verify ID token
		token, err := authService.Verify(cookie.Value, r.Context())
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user", token)
		next(w, r.WithContext(ctx))
	}
}

// func cors(next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Add("Access-Control-Allow-Origin", "*")
// 		w.Header().Add("Access-Control-Allow-Credentials", "true")
// 		w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
// 		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

// 		next(w, r)
// 	}
// }

// func CSFR(next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		token := csrf.Token(r)
// 		w.Header().Set("X-CSRF-Token", token)

// 		next(w, r)
// 	}
// }
