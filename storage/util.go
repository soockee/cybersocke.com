package storage

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

func ImpersonateServiceAccount(ctx context.Context, baseTS oauth2.TokenSource, target string, scopes []string) (oauth2.TokenSource, error) {
	cfg := impersonate.CredentialsConfig{
		TargetPrincipal: target,
		Scopes:          scopes,
		Delegates:       []string{},
	}
	return impersonate.CredentialsTokenSource(ctx, cfg, option.WithTokenSource(baseTS))
}
