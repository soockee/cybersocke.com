package impersonation

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
)

// ServiceAccountTokenSource returns an oauth2.TokenSource that impersonates the provided
// target service account using Application Default Credentials of the running environment.
// Scopes should be the minimal set required (e.g. identitytoolkit for Firebase Auth).
func ServiceAccountTokenSource(ctx context.Context, target string, scopes []string) (oauth2.TokenSource, error) {
	cfg := impersonate.CredentialsConfig{
		TargetPrincipal: target,
		Scopes:          scopes,
		Delegates:       []string{},
	}
	return impersonate.CredentialsTokenSource(ctx, cfg)
}
