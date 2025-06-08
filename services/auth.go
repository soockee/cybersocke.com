package services

import (
	"context"
	"encoding/base64"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type AuthService struct {
	App    *firebase.App
	Client *auth.Client
}

func NewAuthService(ctx context.Context) (*AuthService, error) {
	encoded := os.Getenv("FIREBASE_CREDENTIALS_BASE64")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	config := &firebase.Config{
		ProjectID: "dz-cybersocke01",
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
	return s.Client.VerifyIDToken(ctx, idToken)
}
