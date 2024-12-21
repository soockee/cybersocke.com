package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/soockee/cybersocke.com/storage"
)

// Embed the assets directory into the binary
//
//go:embed assets/*
var assets embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("Starting...")
	logger.Info("Setup Storage...")
	store, err := storage.NewEmbedStore("assets/blog/content", assets)
	if err != nil {
		slog.Any("err", err)
	}

	server := NewApiServer(store, logger)
	server.Run()
}
