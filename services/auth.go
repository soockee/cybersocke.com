package services

import (
	"context"
	"encoding/base64"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type AuthService struct {
	App    *firebase.App
	Client *auth.Client
}

func NewAuthService(ctx context.Context, firebaseCredsBase64 string, projectID string) (*AuthService, error) {
	decoded, err := base64.StdEncoding.DecodeString(firebaseCredsBase64)
	if err != nil {
		return nil, err
	}
	config := &firebase.Config{}
	if projectID != "" {
		config.ProjectID = projectID
	}
	app, err := firebase.NewApp(ctx, config, option.WithCredentialsJSON(decoded))
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	return &AuthService{
		App:    app,
		Client: client,
	}, nil
}

func (s *AuthService) Verify(idToken string, ctx context.Context) (*auth.Token, error) {
	return s.Client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
}
