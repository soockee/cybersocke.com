# Schema Management Architecture

## System Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     GitHub Repository                            │
│                  schema/post_contract.yaml                       │
│                      (version: 2)                                │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     │ Git Push (tag: v*)
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│              GitHub Actions Workflow                             │
│         (.github/workflows/build-and-publish.yaml)               │
├─────────────────────────────────────────────────────────────────┤
│  Job 1: build-and-push-image                                     │
│    ├─ Build Docker image                                         │
│    └─ Push to ghcr.io                                            │
│                                                                   │
│  Job 2: sync-schema (after build)                                │
│    ├─ Authenticate via WIF                                       │
│    │   Service Account: schema-manager@...                       │
│    ├─ Extract local schema version/hash                          │
│    ├─ Fetch remote schema (if exists)                            │
│    ├─ Compare version & hash                                     │
│    └─ Upload if changed → gs://bucket/schema/post_contract.yaml  │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                   GCS Bucket (Source of Truth)                   │
│                                                                   │
│  schema/post_contract.yaml  ← Canonical schema (versioned)       │
│                                                                   │
│  posts/*.md                 ← Blog posts with frontmatter        │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     │ On Startup: GetSchema(ctx)
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│              Application (main.go)                               │
├─────────────────────────────────────────────────────────────────┤
│  1. Load Config                                                  │
│  2. Initialize GCS Store                                         │
│  3. LoadPostContractRemote(ctx, store)                           │
│     ├─ Try: store.GetSchema(ctx)                                 │
│     │   Success → ParsePostContract(remoteData)                  │
│     │             ✓ source: "remote"                             │
│     │                                                             │
│     └─ Fail (auth/network/not found)                             │
│         Fallback → LoadPostContract() [local file]               │
│                    ✓ source: "local"                             │
│                                                                   │
│  4. Log: version, hash, source                                   │
│  5. Start Server                                                 │
└─────────────────────────────────────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────┐
│          Migration Tool (cmd/contract_migrate)                   │
├─────────────────────────────────────────────────────────────────┤
│  Flags: --bucket --schema [--dry]                                │
│                                                                   │
│  1. Read local schema file (--schema path)                       │
│  2. ParsePostContract(data)                                      │
│  3. persistSchema() [unless --dry]                               │
│     ├─ GetSchema(ctx) from remote                                │
│     ├─ Compare hashes                                            │
│     └─ PutSchema(ctx, data) if changed                           │
│                                                                   │
│  4. GetPosts(ctx)                                                │
│  5. For each post:                                               │
│     ├─ Parse frontmatter                                         │
│     ├─ ApplyMigrations(frontmatter)                              │
│     └─ UpdatePost() if changed [unless --dry]                    │
└─────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Remote-First with Graceful Fallback
- **Why:** Ensures consistency across deployments; single source of truth
- **Fallback:** Local schema used on auth/network failures (no startup failure)
- **Logging:** Clear indication of source (`remote` vs `local`)

### 2. Hash-Based Change Detection
- **Why:** Avoids redundant GCS writes; respects object versioning
- **Method:** SHA256 (first 8 bytes, hex-encoded = 16 chars)
- **Comparison:** Version number + hash ensures accurate change detection

### 3. Separate Schema Namespace
- **Prefix:** `schema/` (vs `posts/`)
- **Why:** Logical separation; easier IAM policies; future multi-schema support

### 4. Workflow Integration
- **Trigger:** Tag push (`v*`) - aligns with release cadence
- **Auth:** Workload Identity Federation (no secrets)
- **Conditional:** Upload only if version/hash differs

### 5. Migration Tool Persistence
- **When:** Automatically persists unless `--dry`
- **Failure Mode:** Non-fatal warning; continues with post migrations
- **Use Case:** Manual schema updates, initial bootstrap

## Authentication Methods

### Application Runtime
- **Hetzner Deployment:** `NewGCSStore()` with base64 service account JSON
- **GCP Deployment:** `NewGCSStoreADC()` using Workload Identity / ADC

### Migration Tool
- **Method:** Application Default Credentials (ADC)
- **Setup:** `gcloud auth application-default login`

### GitHub Actions
- **Method:** Workload Identity Federation (OIDC)
- **Provider:** `dz-bootstrap-github-soockee`
- **Service Account:** `schema-manager@dz-cybersocke02.iam.gserviceaccount.com`

## Security Considerations

1. **Minimal Permissions:** `schema-manager` SA should only access `schema/*` path
2. **No Long-Lived Keys:** GitHub Actions uses WIF (short-lived tokens)
3. **Version Control:** GCS object versioning enabled for audit/rollback
4. **Validation:** Schema parsed and validated before persistence

## Migration Path Example

### Initial State
```
Local: schema/post_contract.yaml (version: 2, hash: abc123)
Remote: (not exists)
```

### Step 1: Run Migration Tool
```bash
go run ./cmd/contract_migrate \
  --bucket my-bucket \
  --schema schema/post_contract.yaml
```
**Result:** 
- Schema uploaded to `gs://my-bucket/schema/post_contract.yaml`
- Posts migrated to version 2

### Step 2: Deploy Application
```
Application startup:
  ├─ Try remote schema → Success (version: 2, hash: abc123)
  └─ Log: source=remote
```

### Step 3: Update Schema (version: 3)
```
Developer: Edit schema/post_contract.yaml → version: 3
Git: Commit and push tag v1.2.0
```

**Result:**
- GitHub Actions detects version change (2 → 3)
- Uploads new schema to GCS
- Next deployment loads version 3 from remote

### Step 4: Migrate Posts
```bash
go run ./cmd/contract_migrate \
  --bucket my-bucket \
  --schema schema/post_contract.yaml
```
**Result:**
- Schema already persisted (skipped)
- Posts migrated from v2 → v3

## Troubleshooting

### Application uses local schema instead of remote
- **Check:** GCS bucket permissions for application service account
- **Check:** Schema exists at `gs://<bucket>/schema/post_contract.yaml`
- **Check:** Logs for remote fetch error details

### Migration tool persistence fails
- **Check:** ADC credentials configured (`gcloud auth application-default login`)
- **Check:** Service account has `storage.objects.create` permission
- **Non-fatal:** Posts will still be migrated; manual upload may be needed

### GitHub Actions sync fails
- **Check:** Workload Identity Federation configuration
- **Check:** Service account impersonation permissions
- **Check:** Bucket name in workflow matches actual bucket
