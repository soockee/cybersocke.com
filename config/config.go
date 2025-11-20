package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config consolidates runtime environment configuration.
// All values are sourced from environment variables (12-factor style).
// Required fields must be present for the application to start.
type Config struct {
	SessionSecret             string
	CSRFSecret                string
	FirebaseCredentialsBase64 string
	FirebaseAPIKey            string
	FirebaseAuthDomain        string
	GCPProjectName            string
	GCSBucket                 string
	GCSCredentialsBase64      string
	Origin                    string
	Environment               string // e.g. production, staging, dev
	LocalDev                  bool
}

// Load loads configuration from environment variables using viper.
// It returns an error listing all missing required variables.
func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("LOCAL_DEV", false)

	cfg := &Config{
		SessionSecret:             v.GetString("SESSION_SECRET"),
		CSRFSecret:                v.GetString("CSRF_SECRET"),
		FirebaseCredentialsBase64: v.GetString("FIREBASE_CREDENTIALS_BASE64"),
		FirebaseAPIKey:            v.GetString("FIREBASE_INSENSITIVE_API_KEY"),
		FirebaseAuthDomain:        v.GetString("FIREBASE_AUTH_DOMAIN"),
		GCPProjectName:            v.GetString("GCP_PROJECT_NAME"),
		GCSBucket:                 v.GetString("GCS_BUCKET"),
		GCSCredentialsBase64:      v.GetString("GCS_CREDENTIALS_BASE64"),
		Origin:                    v.GetString("ORIGIN"),
		Environment:               v.GetString("ENVIRONMENT"),
		LocalDev:                  v.GetBool("LOCAL_DEV"),
	}

	missing := []string{}
	if cfg.SessionSecret == "" {
		missing = append(missing, "SESSION_SECRET")
	}
	if cfg.CSRFSecret == "" {
		missing = append(missing, "CSRF_SECRET")
	}
	// Firebase credentials are required unless impersonation target provided.
	if cfg.FirebaseCredentialsBase64 == "" {
		missing = append(missing, "FIREBASE_CREDENTIALS_BASE64")
	}
	if cfg.GCSBucket == "" {
		missing = append(missing, "GCS_BUCKET")
	}
	if cfg.GCSCredentialsBase64 == "" {
		missing = append(missing, "GCS_CREDENTIALS_BASE64")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}
