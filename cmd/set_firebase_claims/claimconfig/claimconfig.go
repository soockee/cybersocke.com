package claimconfig

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds configuration specific to the set_firebase_claims utility.
// It is intentionally separate from the main server config to keep concerns isolated.
// Impersonation-only: requires a target service account email and explicit project ID.
type Config struct {
	FirebaseImpersonateServiceAccount string // required: target service account email to impersonate (caller needs roles/iam.serviceAccountTokenCreator)
	GCPProjectID                      string // required: Firebase/GCP project ID
}

func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	cfg := &Config{
		FirebaseImpersonateServiceAccount: v.GetString("FIREBASE_IMPERSONATE_SERVICE_ACCOUNT"),
		GCPProjectID:                      v.GetString("GCP_PROJECT_ID"),
	}

	if cfg.FirebaseImpersonateServiceAccount == "" {
		return nil, fmt.Errorf("missing required claim utility config: FIREBASE_IMPERSONATE_SERVICE_ACCOUNT")
	}
	if cfg.GCPProjectID == "" {
		return nil, fmt.Errorf("missing required claim utility config: GCP_PROJECT_ID")
	}
	return cfg, nil
}
