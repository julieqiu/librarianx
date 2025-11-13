# Librarian Operations (librarianops)

This document describes `librarianops`, a tool for librarian team operations that manage client libraries across multiple language repositories.

## Overview

`librarianops` is a separate command-line tool used by the librarian team to perform operations across all language repositories (google-cloud-go, google-cloud-python, google-cloud-rust, etc.).

**Key distinction:**
- `librarian` - Used by language teams to manage a single repository
- `librarianops` - Used by librarian team to manage multiple repositories

## Installation

```bash
go install github.com/julieqiu/librarian/cmd/librarianops@latest
```

## Core Concept: Centralized Language Enablement

### Problem

When onboarding a new API, the librarian team needs to:
1. Add it to google-cloud-go/librarian.yaml
2. Add it to google-cloud-python/librarian.yaml
3. Add it to google-cloud-rust/librarian.yaml

**But which languages should be enabled?** The API team should control this.

### Solution

Language enablement is configured in the **service_config.yaml** file in googleapis.

**Example:** `googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml`

```yaml
type: google.api.Service
name: secretmanager.googleapis.com
title: Secret Manager API

publishing:
  documentation_uri: https://cloud.google.com/secret-manager/docs
  new_issue_uri: https://github.com/googleapis/google-cloud-go/issues/new

  # Language enablement (NEW)
  client_libraries:
    go:
      enabled: true
    python:
      enabled: true
    rust:
      enabled: false  # Disabled by API team
      reason: "Rust support planned for Q2 2025"
```

**Default behavior:** If `client_libraries` is not specified, all languages are enabled.

## Commands

### `librarianops sync-repos <api-path>`

Synchronizes an API across all language repositories based on service_config.

```bash
# Sync a single API
librarianops sync-repos google/cloud/newapi/v1

# What it does:
# 1. Reads googleapis/google/cloud/newapi/v1/newapi_v1.yaml
# 2. Checks publishing.client_libraries
# 3. For each enabled language:
#    - Clones/updates the language repo
#    - Runs: librarian add newapi google/cloud/newapi/v1
#    - Creates PR with changes
```

**Options:**
```bash
# Dry-run (show what would happen)
librarianops sync-repos google/cloud/newapi/v1 --dry-run

# Sync all new APIs (not yet in any repo)
librarianops sync-repos --all-new

# Force sync even if already exists
librarianops sync-repos google/cloud/newapi/v1 --force

# Skip PR creation (just update local clones)
librarianops sync-repos google/cloud/newapi/v1 --no-pr
```

**Output:**
```
Syncing google/cloud/newapi/v1...

Reading service config: googleapis/google/cloud/newapi/v1/newapi_v1.yaml
  ✓ Go: enabled
  ✓ Python: enabled
  ✗ Rust: disabled (Rust support planned for Q2 2025)

[google-cloud-go]
  Cloning repository...
  Running: librarian add newapi google/cloud/newapi/v1
  ✓ Added newapi to librarian.yaml
  Creating PR: https://github.com/googleapis/google-cloud-go/pull/12345

[google-cloud-python]
  Cloning repository...
  Running: librarian add google-cloud-newapi google/cloud/newapi/v1
  ✓ Added google-cloud-newapi to librarian.yaml
  Creating PR: https://github.com/googleapis/google-cloud-python/pull/6789

[google-cloud-rust]
  ⊘ Skipped (disabled in service_config)

Summary:
  Go:     PR #12345 created
  Python: PR #6789 created
  Rust:   Skipped (disabled)
```

---

### `librarianops audit-coverage`

Audits which APIs are available in which languages.

```bash
# Audit all APIs
librarianops audit-coverage

# Audit specific API
librarianops audit-coverage google/cloud/secretmanager/v1

# Output format options
librarianops audit-coverage --format=json
librarianops audit-coverage --format=csv
librarianops audit-coverage --format=table  # default
```

**Output:**
```
API Coverage Report
===================

Total APIs: 150
  Go:     148 (98.7%)
  Python: 145 (96.7%)
  Rust:   89 (59.3%)

Missing from Go (2 APIs):
  - google/cloud/example/v1 (disabled in service_config)
  - google/cloud/beta/v1 (disabled in service_config)

Missing from Python (5 APIs):
  - google/cloud/example/v1 (disabled in service_config)
  - google/cloud/internal/v1 (not in librarian.yaml)
  - ...

Missing from Rust (61 APIs):
  - google/cloud/secretmanager/v1 (disabled in service_config)
  - google/cloud/storage/v2 (not in librarian.yaml)
  - ...

Discrepancies (enabled in service_config but not in repo):
  ⚠ google/cloud/foo/v1
    - service_config: go.enabled = true
    - google-cloud-go: NOT FOUND in librarian.yaml
    - Action: Run 'librarianops sync-repos google/cloud/foo/v1'
```

---

### `librarianops validate-config`

Validates service_config files for all APIs.

```bash
# Validate all service configs
librarianops validate-config

# Validate specific API
librarianops validate-config google/cloud/secretmanager/v1
```

