package main

import (
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
}

func NewApiServer(store storage.Storage, logger *slog.Logger) *ApiServer {
	server := &ApiServer{
		store:      store,
		domainName: "cybersocke.com",
		logger:     logger,
	}
	return server
}

func (s *ApiServer) Run() {
	loggingMiddleware := middleware.WithLogging(s.logger)
	sessionMiddleware := middleware.WithSession(s.logger, true, true)
	r := s.InitRoutes()
	router := sessionMiddleware(loggingMiddleware(r))

	httpServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         ":http",
		Handler:      router,
		ErrorLog:     slog.NewLogLogger(s.logger.Handler(), slog.LevelDebug),
	}

	if err := httpServer.ListenAndServe(); err != nil {
		s.logger.Error("Failed to start HTTP server", slog.Any("err", err))
		os.Exit(1)
	}
}

func (s *ApiServer) InitRoutes() *mux.Router {
	router := mux.NewRouter()

	postService := services.NewPostService(s.store)

	homeHandler := handlers.NewHomeHandler(postService, s.logger)
	router.HandleFunc("/", makeHTTPHandleFunc(homeHandler.ServeHTTP))

	postHandler := handlers.NewPostHandler(postService, s.logger)
	router.HandleFunc("/posts/{id}", makeHTTPHandleFunc(postHandler.ServeHTTP))

	return router
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return cors(func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			components.Error(err.Error()).Render(r.Context(), w)
		}
	})
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		next(w, r)
	}
}
