package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

const (
	GCP_OIDC_STS_ENDPOINT         = "https://sts.googleapis.com/v1/token"
	GCP_OIDC_CLOUD_PLATFORM_SCOPE = "https://www.googleapis.com/auth/cloud-platform"
	GCP_AUDIENCE_TEMPLATE         = "//iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s"
)

type GCSStore struct {
	bucket *storage.BucketHandle
	client *storage.Client

	gcpProjectId         string
	gcpWifPoolId         string
	gcpWifPoolProviderId string
	gcpAudience          string
}

type GcpStsAccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func NewGCSStore() (*GCSStore, error) {
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")
	gcpWifPoolId := os.Getenv("GCP_WIF_POOL_ID")
	gcpWifPoolProviderId := os.Getenv("GCP_WIF_POOL_PROVIDER_ID")
	if gcpProjectId == "" || gcpWifPoolId == "" || gcpWifPoolProviderId == "" {
		return nil, os.ErrInvalid
	}
	gcpAudience := fmt.Sprintf(
		GCP_AUDIENCE_TEMPLATE,
		gcpProjectId,
		gcpWifPoolId,
		gcpWifPoolProviderId,
	)

	return &GCSStore{
		bucket: nil,
		client: nil,

		gcpProjectId:         gcpProjectId,
		gcpWifPoolId:         gcpWifPoolId,
		gcpWifPoolProviderId: gcpWifPoolProviderId,
		gcpAudience:          gcpAudience,
	}, nil
}

func (s *GCSStore) GetPost(id string) Post {
	return Post{}
}

func (s *GCSStore) GetPosts() map[string]Post {
	return map[string]Post{}
}

func (s *GCSStore) GetAbout() []byte {
	return []byte{}
}

func (s *GCSStore) GetAssets() http.Handler {
	return nil
}

func (s *GCSStore) CreatePost(content []byte, ctx context.Context) error {
	client, err := s.NewStorageClient(ctx)
	if err != nil {
		return err
	}
	bucket := client.Bucket(os.Getenv("GCS_BUCKET"))
	obj := bucket.Object(uuid.New().String() + ".md").NewWriter(ctx)
	obj.ContentType = "text/markdown"
	if _, err := obj.Write(content); err != nil {
		obj.Close()
		return fmt.Errorf("write object: %w", err)
	}
	if err := obj.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}
	return nil
}

func (s *GCSStore) NewStorageClient(ctx context.Context) (*storage.Client, error) {
	rawID, ok := ctx.Value("session").(string)
	if !ok || rawID == "" {
		return nil, fmt.Errorf("missing session token")
	}
	stsTok, err := s.exchangeSTSToken(rawID, ctx)
	if err != nil {
		return nil, err
	}
	// base token source for STS access token
	baseTS := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: stsTok.AccessToken})
	// impersonate service account
	sa := os.Getenv("IMPERSONATE_STORAGE_SERVICE_ACCOUNT")
	impTS, err := ImpersonateServiceAccount(ctx, baseTS, sa, []string{storage.ScopeFullControl})
	if err != nil {
		return nil, fmt.Errorf("impersonation: %w", err)
	}
	// use impTS for storage client
	client, err := storage.NewClient(ctx, option.WithTokenSource(impTS))
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	return client, nil
}

func (s *GCSStore) exchangeSTSToken(idToken string, ctx context.Context) (*GcpStsAccessToken, error) {
	reqBody := map[string]string{
		"grantType":          "urn:ietf:params:oauth:grant-type:token-exchange",
		"audience":           s.gcpAudience,
		"scope":              GCP_OIDC_CLOUD_PLATFORM_SCOPE,
		"requestedTokenType": "urn:ietf:params:oauth:token-type:access_token",
		"subjectTokenType":   "urn:ietf:params:oauth:token-type:jwt",
		"subjectToken":       idToken,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal STS request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, GCP_OIDC_STS_ENDPOINT, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build STS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("execute STS request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("STS token exchange error: status %d, %s", resp.StatusCode, body)
	}

	var ststoken GcpStsAccessToken
	if err := json.NewDecoder(resp.Body).Decode(&ststoken); err != nil {
		return nil, fmt.Errorf("decode STS response: %w", err)
	}

	return &ststoken, nil
}

func ImpersonateServiceAccount(ctx context.Context, baseTS oauth2.TokenSource, target string, scopes []string) (oauth2.TokenSource, error) {
	cfg := impersonate.CredentialsConfig{
		TargetPrincipal: target,
		Scopes:          scopes,
		Delegates:       []string{},
	}
	return impersonate.CredentialsTokenSource(ctx, cfg, option.WithTokenSource(baseTS))
}
