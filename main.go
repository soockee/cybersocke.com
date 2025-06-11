package main

import (
	"context"
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
	logger.Info("Setup Embed Storage...")
	embedStore, err := storage.NewEmbedStore("assets/content/blog", "assets/public", assets)
	if err != nil {
		logger.Error("Failed to setup embedStore", slog.Any("error msg", err))
		os.Exit(1)
	}
	ctx := context.Background()
	logger.Info("Setup GCS Storage...")
	gcsStore, err := storage.NewGCSStore(ctx)
	if err != nil {
		logger.Error("Failed to setup gcsStore", slog.Any("error msg", err))
		os.Exit(1)
	}

	server := NewApiServer(embedStore, gcsStore, logger, assets)
	server.Run()
}
