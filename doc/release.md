# Release Process

This document describes the release process for Librarian-managed client libraries.

## Overview

Librarian uses a **two-command release workflow**:

1. **`librarian release`** - Prepares the release (commit + tag + push)
2. **`librarian publish`** - Publishes to package registries (PyPI, crates.io) or verifies pkg.go.dev indexing

This design follows Russ Cox's philosophy of simplicity and explicitness: each command does one thing well, with clear separation between version control operations (git) and distribution operations (registry).

## Design Principles

- **Dry-run by default** - Commands show what would happen without making changes
- **Explicit over implicit** - Clear command names, no confusing flags
- **Composable** - Commands work together: `librarian release X && librarian publish X`
- **Reproducible** - Conventional commits determine version bumps automatically
- **Retryable** - Each command is idempotent and can be retried safely

## Command 1: `librarian release`

Creates a release by preparing files, creating a commit, tagging, and pushing to remote.

### Usage

```bash
# Dry-run (shows what would happen)
librarian release secretmanager

# Execute the release
librarian release secretmanager --execute

# Release all changed libraries
librarian release --all --execute

# Override version (instead of auto-detect)
librarian release secretmanager --execute --version 1.16.0

# Create release candidate
librarian release secretmanager --execute --version rc

# Skip tests
librarian release secretmanager --execute --skip-tests
```

### What It Does

When executed, `librarian release` performs these steps:

1. **Validates** working directory is clean
2. **Analyzes** git commits since last tag
3. **Determines** version bump from conventional commits
4. **Runs** tests (unless `--skip-tests`)
5. **Invokes** container to update release files:
   - Updates CHANGES.md with new version and changelog
   - Updates internal/version.go with version constant
   - Updates snippet metadata JSON files
6. **Creates** git commit with all changes
7. **Creates** git tag (e.g., `secretmanager/v1.16.0`)
8. **Pushes** tag to remote

**Does NOT publish to package registries** - use `librarian publish` for that.

### Dry-Run Output

```bash
librarian release secretmanager
```

Shows:
```
Analyzing release for secretmanager...

Current version: 1.15.0
Commits since v1.15.0:
  abc1234 feat(secretmanager): add Secret rotation support
  def5678 fix(secretmanager): handle nil pointers correctly

Proposed version: 1.15.0 → 1.16.0 (minor bump)
  Reason: Found 1 feat commit

Would perform:
  ✓ Run tests (go test ./secretmanager/...)
  ✓ Update CHANGES.md
  ✓ Update internal/version.go
  ✓ Update snippet metadata
  ✓ Create commit: chore(release): secretmanager v1.16.0
  ✓ Create tag: secretmanager/v1.16.0
  ✓ Push tag to origin

To execute:
  librarian release secretmanager --execute

Then publish:
  librarian publish secretmanager --execute
```

### Execute Output

```bash
librarian release secretmanager --execute
```

Shows:
```
Releasing secretmanager...

✓ Validated working directory (clean)
✓ Found last release: secretmanager/v1.15.0
✓ Analyzed 2 commits since last release
✓ Determined version bump: 1.15.0 → 1.16.0 (minor)
✓ Ran tests
✓ Updated CHANGES.md
✓ Updated internal/version.go
✓ Updated snippet metadata
✓ Created commit: chore(release): secretmanager v1.16.0
✓ Created tag: secretmanager/v1.16.0
✓ Pushed tag to origin

Release complete!

Next step:
  librarian publish secretmanager --execute
```

### Version Detection

By default, `librarian release` analyzes conventional commits to determine the version bump:

#### Automatic Version Detection (Default)

```bash
librarian release secretmanager --execute
```

**Commit analysis:**
- `BREAKING CHANGE:` or `feat!:`, `fix!:` → **major bump** (X.0.0)
- `feat:` → **minor bump** (0.X.0)
- `fix:`, `docs:`, `chore:`, `refactor:`, `perf:` → **patch bump** (0.0.X)

**Example:**
```
Current: 1.15.0
Commits:
  feat(secretmanager): add rotation
  fix(secretmanager): handle nil

Result: 1.16.0 (minor bump wins)
```

#### Manual Version Override

Use `--version` to specify the version explicitly:

**Explicit semver:**
```bash
librarian release secretmanager --execute --version 1.16.0
librarian release secretmanager --execute --version 2.0.0-rc.1
```

**Bump level keywords:**
```bash
librarian release secretmanager --execute --version patch   # 1.15.0 → 1.15.1
librarian release secretmanager --execute --version minor   # 1.15.0 → 1.16.0
librarian release secretmanager --execute --version major   # 1.15.0 → 2.0.0
```

