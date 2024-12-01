package main

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/soockee/cybersocke.com/components"
	"github.com/soockee/cybersocke.com/handlers"
)

type apiFunc func(w http.ResponseWriter, r *http.Request) error

type ApiError struct {
	Error string
}

type ApiServer struct {
	store      Storage
	fs         http.Handler
	domainName string
}

func NewApiServer(store Storage, fs http.Handler) *ApiServer {
	server := &ApiServer{
		store: store,
		fs:    fs,
	}

	server.domainName = "stockhause.info"
	return server
}

func (s *ApiServer) InitRoutes() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", makeHTTPHandleFunc(handlers.NewHomeHandler(slog.Default()).ServeHTTP))
	router.HandleFunc("/posts/{id}", makeHTTPHandleFunc(handlers.NewPostHandler(slog.Default()).ServeHTTP))
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", s.fs))

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
