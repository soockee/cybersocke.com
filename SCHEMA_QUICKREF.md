# Quick Reference: Schema Management

## For Developers

### Local Development
The application loads the contract from GCS (`gs://cybersocke2-media-storage/schema/post_contract.yaml`) if available, else falls back to the local file. Logs show the source:
```
Post contract loaded source=remote version=2 hash=<hash>
```

### Updating the Schema
1. Edit the file:
```bash
vim schema/post_contract.yaml
```
2. (If version bump) adjust `version:` and add migration block.
3. Commit + tag release:
```bash
git add schema/post_contract.yaml
git commit -m "chore: update schema"
git tag vX.Y.Z
git push origin main --tags
```
4. Release workflow syncs schema if version/hash changed.

### Running Migrations
Dry run:
```bash
go run ./cmd/contract_migrate \
  --bucket cybersocke2-media-storage \
  --schema schema/post_contract.yaml \
  --dry
```
Apply:
```bash
go run ./cmd/contract_migrate \
  --bucket cybersocke2-media-storage \
  --schema schema/post_contract.yaml
```

### Authentication (local)
```bash
gcloud auth application-default login
```

## For Operations
Check current schema version:
```bash
gsutil cat gs://cybersocke2-media-storage/schema/post_contract.yaml | grep '^version:'
```

Manual upload (fallback):
```bash
gsutil -h 'Content-Type:application/x-yaml' cp schema/post_contract.yaml gs://cybersocke2-media-storage/schema/post_contract.yaml
```

List object versions:
```bash
gsutil ls -la gs://cybersocke2-media-storage/schema/post_contract.yaml
```

Rollback to earlier generation:
```bash
gsutil cp gs://cybersocke2-media-storage/schema/post_contract.yaml#<generation> gs://cybersocke2-media-storage/schema/post_contract.yaml
```

## Service Accounts
GitHub Actions sync uses Workload Identity Federation:
- Service Account: `schema-manager@dz-cybersocke02.iam.gserviceaccount.com`
- Provider: `dz-bootstrap-github-soockee`
- Required IAM: `storage.objects.get`, `storage.objects.create` on `schema/*`

## Schema Version Guidelines

### When to Increment Version

**Increment version when:**
- Adding new required fields
- Removing fields (use migration to clean up posts)
- Changing field validation rules (type, format, etc.)
- Modifying tag family constraints

**Version stays same when:**
- Adding optional fields
- Updating field descriptions
- Fixing typos in comments

### Migration Steps Structure

```yaml
migrations:
  - from: 2
    to: 3
    steps:
      - op: removeField
        field: old_field
      - op: renameField
        field: old_name
        toField: new_name
      - op: setField
        field: new_field
        value: "default"
```

**Supported operations:**
- `removeField`: Delete a field
- `renameField`: Rename a field
- `setField`: Add/update a field with value

## Emergency Procedures

### Schema Corruption in Bucket

1. Check local repo version:
   ```bash
   cat schema/post_contract.yaml | grep "^version:"
   ```

2. Restore from repo:
```bash
gsutil cp schema/post_contract.yaml gs://cybersocke2-media-storage/schema/post_contract.yaml
```

3. Restart application (will pick up corrected schema)

### Posts Require Urgent Rollback

1. Revert schema version in repo
2. Push tag to trigger workflow
3. Run migration tool with older schema
4. Deploy application with matching schema

## Monitoring

### Key Metrics to Watch

- Schema load source (`remote` vs `local` in logs)
- Schema version mismatches across environments
- Migration tool success/failure rates
- GitHub Actions sync job status

### Health Checks

**Application startup:**
```
✓ Post contract loaded source=remote version=2 hash=abc123
```

**Migration tool:**
```
✓ schema loaded path=schema/post_contract.yaml version=2 hash=abc123
✓ schema persisted version=2 hash=abc123
✓ migration complete changed=5 total=42
```

**GitHub Actions:**
```
✓ Local schema: version=2 hash=abc123
✓ Remote schema: version=2 hash=abc123
✓ Schema unchanged (skipping upload)
```