**Checks:**
- service_config file exists
- `publishing.client_libraries` format is valid
- Language names are valid (go, python, rust)
- If `enabled: false`, `reason` is provided
- No conflicts between service_config and actual repos

**Output:**
```
Validating service configs...

✓ google/cloud/secretmanager/v1
  - service_config found
  - client_libraries config valid
  - All languages synced correctly

✗ google/cloud/newapi/v1
  - service_config found
  - client_libraries config valid
  - ERROR: go.enabled = true but not found in google-cloud-go/librarian.yaml
    Action: Run 'librarianops sync-repos google/cloud/newapi/v1'

⚠ google/cloud/oldapi/v1
  - service_config found
  - WARNING: client_libraries not specified (assuming all enabled)
  - python: found in google-cloud-python
  - go: found in google-cloud-go
  - rust: NOT found in google-cloud-rust
    Action: Add client_libraries config to service_config

Summary:
  ✓ Valid: 145
  ⚠ Warnings: 3
  ✗ Errors: 2
```

---

### `librarianops update-repos <api-path>`

Updates an existing API across all repos (regenerates code).

```bash
# Update a single API across all repos
librarianops update-repos google/cloud/secretmanager/v1

# What it does:
# 1. For each language repo:
#    - Clones/updates the repo
#    - Runs: librarian generate secretmanager
#    - Creates PR with updated code
```

---

### `librarianops remove-repos <api-path>`

Removes an API from all language repositories.

```bash
# Remove an API from all repos
librarianops remove-repos google/cloud/deprecated/v1

# What it does:
# 1. For each language repo where the API exists:
#    - Clones/updates the repo
#    - Runs: librarian remove <library-name>
#    - Creates PR with removal

# Requires confirmation
librarianops remove-repos google/cloud/deprecated/v1 --confirm
```

---

## Configuration

`librarianops` uses a configuration file: `~/.config/librarianops/config.yaml`

```yaml
# Repository locations
repos:
  go:
    url: https://github.com/googleapis/google-cloud-go
    path: ~/code/google-cloud-go
  python:
    url: https://github.com/googleapis/google-cloud-python
    path: ~/code/google-cloud-python
  rust:
    url: https://github.com/googleapis/google-cloud-rust
    path: ~/code/google-cloud-rust

# googleapis location
googleapis:
  url: https://github.com/googleapis/googleapis
  path: ~/code/googleapis

# GitHub settings
github:
  token: $GITHUB_TOKEN  # or specify directly
  pr_template: |
    chore: add {{.API}} client library

    This PR adds the {{.API}} API to this repository.

    Source: googleapis commit {{.GoogleapisCommit}}

    Generated by librarianops.

# Default PR settings
pr:
  auto_merge: false
  draft: true  # Create as draft PRs
  labels:
    - autogenerated
    - api-addition
```

---

## service_config.yaml Schema

### Minimal Example (all languages enabled)

```yaml
type: google.api.Service
name: secretmanager.googleapis.com
title: Secret Manager API

publishing:
  documentation_uri: https://cloud.google.com/secret-manager/docs
  # No client_libraries section = all languages enabled by default
```

### Full Example (selective enablement)

```yaml
publishing:
  documentation_uri: https://cloud.google.com/secret-manager/docs

  client_libraries:
    # Each language can be:
    # 1. Omitted (defaults to enabled: true)
    # 2. Explicitly enabled
    # 3. Explicitly disabled with reason

    go:
      enabled: true
      # Optional: language-specific overrides
      # These override BUILD.bazel defaults
      module_name: cloud.google.com/go/secretmanager
      release_level: ga

    python:
      enabled: true
      package_name: google-cloud-secret-manager

    rust:
      enabled: false
      reason: "Waiting for Rust team bandwidth - Q2 2025"
      # Optional: planned enablement
      planned_date: "2025-06-01"

    # Future languages
    java:
      enabled: true
    csharp:
      enabled: true
    php:
      enabled: false
      reason: "API team decision - limited PHP demand"
```

### Validation Rules

1. **enabled** (boolean, optional, default: true)
   - If `false`, `reason` is required

2. **reason** (string, required if enabled: false)
   - Human-readable explanation

3. **Language names** must be valid:
   - Supported: `go`, `python`, `rust`, `java`, `csharp`, `php`, `ruby`, `nodejs`

4. **Language-specific fields** are optional and override BUILD.bazel values

---

## Workflows

### Onboarding a New API

**Scenario:** API team adds `google/cloud/newapi/v1` to googleapis

**Steps:**

1. **API team updates service_config:**
   ```yaml
   # googleapis/google/cloud/newapi/v1/newapi_v1.yaml
   publishing:
     client_libraries:
       go:
         enabled: true
       python:
         enabled: true
       rust:
         enabled: false
         reason: "Rust support planned for Q2 2025"
   ```

2. **Librarian team runs sync:**
   ```bash
   cd ~/code/librarianops
   librarianops sync-repos google/cloud/newapi/v1 --dry-run
   # Review output, verify it looks correct

   librarianops sync-repos google/cloud/newapi/v1
   # Creates PRs in google-cloud-go and google-cloud-python
   ```

3. **Review and merge PRs**
   - Language teams review the auto-generated PRs
   - Merge when ready

