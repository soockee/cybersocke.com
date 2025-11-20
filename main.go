package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/storage"
)

// Embed the assets directory into the binary
//
//go:embed assets/*
var assets embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logger.Info("Starting...")
	logger.Info("Load configuration...")
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Setup Embed Storage...")
	embedStore, err := storage.NewEmbedStore("assets/content/blog", "assets/public", assets)
	if err != nil {
		logger.Error("Failed to setup embedStore", slog.Any("error msg", err))
		os.Exit(1)
	}
	ctx := context.Background()
	logger.Info("Setup GCS Storage...", slog.String("bucket", cfg.GCSBucket))
	gcsStore, err := storage.NewGCSStore(ctx, logger, cfg.GCSBucket, cfg.GCSCredentialsBase64)
	if err != nil {
		logger.Error("Failed to setup gcsStore", slog.Any("error msg", err))
		os.Exit(1)
	}

	server := NewApiServer(embedStore, gcsStore, logger, assets, cfg)
	server.Run()
}
