# cybersocke.com

add env:
```bash
set -o allexport
source .env
set +o allexport
```

todo: 
doc authentication design

Required environment (secrets & config):

```bash
FIREBASE_CREDENTIALS_BASE64=        # Firebase admin SDK service account (base64 JSON)
CSRF_SECRET=                       # Random 32+ bytes base64
FIREBASE_INSENSITIVE_API_KEY=      # Firebase Web API key (frontend)
FIREBASE_AUTH_DOMAIN=              # Firebase auth domain
GCP_PROJECT_ID=                    # GCP project hosting the storage bucket
GCS_BUCKET=                        # Name of the GCS bucket with blog posts
GCS_CREDENTIALS_BASE64=            # Service account JSON (base64) used for GCS access from Hetzner
```

Running outside GCP (e.g. Hetzner) means no Workload Identity / OIDC is available. The application therefore authenticates to Google Cloud Storage using a service account JSON key supplied via `GCS_CREDENTIALS_BASE64`. Provide the raw JSON key, base64-encoded. Do NOT commit the decoded key.