**Pre-release keywords:**
```bash
librarian release secretmanager --execute --version rc      # Add/increment -rc.N
librarian release secretmanager --execute --version alpha   # Add/increment -alpha.N
librarian release secretmanager --execute --version beta    # Add/increment -beta.N
```

**Promote to stable:**
```bash
librarian release secretmanager --execute --version release  # 1.16.0-rc.3 → 1.16.0
```

### Pre-Release Workflow

#### Creating a Release Candidate

```bash
# Current: 1.15.0

# Create RC1
librarian release secretmanager --execute --version rc
# → Creates: 1.16.0-rc.1

# Found bugs, create RC2
librarian release secretmanager --execute --version rc
# → Creates: 1.16.0-rc.2 (auto-increments)

# RC2 is good, promote to stable
librarian release secretmanager --execute --version release
# → Creates: 1.16.0 (strips -rc.2)
```

#### Alpha → Beta → Stable

```bash
# Current: 1.15.0

librarian release secretmanager --execute --version alpha
# → 1.16.0-alpha.1

librarian release secretmanager --execute --version beta
# → 1.16.0-beta.1

librarian release secretmanager --execute --version release
# → 1.16.0
```

### Flags

```
--execute              Actually perform the release (default: dry-run)
--version <version>    Version to release (default: auto-detect from commits)
--skip-tests           Skip running tests
--all                  Release all libraries with changes
```

### Version Flag Values

The `--version` flag accepts:

| Value | Description | Example |
|-------|-------------|---------|
| *none* | Auto-detect from commits | `librarian release lib --execute` |
| `1.16.0` | Explicit version | `--version 1.16.0` |
| `1.16.0-rc.1` | Explicit pre-release | `--version 1.16.0-rc.1` |
| `patch` | Bump patch version | `1.15.3 → 1.15.4` |
| `minor` | Bump minor version | `1.15.3 → 1.16.0` |
| `major` | Bump major version | `1.15.3 → 2.0.0` |
| `rc` | Add/increment RC | `1.15.0 → 1.16.0-rc.1` or `1.16.0-rc.1 → 1.16.0-rc.2` |
| `alpha` | Add/increment alpha | `1.15.0 → 1.16.0-alpha.1` |
| `beta` | Add/increment beta | `1.15.0 → 1.16.0-beta.1` |
| `release` | Strip pre-release | `1.16.0-rc.3 → 1.16.0` |

### Error Handling

**Working directory not clean:**
```
Error: Working directory is not clean

Uncommitted changes:
  M secretmanager/client.go
  ?? temp.txt

Commit or stash your changes before releasing.
```

**Tests failed:**
```
Error: Tests failed for secretmanager

--- FAIL: TestGetSecret (0.00s)
    client_test.go:42: expected nil, got error

Fix tests or use --skip-tests (not recommended).
```

**No changes since last release:**
```
No changes found for secretmanager since v1.15.0

Nothing to release.
```

**Tag already exists:**
```
Error: Tag 'secretmanager/v1.16.0' already exists

If you need to re-release:
  git tag -d secretmanager/v1.16.0
  git push origin :secretmanager/v1.16.0

Then run release again.
```

## Command 2: `librarian publish`

Publishes a released library to package registries. Behavior varies by language.

### Usage

```bash
# Dry-run (shows what would be published)
librarian publish secretmanager

# Execute the publish
librarian publish secretmanager --execute

# Publish all tagged libraries
librarian publish --all --execute
```

### What It Does

1. **Finds** the latest git tag for the library
2. **Verifies** tag exists in remote
3. **Publishes** to package registry (language-specific)

### Language-Specific Behavior

#### Go Libraries

For Go, publishing is **verification only** since pkg.go.dev automatically indexes tags.

```bash
librarian publish secretmanager --execute
```

**Output:**
```
Publishing secretmanager...

✓ Found tag: secretmanager/v1.16.0
✓ Tag exists in remote
✓ Checking pkg.go.dev indexing...

Published to pkg.go.dev (auto-indexed from tag)
Track: https://pkg.go.dev/cloud.google.com/go/secretmanager/apiv1@v1.16.0

Note: pkg.go.dev indexes new tags within a few minutes.
```

#### Python Libraries

For Python, publishes to PyPI using `twine`.

```bash
librarian publish google-cloud-secret-manager --execute
```

**Output:**
```
Publishing google-cloud-secret-manager...

✓ Found tag: google-cloud-secret-manager/v1.16.0
✓ Tag exists in remote
✓ Checked out tag
✓ Built distribution
  - google_cloud_secret_manager-1.16.0.tar.gz
  - google_cloud_secret_manager-1.16.0-py3-none-any.whl
✓ Uploaded to PyPI

Published: https://pypi.org/project/google-cloud-secret-manager/1.16.0/
```

