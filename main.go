package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("Starting...")
	logger.Info("Setup Storage...")
	store, err := NewSQLiteStore()
	if err != nil {
		slog.Any("err", err)
	}

	fs := http.FileServer(http.Dir("./assets"))

	server := NewApiServer(store, fs)
	server.Run(logger)
}
