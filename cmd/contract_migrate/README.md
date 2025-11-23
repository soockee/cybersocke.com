# Contract Migration Tool

CLI tool for migrating blog post frontmatter schemas and persisting the contract schema to GCS.

## Features

- Loads a post contract schema from a local file
- Applies migrations to all posts in a GCS bucket
- Persists the schema to `schema/post_contract.yaml` in the bucket (with version comparison)
- Supports dry-run mode for preview without changes

## Usage

```bash
go run ./cmd/contract_migrate \
  --bucket <gcs-bucket-name> \
  --schema schema/post_contract.yaml \
  [--dry]
```

### Flags

- `--bucket` (required): GCS bucket name where posts are stored
- `--schema` (required): Path to the local schema YAML file
- `--dry`: Dry run mode - shows planned changes without writing

### Authentication

Uses Application Default Credentials (ADC). Authenticate via:

```bash
gcloud auth application-default login
```

Or configure Workload Identity Federation in CI/CD environments.

## Examples

### Dry run to preview changes

```bash
go run ./cmd/contract_migrate \
  --bucket my-blog-bucket \
  --schema schema/post_contract.yaml \
  --dry
```

### Apply migrations and persist schema

```bash
go run ./cmd/contract_migrate \
  --bucket my-blog-bucket \
  --schema schema/post_contract.yaml
```

## Schema Persistence

When not in dry-run mode, the tool:

1. Compares the provided schema hash with the remote `schema/post_contract.yaml`
2. Uploads the schema only if the version or hash differs
3. Logs the persistence status (uploaded or skipped)

The remote schema serves as the source of truth for the application at runtime.