---

### Disabling a Language for an API

**Scenario:** Rust team decides to stop supporting Secret Manager

**Steps:**

1. **Update service_config:**
   ```yaml
   # googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml
   publishing:
     client_libraries:
       rust:
         enabled: false
         reason: "Deprecated due to low usage"
   ```

2. **Remove from Rust repo:**
   ```bash
   librarianops remove-repos google/cloud/secretmanager/v1 --confirm
   # Creates PR in google-cloud-rust to remove the library
   ```

---

### Auditing API Coverage

**Scenario:** Check which APIs are missing from Rust

```bash
librarianops audit-coverage --format=table | grep "Missing from Rust"

# Or get structured output
librarianops audit-coverage --format=json > coverage.json
```

---

### Validating Configuration

**Scenario:** Ensure all service_config files are valid and in sync

```bash
# Run before/after googleapis changes
librarianops validate-config

# Fix any errors
librarianops sync-repos google/cloud/problematic/v1
```

---

## Implementation Notes

### Repository Management

`librarianops` manages local clones of all language repos:

```
~/.config/librarianops/repos/
├── google-cloud-go/
├── google-cloud-python/
└── google-cloud-rust/
```

**Update strategy:**
1. Clone once (if not exists)
2. Pull latest `main` before each operation
3. Create feature branch for changes
4. Push and create PR

### PR Creation

Uses GitHub API to create PRs:
- Title: `chore: add {api-name} client library`
- Body: Generated from template
- Labels: `autogenerated`, `api-addition`
- Draft mode by default (requires manual approval)

### Error Handling

If sync fails for one language, continue with others:

```
[google-cloud-go]
  ✓ Success

[google-cloud-python]
  ✗ Error: BUILD.bazel not found for google/cloud/newapi/v1

[google-cloud-rust]
  ⊘ Skipped (disabled)

Summary:
  1 success, 1 error, 1 skipped

Review errors above and retry:
  librarianops sync-repos google/cloud/newapi/v1 --retry-failed
```

---

## Relationship to librarian

| Command | Used By | Purpose |
|---------|---------|---------|
| `librarian` | Language teams | Manage single repository |
| `librarianops` | Librarian team | Manage across all repositories |

**Example:**

```bash
# Language team (google-cloud-go maintainer)
cd google-cloud-go
librarian add secretmanager google/cloud/secretmanager/v1
librarian generate secretmanager

# Librarian team (cross-repo operations)
librarianops sync-repos google/cloud/secretmanager/v1
librarianops audit-coverage
```

---

## Design Decisions

### Language Team Overrides

**Q: What if service_config says `rust.enabled: true` but the Rust team doesn't want to generate it?**

**A:** Language teams can override by:
1. Not accepting the auto-generated PR from `librarianops sync-repos`
2. Adding the library to their `librarian.yaml` with `disabled: true`:

```yaml
# google-cloud-rust/librarian.yaml
libraries:
  - name: google-cloud-secretmanager-v1
    disabled: true
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
    # Comment explaining why Rust team disabled it
```

This creates a clear audit trail: service_config says "enabled" but language repo says "disabled" with a reason.

### Service Config Ownership

**Q: Who owns the service_config updates?**

**A:** API teams own the service_config files. They update `publishing.client_libraries` when:
- Launching a new API
- Enabling/disabling languages for their API
- Changing API metadata

### Backfilling Existing APIs

**Q: What about existing APIs?**

**A:** We need to backfill `publishing.client_libraries` for all existing APIs.

**Backfilling workflow:**

```bash
# 1. Generate backfill data by auditing current repos
librarianops backfill-generate > backfill.yaml

# This creates a YAML file with current state:
# apis:
#   - path: google/cloud/secretmanager/v1
#     languages:
#       go: true      # Found in google-cloud-go
#       python: true  # Found in google-cloud-python
#       rust: false   # Not found in google-cloud-rust

# 2. Review and edit backfill.yaml
# Add reasons for disabled languages if needed

# 3. Apply backfill to googleapis service_config files
librarianops backfill-apply backfill.yaml --dry-run
librarianops backfill-apply backfill.yaml

# This updates all service_config files with client_libraries sections
```

**Backfill command details:**

`librarianops backfill-generate`:
- Scans all language repos
- For each API, determines which languages have it
- Outputs YAML mapping APIs → languages

`librarianops backfill-apply`:
- Reads backfill YAML
- Updates googleapis service_config files
- Adds `publishing.client_libraries` section
- Creates one large googleapis PR with all updates

## Future Enhancements

1. **Automated PR merging** - Auto-merge if tests pass
2. **Rollback support** - Undo a sync operation
3. **Batch operations** - Sync multiple APIs at once
4. **Webhooks** - Auto-sync when googleapis changes
5. **Metrics dashboard** - Track API coverage over time
6. **Language-specific overrides** - Override BUILD.bazel values from service_config
7. **Backfill validation** - Validate backfill data before applying

---

## See Also

- [userguide.md](userguide.md) - CLI for single repository management
- [config.md](config.md) - librarian.yaml schema reference
- [prd.md](prd.md) - Project objectives and design principles
