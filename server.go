package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/soockee/cybersocke.com/middleware"
)

func (s *ApiServer) Run(logger *slog.Logger) {
	loggingMiddleware := middleware.WithLogging(logger)
	sessionMiddleware := middleware.WithSession(logger, true, true)
	r := s.InitRoutes()
	router := sessionMiddleware(loggingMiddleware(r))

	httpServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         ":http",
		Handler:      router,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelDebug),
	}

	if err := httpServer.ListenAndServe(); err != nil {
		logger.Error("Failed to start HTTP server", slog.Any("err", err))
		os.Exit(1)
	}
}