**Requires credentials:**
```bash
export TWINE_USERNAME=__token__
export TWINE_PASSWORD=pypi-...
```

Or configure `~/.pypirc`:
```ini
[pypi]
username = __token__
password = pypi-...
```

**Pre-release versions:**
Python normalizes pre-release versions to PEP 440 format:
- `1.16.0-rc.1` → `1.16.0rc1`
- `1.16.0-alpha.1` → `1.16.0a1`
- `1.16.0-beta.1` → `1.16.0b1`

#### Rust Libraries

For Rust, publishes to crates.io using `cargo publish`.

```bash
librarian publish google-cloud-bigtable-admin-v2 --execute
```

**Output:**
```
Publishing google-cloud-bigtable-admin-v2...

✓ Found tag: google-cloud-bigtable-admin-v2/v1.16.0
✓ Tag exists in remote
✓ Checked out tag
✓ Ran cargo semver-checks (no breaking changes detected)
✓ Published to crates.io

Published: https://crates.io/crates/google-cloud-bigtable-admin-v2/1.16.0
```

**Requires credentials:**
```bash
export CARGO_REGISTRY_TOKEN=...
```

Or `~/.cargo/credentials`:
```toml
[registry]
token = "..."
```

**Pre-release versions:**
Rust uses semver format as-is: `1.16.0-rc.1`, `1.16.0-alpha.1`

### Flags

```
--execute              Actually publish (default: dry-run)
--all                  Publish all tagged libraries
```

### Error Handling

**No tag found:**
```
Error: No release tag found for secretmanager

Create a release first:
  librarian release secretmanager --execute

Then publish:
  librarian publish secretmanager --execute
```

**Already published (PyPI):**
```
Error: Version 1.16.0 already exists on PyPI

Package versions on PyPI cannot be overwritten.
Bump version and release again.
```

**Missing credentials (PyPI):**
```
Error: PyPI credentials not found

Set environment variables:
  export TWINE_USERNAME=__token__
  export TWINE_PASSWORD=pypi-...

Or configure ~/.pypirc
```

**Missing credentials (Rust):**
```
Error: Cargo registry token not found

Set environment variable:
  export CARGO_REGISTRY_TOKEN=...

Or run:
  cargo login
```

**Semver violation (Rust):**
```
Error: cargo semver-checks failed

Breaking changes detected in minor release:
  - Removed public function: Client::query()
  - Changed signature: Client::new() now returns Result

This requires a major version bump.
Re-release with:
  librarian release mylib --execute --version major
```

## Complete Workflows

### Standard Release (Go)

```bash
# 1. Preview the release
librarian release secretmanager

# 2. Create the release (includes tag)
librarian release secretmanager --execute

# 3. Verify pkg.go.dev indexing
librarian publish secretmanager --execute
```

### Standard Release (Python/Rust)

```bash
# 1. Preview the release
librarian release google-cloud-storage --execute

# 2. Create the release (includes tag)
librarian release google-cloud-storage --execute

# 3. Publish to registry
librarian publish google-cloud-storage --execute
```

### Release Candidate Workflow

```bash
# Create RC1
librarian release secretmanager --execute --version rc
librarian publish secretmanager --execute

# Test RC1, find bugs, create RC2
librarian release secretmanager --execute --version rc
librarian publish secretmanager --execute

# RC2 is good, promote to stable
librarian release secretmanager --execute --version release
librarian publish secretmanager --execute
```

### Batch Release Multiple Libraries

```bash
# 1. Preview all changes
librarian release --all

# 2. Release all changed libraries
librarian release --all --execute

# 3. Publish all (Python/Rust only)
librarian publish --all --execute
```

### Retry Failed Publish

```bash
# Release succeeded, but publish failed
librarian release mylib --execute
# ✓ Tag created

librarian publish mylib --execute
# ✗ Error: Network timeout

# Fix network, retry publish
librarian publish mylib --execute
# ✓ Published
```

### Review Before Tagging

```bash
# Create release commit locally first
librarian release secretmanager --execute

# Review the changes
git show HEAD
git diff HEAD~1

# If changes look good, the tag is already created and pushed
# (the command does both commit and tag in one step)

# If you need to amend, you'll need to:
# 1. Delete the tag locally and remotely
# 2. Amend the commit
# 3. Re-run librarian release
```

## Files Modified by Release

When `librarian release` runs, these files are updated:

