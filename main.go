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

	ctx := context.Background()
	logger.Info("Setup GCS Storage...", slog.String("bucket", cfg.GCSBucket))
	gcsStore, err := storage.NewGCSStore(ctx, logger, cfg.GCSBucket, cfg.GCSCredentialsBase64)
	if err != nil {
		logger.Error("Failed to setup gcsStore", slog.Any("error msg", err))
		os.Exit(1)
	}
	// Contract already loaded remote-first inside storage constructor (attempted).
	if config.Contract != nil {
		logger.Info("Post contract ready", slog.String("hash", config.Contract.Hash()), slog.Int("version", config.Contract.Version))
	} else {
		logger.Warn("post contract unavailable after storage init")
	}

	logger.Info("Setup Embed Storage...")
	embedStore, err := storage.NewAssetsStore("assets/public", assets)
	if err != nil {
		logger.Error("Failed to setup embedStore", slog.Any("error msg", err))
		os.Exit(1)
	}

	server, err := NewAPIServer(embedStore, gcsStore, gcsStore, logger, assets, cfg)
	if err != nil {
		logger.Error("Failed to initialize server", slog.Any("err", err))
		os.Exit(1)
	}
	if err := server.Run(); err != nil {
		os.Exit(1)
	}
}
