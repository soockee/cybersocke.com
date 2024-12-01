package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gorilla/mux"

	"github.com/soockee/cybersocke.com/components"
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
	router.HandleFunc("/", makeHTTPHandleFunc(s.handleHome))
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", s.fs))

	return router
}

func WriteJson(w http.ResponseWriter, status int, v any) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return cors(func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJson(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	})
}

func (s *ApiServer) handleHome(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		component := components.Home()
		handler := templ.Handler(component)
		handler.ServeHTTP(w, r)
		return nil
	case "POST":
		WriteJson(w, http.StatusNotImplemented, "")
	default:
		return errors.New("method not allowed")
	}
	return nil
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