```
<library>/
├── CHANGES.md                    # Changelog with new version
└── internal/
    └── version.go                # Version constant

internal/generated/snippets/<library>/
└── apiv<N>/
    └── snippet_metadata.*.json   # Version field updated
```

Plus git operations:
- New commit in git history
- New tag pointing to that commit
- Tag pushed to remote

## Tag Format

Tags follow the format: `{library}/v{version}`

Examples:
- `secretmanager/v1.16.0`
- `pubsub/v2.5.1`
- `spanner/v4.0.0`
- `secretmanager/v1.16.0-rc.1` (pre-release)
- `google-cloud-secret-manager/v1.16.0` (Python)
- `google-cloud-bigtable-admin-v2/v4.0.0` (Rust)

## Changelog Format

CHANGES.md is updated with entries in Google Cloud Go style:

```markdown
# Changes

## [1.16.0](https://github.com/googleapis/google-cloud-go/releases/tag/secretmanager%2Fv1.16.0) (2025-11-12)

### Features

* add Secret rotation support ([abc1234](https://github.com/googleapis/google-cloud-go/commit/abc1234...))
* another feature ([def5678](https://github.com/googleapis/google-cloud-go/commit/def5678...))

### Bug Fixes

* handle nil pointers correctly ([123456a](https://github.com/googleapis/google-cloud-go/commit/123456a...))

## [1.15.0]...
```

Pre-release versions are included:

```markdown
## [1.16.0] (2025-11-15)
Final release

## [1.16.0-rc.2] (2025-11-14)
Release candidate 2

## [1.16.0-rc.1] (2025-11-13)
Release candidate 1
```

## Configuration

Release behavior is configured in `librarian.yaml`:

```yaml
language: go

release:
  tag_format: '{name}/v{version}'  # Default tag format

libraries:
  - name: secretmanager
    version: 1.15.0                # Current version (informational)
    path: secretmanager/
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
```

The `version` field in library configuration is informational. The source of truth is the git tags.

## Why Two Commands?

This design separates concerns:

- **`librarian release`** = Version control operations (git commit, tag, push)
- **`librarian publish`** = Distribution operations (PyPI, crates.io, pkg.go.dev)

### Benefits

**1. Clear Separation**

Release is about git. Publish is about registries. No confusion.

**2. Retryable**

If publish fails, just retry it. No need to skip tag creation or use complex flags.

```bash
librarian publish mylib --execute  # Failed
librarian publish mylib --execute  # Retry
```

**3. Different Cadences**

Some workflows need different timing:
- Release (tag) daily for Go users
- Publish (PyPI) weekly after QA

**4. Go vs Python/Rust Distinction**

For Go, tag **IS** the release. Publishing is just verification.

For Python/Rust, tag prepares the release. Publishing is distribution.

Having two commands makes this explicit.

**5. No Confusing Flags**

No `--skip-publish`, `--no-tag`, `--push`, etc. Just two simple commands.

## Implementation Notes

### Existing Infrastructure

The release implementation leverages existing container-based code generation:

1. **CLI Layer** (`internal/librarian/` or new `internal/release/`):
   - Parses commits since last tag
   - Determines version bump
   - Runs tests
   - Invokes container's `release-stage` command
   - Creates git commit/tag
   - Pushes to remote

2. **Container Layer** (existing `internal/generate/golang/release`):
   - Already implemented for updating files
   - Reads `release-stage-request.json`
   - Updates CHANGES.md, version files, snippet metadata
   - Writes to output directory

### Container Invocation

The `librarian release` command prepares a request JSON:

```json
{
  "libraries": [{
    "id": "secretmanager",
    "version": "1.16.0",
    "release_triggered": true,
    "source_roots": ["secretmanager"],
    "apis": [{"path": "google/cloud/secretmanager/v1"}],
    "changes": [
      {
        "type": "feat",
        "subject": "add Secret rotation support",
        "commit_hash": "abc1234..."
      },
      {
        "type": "fix",
        "subject": "handle nil pointers",
        "commit_hash": "def5678..."
      }
    ],
    "tag_format": "{id}/v{version}"
  }]
}
```

Then invokes the container which updates the files.

### Language-Specific Publish

The `librarian publish` command delegates to language-specific implementations:

- **Go**: `internal/release/golang/publish.go` - Verifies pkg.go.dev indexing
- **Python**: `internal/release/python/publish.go` - Runs `python -m build` and `twine upload`
- **Rust**: `internal/release/rust/publish.go` - Runs `cargo semver-checks` and `cargo publish`

## Related Documentation

- [Configuration Reference](config.md) - Library version field and tag format
- [User Guide](userguide.md) - Command examples and library types
- [Alternatives Considered](alternatives-considered.md) - Other release designs we evaluated
