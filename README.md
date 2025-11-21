# cybersocke.com

This repository contains the server for cybersocke.com and a small utility program used to set Firebase claims (`cmd/set_firebase_claims`).

Below are the environment variables actually read by the code (from `config/config.go` and the utility's `claimconfig`). Documented are required and optional variables and which program they apply to.

Loading environment from a `.env` file (example):

```bash
set -o allexport
source .env
set +o allexport
```

## Server (main web application)

The server reads environment variables using `viper`. The following variables are required for the server to start:

Required:

```bash
SESSION_SECRET=                   # Required. Random secret for session signing.
CSRF_SECRET=                      # Required. Random secret used for CSRF token generation.
FIREBASE_CREDENTIALS_BASE64=      # Required. Firebase Admin SDK service account JSON, base64-encoded.
GCS_BUCKET=                       # Required. Name of the Google Cloud Storage bucket containing blog posts and assets.
GCS_CREDENTIALS_BASE64=           # Required. Service account JSON (base64) used for GCS access when not using Workload Identity.
```

Optional / additional configuration (server will start without these but they affect runtime behavior):

```bash
FIREBASE_INSENSITIVE_API_KEY=     # Optional. Firebase Web API key used by the frontend (note: variable name in code is FIREBASE_INSENSITIVE_API_KEY).
FIREBASE_AUTH_DOMAIN=             # Optional. Firebase Auth domain (e.g. "example.firebaseapp.com").
GCP_PROJECT_NAME=                 # Optional informational/project name used by the server (config key: GCP_PROJECT_NAME).
ORIGIN=                           # Optional. Origin used for CORS or security policy.
ENVIRONMENT=                      # Optional. Defaults to "development". Example: production, staging, dev.
LOCAL_DEV=                         # Optional. Boolean flag for local dev behavior (default false).
```

Notes:
- `FIREBASE_CREDENTIALS_BASE64` and `GCS_CREDENTIALS_BASE64` must contain the raw JSON credential file encoded using base64 (do not commit the decoded JSON).
- The code expects the API key variable to be named `FIREBASE_INSENSITIVE_API_KEY` (this is the name used in `config/config.go`).
- The server requires `SESSION_SECRET` and `CSRF_SECRET`.

## Utility: set_firebase_claims (cmd/set_firebase_claims)

This small CLI/utility has its own configuration separate from the server. It is used to impersonate a service account and set Firebase custom claims. Its required environment variables are:

```bash
FIREBASE_IMPERSONATE_SERVICE_ACCOUNT=  # Required. The email of the service account to impersonate (caller must have iam.serviceAccountTokenCreator).
GCP_PROJECT_ID=                         # Required. The GCP/Firebase project ID used by the utility.
```

Note: The utility uses `GCP_PROJECT_ID` whereas the main server uses `GCP_PROJECT_NAME` (if provided). Keep these distinct.

## Running outside GCP

When running the server outside GCP (for example on Hetzner), Workload Identity / OIDC will not be available. The application therefore authenticates to Google Cloud Storage using the service account JSON supplied via `GCS_CREDENTIALS_BASE64`. Provide the raw JSON key base64-encoded.

## Example `.env` snippet

```bash
SESSION_SECRET="$(openssl rand -base64 32)"
CSRF_SECRET="$(openssl rand -base64 32)"
FIREBASE_CREDENTIALS_BASE64="<base64-encoded-json>"
GCS_CREDENTIALS_BASE64="<base64-encoded-json>"
GCS_BUCKET="my-gcs-bucket"
FIREBASE_INSENSITIVE_API_KEY="your-web-api-key"
FIREBASE_AUTH_DOMAIN="example.firebaseapp.com"
GCP_PROJECT_NAME="my-gcp-project-name"
ENVIRONMENT=production
LOCAL_DEV=false
```

## Quick verification

After exporting your environment, run the server (for example `go run .` from the repository root) and it will fail fast with a clear error if any required config is missing. The error lists the missing variables.

If you intend to run the `set_firebase_claims` utility, ensure the `FIREBASE_IMPERSONATE_SERVICE_ACCOUNT` and `GCP_PROJECT_ID` variables are set in your environment.
 
## Tag-Based Navigation & Content Graph

The storage layer builds an in-memory reverse index of tags to posts and exposes higher-level navigation helpers. Each post's frontmatter `tags` implements a lightweight architecture using families (`type/`, `source/`, `theme/`, etc.).

### Validation

`CreatePost` now validates tag families & cardinalities (MVP subset):
* `type/*`: 1–2 required
* `theme/*`: 1–5 required
* `source/*`: ≤1
* `structure/*`: ≤1

Unknown families or malformed values (`family/value` required) are rejected.
## JSON API Separation

All JSON responses are now served under the `/api` prefix to clearly distinguish them from HTML content pages.

Current JSON endpoints:

```
GET /api/graph                 # Tag graph JSON (query: minSharedTags, includeTags, maxEdges)
GET /api/posts/{id}/adjacency  # Neighboring posts sharing tags (query: includeTags, minShared, limit)
```

Legacy path `GET /posts/{id}/adjacency` has been removed from the public non-API router. Update any clients to use `/api/posts/{id}/adjacency`.

Example usage:

```bash
curl http://localhost:8080/api/graph
curl "http://localhost:8080/api/posts/some-post.md/adjacency?limit=12"
```

http://localhost:8080/posts/simon-stockhause.md