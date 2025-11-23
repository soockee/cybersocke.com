package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/storage"
	yaml "gopkg.in/yaml.v2"
)

// Simple migration CLI: loads contract migrations and applies them to all posts.
// Writes back any changed frontmatter (removing deprecated fields, updating schema_version).
// Usage: go run ./cmd/contract_migrate --bucket <bucket> --schema <path> [--dry]
func main() {
	bucket := flag.String("bucket", "", "GCS bucket name (required)")
	schemaPath := flag.String("schema", "", "Path to schema file (required)")
	dryRun := flag.Bool("dry", false, "Dry run: show planned changes without writing")
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *bucket == "" {
		fmt.Fprintln(os.Stderr, "--bucket is required")
		os.Exit(1)
	}
	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "--schema is required")
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := storage.NewGCSStoreADC(ctx, logger, *bucket)
	if err != nil {
		logger.Error("create store", "err", err)
		os.Exit(1)
	}
	previousVersion := 0
	previousHash := ""
	if config.Contract != nil {
		previousVersion = config.Contract.Version
		previousHash = config.Contract.Hash()
	}

	// Read and parse schema file
	schemaData, err := os.ReadFile(*schemaPath)
	if err != nil {
		logger.Error("read schema file", "path", *schemaPath, "err", err)
		os.Exit(1)
	}
	contract, err := config.ParsePostContract(schemaData)
	if err != nil {
		logger.Error("parse schema", "err", err)
		os.Exit(1)
	}
	config.Contract = contract
	logger.Info(
		"schema loaded",
		"path", *schemaPath,
		"version", contract.Version,
		"hash", contract.Hash(),
		"previous_version", previousVersion,
		"previous_hash", previousHash,
	)

	if previousVersion > 0 && previousHash != "" && previousVersion == contract.Version && previousHash == contract.Hash() {
		logger.Info("schema version and hash unchanged; skipping migrations")
		return
	}

	baselineVersion := previousVersion

	// Persist schema to bucket (canonical + versioned snapshot) if not dry run
	if !*dryRun {
		if err := persistSchema(ctx, store, contract, schemaData, logger); err != nil {
			logger.Warn("schema persistence failed", "err", err)
			// Non-fatal: continue with migrations even if schema upload fails
		}
	} else {
		logger.Info("dry run: skipping schema persistence")
	}

	posts, err := store.GetPosts(ctx)
	if err != nil {
		logger.Error("list posts", "err", err)
		os.Exit(1)
	}

	changed := 0
	for slug := range posts {
		raw, err := store.GetRaw(slug, ctx)
		if err != nil {
			logger.Warn("read raw", "slug", slug, "err", err)
			continue
		}
		updated, didChange, err := migrateRawFrontmatter(raw, baselineVersion)
		if err != nil {
			logger.Warn("migrate", "slug", slug, "err", err)
			continue
		}
		if !didChange {
			continue
		}
		changed++
		if *dryRun {
			logger.Info("would migrate", "slug", slug)
			continue
		}
		if err := store.UpdatePostSystem(slug, updated, ctx); err != nil {
			logger.Warn("write updated", "slug", slug, "err", err)
			continue
		}
		logger.Info("migrated", "slug", slug)
	}
	logger.Info("migration complete", "changed", changed, "total", len(posts), "dry", *dryRun)
	logger.Info("auth", "mode", "ADC (ambient credentials)")
}

// migrateRawFrontmatter parses raw markdown, applies migrations to frontmatter, returns new bytes & change flag.
// baselineVersion controls the minimum schema_version to treat documents as already migrated to.
func migrateRawFrontmatter(raw []byte, baselineVersion int) ([]byte, bool, error) {
	s := string(raw)
	if !strings.HasPrefix(s, "---") {
		return raw, false, nil
	}
	firstNL := strings.IndexByte(s, '\n')
	if firstNL < 0 {
		return raw, false, fmt.Errorf("malformed frontmatter")
	}
	fmStart := firstNL + 1
	cursor := fmStart
	fmEnd := -1
	bodyStart := -1
	for cursor <= len(s) {
		nextNL := strings.IndexByte(s[cursor:], '\n')
		lineEnd := len(s)
		line := ""
		if nextNL >= 0 {
			lineEnd = cursor + nextNL
			line = s[cursor:lineEnd]
		} else if cursor < len(s) {
			line = s[cursor:]
		}
		if line == "---" {
			fmEnd = cursor
			if nextNL >= 0 {
				bodyStart = lineEnd + 1 // skip newline after delimiter
			} else {
				bodyStart = lineEnd
			}
			break
		}
		if nextNL < 0 {
			break
		}
		cursor = lineEnd + 1
	}
	if fmEnd == -1 {
		return raw, false, fmt.Errorf("unterminated frontmatter")
	}
	fmBlock := s[fmStart:fmEnd]
	body := ""
	if bodyStart >= 0 && bodyStart <= len(s) {
		body = s[bodyStart:]
	}
	yamlMap := map[string]any{}
	if err := yaml.Unmarshal([]byte(fmBlock), &yamlMap); err != nil {
		return raw, false, fmt.Errorf("unmarshal frontmatter: %w", err)
	}
	beforeBytes, err := yaml.Marshal(yamlMap)
	if err != nil {
		return raw, false, fmt.Errorf("marshal frontmatter (before): %w", err)
	}
	if _, err := config.Contract.ApplyMigrationsFrom(yamlMap, baselineVersion); err != nil {
		return raw, false, err
	}
	afterBytes, err := yaml.Marshal(yamlMap)
	if err != nil {
		return raw, false, fmt.Errorf("marshal frontmatter (after): %w", err)
	}
	if string(beforeBytes) == string(afterBytes) {
		return raw, false, nil
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(afterBytes)
	b.WriteString("---\n")
	b.WriteString(body)
	return []byte(b.String()), true, nil
}

// persistSchema uploads the provided schema to GCS (canonical location).
// Compares with existing remote schema by hash to skip redundant writes.
func persistSchema(ctx context.Context, store *storage.GCSStore, contract *config.PostContract, schemaData []byte, logger *slog.Logger) error {
	// Check if remote schema already matches (compare hash)
	existingData, err := store.GetSchema(ctx)
	if err == nil {
		existingContract, parseErr := config.ParsePostContract(existingData)
		if parseErr == nil && existingContract.Hash() == contract.Hash() {
			logger.Info("schema unchanged, skipping persistence", "hash", contract.Hash())
			return nil
		}
	}
	// Upload canonical schema
	if err := store.PutSchema(ctx, schemaData); err != nil {
		return fmt.Errorf("put schema: %w", err)
	}
	logger.Info("schema persisted", "version", contract.Version, "hash", contract.Hash())
	return nil
}
