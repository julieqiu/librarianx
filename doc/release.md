# Release Process Design

This document describes the release process for Librarian-managed client libraries.

## Overview

Librarian uses a **git-based release workflow** where:
- **Git history** is the source of truth for what needs to be released
- **Conventional commits** determine the type of version bump
- **Single command** does everything (version bump, changelog, tag, publish)
- **Dry-run by default** for safety - must use `--execute` to actually release
- **No GitHub dependency** - works with any git hosting

## librarian release

The `librarian release` command is an all-in-one release tool that handles the complete release workflow.

### What it does

When executed (with `--execute` flag), the command performs these steps in order:

1. **Analyzes conventional commits** since the last tag to determine version bump type
2. **Updates version files** with the new version
3. **Updates changelog files** with commit history since last release
4. **Creates a commit** with all version and changelog updates
5. **Creates git tags** for the release
6. **Pushes tags** to remote repository
7. **Publishes to package registries** (PyPI, crates.io, or auto-indexed by pkg.go.dev)

### Usage

```bash
# Dry-run (default): show what would be released
librarian release secretmanager

# Actually perform the release
librarian release secretmanager --execute

# Release all changed libraries (dry-run)
librarian release --all

# Release all changed libraries (execute)
librarian release --all --execute

# Skip running tests before release
librarian release secretmanager --execute --skip-tests

# Skip publishing to registries (only create tags)
librarian release secretmanager --execute --skip-publish
```

### Version Bump Detection

The command analyzes conventional commits since the last tag to determine the version bump type:

- **Major bump** (X.0.0) - Any commit with `!` or `BREAKING CHANGE:` footer
  - `feat!:`, `fix!:`, `BREAKING CHANGE: ...`
- **Minor bump** (0.X.0) - `feat:` commits (new features)
- **Patch bump** (0.0.X) - `fix:`, `chore:`, `docs:`, `refactor:`, etc.

Example:
```bash
# Since secretmanager/v1.11.0
git log secretmanager/v1.11.0..HEAD --oneline -- secretmanager/

feat(secretmanager): add Secret rotation support    → minor bump
fix(secretmanager): handle nil pointers correctly   → patch bump
```

Result: 1.11.0 → 1.12.0 (minor wins over patch)

### Changelog Format

The command generates changelogs in Keep a Changelog format:

```markdown
# Changelog

## [1.12.0] - 2025-01-15

### Added
- Add Secret rotation support

### Fixed
- Handle nil pointers correctly in GetSecret

## [1.11.0] - 2025-01-01
...
```

### Output (Dry-Run Mode)

Running `librarian release secretmanager` (without `--execute`):

```
Analyzing secretmanager for release...

Pending changes since v1.11.0:
  feat(secretmanager): add Secret rotation support
  fix(secretmanager): handle nil pointers correctly

Proposed version: 1.11.0 → 1.12.0 (minor bump)

Would perform:
  ✓ Run tests for secretmanager
  ✓ Update secretmanager/internal/version.go: 1.11.0 → 1.12.0
  ✓ Update secretmanager/CHANGELOG.md
  ✓ Create commit: chore(release): secretmanager v1.12.0
  ✓ Create git tag: secretmanager/v1.12.0
  ✓ Push tag to origin
  ✓ Publish to pkg.go.dev (auto-indexed from tag)

To proceed, run:
  librarian release secretmanager --execute
```

### Output (Execute Mode)

Running `librarian release secretmanager --execute`:

```
Releasing secretmanager...

✓ Ran tests for secretmanager
✓ Updated secretmanager/internal/version.go: 1.11.0 → 1.12.0
✓ Updated secretmanager/CHANGELOG.md
✓ Created commit: chore(release): secretmanager v1.12.0
✓ Created tag: secretmanager/v1.12.0
✓ Pushed tag to origin
✓ Published to pkg.go.dev

Release complete!
Track indexing: https://pkg.go.dev/cloud.google.com/go/secretmanager/apiv1@v1.12.0
```

### Output (Multiple Libraries)

Running `librarian release --all`:

```
Analyzing all libraries for release...

Found 3 libraries with pending releases:
  - secretmanager: 1.11.0 → 1.12.0 (minor)
  - pubsub: 2.5.0 → 2.5.1 (patch)
  - spanner: 3.2.1 → 4.0.0 (major - breaking change)

Would perform releases for all 3 libraries.
To proceed, run:
  librarian release --all --execute
```

## Language-Specific Behavior

The `librarian release` command handles publishing differently for each language.

### Go Publishing

For Go libraries, no registry upload is needed. The command:
1. Verifies the tag exists
2. Notes that pkg.go.dev will automatically index it
3. Provides a tracking URL

```
secretmanager/v1.12.0
  ✓ Tag created and pushed: secretmanager/v1.12.0
  ✓ Published to pkg.go.dev (auto-indexed from tag)
  Track indexing: https://pkg.go.dev/cloud.google.com/go/secretmanager/apiv1@v1.12.0
```

### Python Publishing

For Python libraries, the command:
1. Runs tests (unless `--skip-tests`)
2. Updates version in `setup.py` or `pyproject.toml`
3. Updates `CHANGELOG.md`
4. Creates commit and tag
5. Builds distribution with `python -m build`
6. Uploads to PyPI with `twine upload`

