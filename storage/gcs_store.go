package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/allegro/bigcache/v3"
	"github.com/gorilla/sessions"
	"github.com/soockee/cybersocke.com/parser/frontmatter"
	"github.com/soockee/cybersocke.com/session"
	"golang.org/x/oauth2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	GCP_OIDC_STS_ENDPOINT         = "https://sts.googleapis.com/v1/token"
	GCP_OIDC_CLOUD_PLATFORM_SCOPE = "https://www.googleapis.com/auth/cloud-platform"
	GCP_AUDIENCE_TEMPLATE         = "//iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s"
)

type GCSStore struct {
	bucketName string
	clientRW   *http.Client
	clientRO   *storage.Client
	logger     *slog.Logger

	cache *bigcache.BigCache

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

type GcpTokenRequest struct {
	GrantType          string `json:"grantType"`
	Audience           string `json:"audience"`
	Scope              string `json:"scope"`
	RequestedTokenType string `json:"requestedTokenType"`
	SubjectTokenType   string `json:"subjectTokenType"`
	SubjectToken       string `json:"subjectToken"`
}

func NewGCSStore(ctx context.Context, logger *slog.Logger) (*GCSStore, error) {
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")
	gcpWifPoolId := os.Getenv("GCP_WIF_POOL_ID")
	gcpWifPoolProviderId := os.Getenv("GCP_WIF_POOL_PROVIDER_ID")
	encodedgcpStorageRoKey := os.Getenv("GCS_STORAGE_RO_SERVICE_ACCOUNT_KEY_BASE64")
	bucketName := os.Getenv("GCS_BUCKET")
	gcpStorageRoKey, err := base64.StdEncoding.DecodeString(encodedgcpStorageRoKey)
	if err != nil {
		return nil, fmt.Errorf("decode GCS_STORAGE_RO_SERVICE_ACCOUNT_KEY_BASE64: %w", err)
	}
	if gcpProjectId == "" || gcpWifPoolId == "" || gcpWifPoolProviderId == "" || encodedgcpStorageRoKey == "" {
		return nil, os.ErrInvalid
	}
	clientRW := &http.Client{Timeout: 10 * time.Second}
	clientRO, err := storage.NewClient(
		ctx,
		option.WithCredentialsJSON(gcpStorageRoKey),
		option.WithScopes(storage.ScopeReadOnly),
		option.WithUserAgent("cybersocke.com/storage-gcs-ro"),
	)

	if err != nil {
		return nil, fmt.Errorf("creating read-only GCS client: %w", err)
	}

	cache, _ := bigcache.New(ctx, bigcache.DefaultConfig(1*time.Hour))

	store := &GCSStore{
		logger:               logger,
		bucketName:           bucketName,
		clientRW:             clientRW,
		clientRO:             clientRO,
		cache:                cache,
		gcpProjectId:         gcpProjectId,
		gcpWifPoolId:         gcpWifPoolId,
		gcpWifPoolProviderId: gcpWifPoolProviderId,
		// https://cloud.google.com/iam/docs/reference/sts/rest/v1beta/TopLevel/token
		// audience without https:// prefix
		gcpAudience: fmt.Sprintf("//iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s", gcpProjectId, gcpWifPoolId, gcpWifPoolProviderId),
	}

	// Preload cache with all blog posts
	if err := store.preloadCache(ctx); err != nil {
		return nil, fmt.Errorf("preloading cache: %w", err)
	}

	return store, nil
}

// GetPost returns a Post pointer from cache or fetches & parses on miss
func (s *GCSStore) GetPost(slug string, ctx context.Context) (*Post, error) {
	var raw []byte
	if data, err := s.cache.Get(slug); err == nil {
		s.logger.Info("Cache hit for post", slog.String("slug", slug))
		raw = data
	} else {
		var err2 error
		raw, err2 = s.readWithExtension(ctx, slug, ".md")
		if err2 != nil {
			return nil, err2
		}
		s.cache.Set(slug, raw)
	}
	return parsePost(raw)
}

// GetPosts returns all posts as pointers parsed from cache or GCS
func (s *GCSStore) GetPosts(ctx context.Context) (map[string]*Post, error) {
	result := make(map[string]*Post)
	it := s.clientRO.Bucket(s.bucketName).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing objects: %w", err)
		}
		slug := strings.TrimSuffix(attrs.Name, ".md")

		var raw []byte
		if data, err := s.cache.Get(slug); err == nil {
			s.logger.Info("Cache hit for post", slog.String("slug", slug))
			raw = data
		} else {
			raw, err = s.readWithExtension(ctx, slug, ".md")
			if err != nil {
				return nil, err
			}
			s.cache.Set(slug, raw)
		}

		postPtr, err := parsePost(raw)
		if err != nil {
			return nil, err
		}
		result[attrs.Name] = postPtr
	}
	return result, nil
}

func (s *GCSStore) GetAbout() []byte {
	return []byte{}
}

func (s *GCSStore) GetAssets() http.Handler {
	return nil
}

func (s *GCSStore) CreatePost(content []byte, ctx context.Context) error {
	client, err := s.newFederatedStorageClient(ctx)
	if err != nil {
		return err
	}

	postMeta := PostMeta{}
	_, err = frontmatter.Parse(strings.NewReader(string(content)), &postMeta)
	if err != nil {
		return err
	}

	if err := postMeta.Validate(); err != nil {
		return err
	}

	bucket := client.Bucket(s.bucketName)
	obj := bucket.Object(postMeta.Slug + ".md").NewWriter(ctx)
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

func (s *GCSStore) newFederatedStorageClient(ctx context.Context) (*storage.Client, error) {
	session := ctx.Value(session.SessionKey).(*sessions.Session)
	rawID, ok := session.Values["id_token"].(string)

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
	reqBody := GcpTokenRequest{
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		Audience:           s.gcpAudience,
		Scope:              GCP_OIDC_CLOUD_PLATFORM_SCOPE,
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:jwt",
		SubjectToken:       idToken,
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

	resp, err := s.clientRW.Do(req)

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

// preloadCache lists all objects in the bucket and stores their content in cache
func (s *GCSStore) preloadCache(ctx context.Context) error {
	it := s.clientRO.Bucket(s.bucketName).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("listing objects: %w", err)
		}

		obj := s.clientRO.Bucket(s.bucketName).Object(attrs.Name)
		rc, err := obj.NewReader(ctx)
		if err != nil {
			return fmt.Errorf("reading object %s: %w", attrs.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("reading data %s: %w", attrs.Name, err)
		}

		// Store raw content under key = object name
		if err := s.cache.Set(attrs.Name, data); err != nil {
			return fmt.Errorf("caching content %s: %w", attrs.Name, err)
		}
	}
	return nil
}

// readObject reads raw bytes from GCS
func (s *GCSStore) readObject(ctx context.Context, name string) ([]byte, error) {
	rc, err := s.clientRO.Bucket(s.bucketName).Object(name).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("opening object %s: %w", name, err)
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("reading data %s: %w", name, err)
	}
	return data, nil
}

func (s *GCSStore) readWithExtension(ctx context.Context, name, extension string) ([]byte, error) {
	data, err := s.readObject(ctx, name+extension)
	if err != nil {
		return nil, fmt.Errorf("reading object with extension %s: %w", extension, err)
	}
	return data, nil
}

// parsePost converts raw frontmatter+content bytes into a Post
func parsePost(raw []byte) (*Post, error) {
	var meta PostMeta
	body, err := frontmatter.Parse(strings.NewReader(string(raw)), &meta)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}
	return &Post{Meta: meta, Content: body}, nil
}