```
google-cloud-secret-manager v1.12.0
  ✓ Ran tests
  ✓ Updated version to 1.12.0
  ✓ Updated CHANGELOG.md
  ✓ Created commit: chore(release): google-cloud-secret-manager v1.12.0
  ✓ Created tag: google-cloud-secret-manager/v1.12.0
  ✓ Pushed tag to origin
  ✓ Built distribution
  ✓ Uploaded to PyPI
  Published: https://pypi.org/project/google-cloud-secret-manager/1.12.0/
```

**Credentials**: Requires `~/.pypirc` or environment variables:
```bash
export TWINE_USERNAME=__token__
export TWINE_PASSWORD=pypi-...
```

### Rust Publishing

For Rust libraries, the command:
1. Runs tests (unless `--skip-tests`)
2. Updates version in `Cargo.toml`
3. Updates `CHANGELOG.md`
4. Creates commit and tag
5. Runs `cargo semver-checks` to validate API compatibility
6. Runs `cargo publish` to upload to crates.io

```
google-cloud-bigtable-admin-v2 v4.0.0
  ✓ Ran tests
  ✓ Updated version to 4.0.0
  ✓ Updated CHANGELOG.md
  ✓ Created commit: chore(release): google-cloud-bigtable-admin-v2 v4.0.0
  ✓ Created tag: google-cloud-bigtable-admin-v2/v4.0.0
  ✓ Pushed tag to origin
  ✓ Ran cargo semver-checks
  ✓ Published to crates.io
  Published: https://crates.io/crates/google-cloud-bigtable-admin-v2/4.0.0
```

**Credentials**: Requires `~/.cargo/credentials` or environment variable:
```bash
export CARGO_REGISTRY_TOKEN=...
```

## Tag Format

Tags follow the format: `{library}/v{version}`

Examples:
- `secretmanager/v1.12.0`
- `pubsub/v2.5.1`
- `spanner/v4.0.0`
- `google-cloud-secret-manager/v1.12.0` (Python)
- `google-cloud-bigtable-admin-v2/v4.0.0` (Rust)

## Validation

Before releasing, the command validates:

**All languages:**
- Working directory is clean
- Tests pass (unless `--skip-tests`)
- Version follows semver
- No uncommitted changes

**Python-specific:**
- Distribution builds successfully
- Version in `setup.py`/`pyproject.toml` matches tag

**Rust-specific:**
- `cargo semver-checks` passes for non-major releases
- No breaking changes in patch/minor releases

## Error Handling

**Working directory not clean:**
```
Error: Working directory is not clean.
Commit or stash your changes before releasing.
```

**Tests failed:**
```
Error: Tests failed for secretmanager
Fix the tests or use --skip-tests to bypass.
```

**Tag already exists:**
```
Error: Tag 'secretmanager/v1.12.0' already exists
If you want to re-release, delete the tag first:
  git tag -d secretmanager/v1.12.0
  git push origin :secretmanager/v1.12.0
```

**Already published (Python):**
```
Error: Package 'google-cloud-secret-manager' version '1.12.0' already exists on PyPI
Skipping publish.
```

**Semver violation (Rust):**
```
Error: cargo semver-checks failed for google-cloud-bigtable-admin-v2

Breaking changes detected in minor release:
  - Removed public function: Client::query()
  - Changed signature: Client::new() now returns Result instead of Self

This requires a major version bump (4.0.0).
Release aborted.
```

**Missing credentials:**
```
Error: PyPI credentials not found
Configure credentials in ~/.pypirc or set environment variables:
  export TWINE_USERNAME=__token__
  export TWINE_PASSWORD=pypi-...
```

## Workflows

### Standard Release Workflow

Developer makes changes and wants to release:

```bash
# 1. Make changes and commit with conventional commits
git add secretmanager/
git commit -m "feat(secretmanager): add Secret rotation support"
git push

# 2. See what would be released (dry-run)
librarian release secretmanager

# 3. Actually release
librarian release secretmanager --execute
```

### Releasing Multiple Libraries

After updating googleapis and regenerating all libraries:

```bash
# 1. Update sources and regenerate
librarian update --googleapis
librarian generate --all
git add .
git commit -m "feat: update to googleapis abc123"
git push

# 2. See what would be released
librarian release --all

# 3. Release everything
librarian release --all --execute
```

### Release Only Tags (No Registry Upload)

For testing or when you want to publish to registries manually later:

```bash
librarian release secretmanager --execute --skip-publish
```

This creates the tag and pushes it, but skips the PyPI/crates.io upload.

## Configuration

Release behavior is configured in `librarian.yaml`:

```yaml
libraries:
  - name: secretmanager
    version: 1.11.0              # Current version

release:
  tag_format: '{name}/v{version}' # Tag format for all libraries
```

## Initial Release

For libraries that have never been released (version: null):

```bash
# After creating a library with 'librarian add' or 'librarian create'
cat secretmanager/.librarian.yaml
# release:
#   version: null

# First release
librarian release secretmanager --execute
# Detects version: null → 0.1.0
```

The command automatically bumps to `0.1.0` for initial releases.

## Breaking Change Detection

The command detects breaking changes from conventional commits:

**Explicit breaking change:**
```
feat!: remove deprecated Query method
```

**Breaking change footer:**
```
feat: redesign Client API

BREAKING CHANGE: Client.New() now returns Result instead of Self
```

Both trigger a major version bump (unless current version is 0.x.x, then minor bump).

## Dry-Run Safety

By default, `librarian release` runs in dry-run mode. This is a safety feature that:
- Shows exactly what would happen
- Allows you to review the version bump
- Prevents accidental releases
- Lets you verify the changelog looks correct

You must explicitly use `--execute` to actually perform the release.
