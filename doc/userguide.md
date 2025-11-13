# Overview

Librarian provides a consistent CLI for managing Google Cloud client libraries across Go, Python, and Rust.

## Installation

```bash
go install github.com/julieqiu/librarian/cmd/librarian@latest
```

## Quick Start

### 1. Initialize a Repository

```bash
# Create a new repository for a specific language
librarian init go       # Creates librarian.yaml for Go
librarian init python   # Creates librarian.yaml for Python
librarian init rust     # Creates librarian.yaml for Rust
```

This creates `librarian.yaml` with language-specific defaults.

### 2. Create a Library

```bash
# Create a fully generated library
librarian create secretmanager --apis google/cloud/secretmanager/v1

# Create with explicit path (override default)
librarian create google-cloud-storage --path packages/storage/ --apis google/storage/v2
```

This command:
- Adds the library configuration to `librarian.yaml`
- Generates the client code immediately
- Creates scaffolding files (README.md, CHANGES.md, etc.)

### 3. Add More APIs

```bash
# Add another API version to an existing library
librarian add secretmanager google/cloud/secretmanager/v1beta2

# Then regenerate
librarian generate secretmanager
```

### 4. Add a Handwritten Library

```bash
# Add a library that contains only handwritten code
librarian add pubsub --path pubsub/
```

This adds a minimal entry to `librarian.yaml` without a `generate` section.

### 5. Regenerate Code

```bash
# Regenerate a specific library
librarian generate secretmanager

# Regenerate all libraries
librarian generate --all
```

### 6. Release a Library

```bash
# Dry-run (shows what would happen)
librarian release secretmanager

# Actually perform the release
librarian release secretmanager --execute
```

## Library Types

Librarian supports three types of libraries based on their `librarian.yaml` configuration:

### Fully Handwritten

Contains only `name` and `path`. The generator never touches this directory.

```yaml
libraries:
  - name: pubsub
    path: pubsub/
```

### Fully Generated

Contains `name`, `path`, and `generate` block. The generator deletes and recreates the entire directory on every run.

```yaml
libraries:
  - name: secretmanager
    path: secretmanager/
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
```

### Hybrid (Generated + Handwritten)

Contains `name`, `path`, `generate`, and `keep` block. The `keep` block protects specific files from being overwritten.

```yaml
libraries:
  - name: bigquery
    path: bigquery/
    generate:
      apis:
        - path: google/cloud/bigquery/storage/v1
    keep:
      - bigquery/client.go      # Protected from regeneration
      - bigquery/samples/       # Protected directory
```

## Commands

### `librarian init <language>`

Initialize a new repository with language-specific configuration.

```bash
librarian init go
librarian init python
librarian init rust
```

Creates `librarian.yaml` with:
- Language setting
- Default directory for generated code
- Container image reference
- Release tag format
- googleapis source reference

### `librarian create <name> --apis <api-paths...>`

Create a new library and generate its code immediately.

```bash
# Simple case (uses default path)
librarian create secretmanager --apis google/cloud/secretmanager/v1

# Multiple APIs
librarian create secretmanager --apis google/cloud/secretmanager/v1 google/cloud/secretmanager/v1beta2

# Custom path
librarian create google-cloud-storage --path packages/storage/ --apis google/storage/v2
```

This is syntactic sugar for `librarian add` + `librarian generate`.

### `librarian add <name> [apis...]`

Add or update a library configuration in `librarian.yaml`.

```bash
# Add handwritten library (no APIs)
librarian add pubsub --path pubsub/

# Add generated library with APIs
librarian add secretmanager google/cloud/secretmanager/v1

# Add more APIs to existing library
librarian add secretmanager google/cloud/secretmanager/v1beta2
```

Note: After adding APIs, run `librarian generate <name>` to generate the code.

### `librarian generate <name|--all>`

Generate or regenerate client library code.

```bash
# Generate specific library
librarian generate secretmanager

# Generate all libraries
librarian generate --all
```

The generation process:
1. Downloads googleapis (if needed) and verifies SHA256
2. Extracts API configuration from BUILD.bazel
3. Runs the container to generate code
4. Applies `keep` rules to preserve handwritten files
5. Copies generated code to the library directory

### `librarian release <name|--all> [--execute]`

Release one or more libraries.

```bash
# Dry-run (show what would happen)
librarian release secretmanager

# Actually perform the release
librarian release secretmanager --execute

# Release all changed libraries
librarian release --all --execute

# Skip tests
librarian release secretmanager --execute --skip-tests

# Skip publishing (only create tags)
librarian release secretmanager --execute --skip-publish
```

The release process:
1. Analyzes conventional commits to determine version bump
2. Updates version files
3. Updates changelog
4. Creates a commit
5. Creates git tags
6. Pushes tags to remote
7. Publishes to package registries (PyPI, crates.io, or auto-indexed by pkg.go.dev)

### `librarian remove <name> [apis...]`

Remove a library or specific APIs from a library.

```bash
# Remove entire library
librarian remove secretmanager

# Remove specific API from library
librarian remove secretmanager google/cloud/secretmanager/v1beta2
```

### `librarian update --googleapis [--discovery]`

Update the googleapis (or discovery) source reference to the latest commit.

```bash
# Update googleapis source
librarian update --googleapis

# Update discovery source
librarian update --discovery

# Update both
librarian update --googleapis --discovery
```

This updates the `sources` section in `librarian.yaml` with the latest commit SHA and tarball URL.

## Configuration File

All configuration lives in a single `librarian.yaml` file at the repository root.

Example for Go:

```yaml
version: v1
language: go

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98

container:
  image: us-central1-docker.pkg.dev/.../librarian-go
  tag: latest

generate:
  dir: ./

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: secretmanager
    path: secretmanager/
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          # Additional configuration extracted from BUILD.bazel
```

See [config.md](config.md) for the complete schema reference.

## Container Architecture

Librarian uses a command-based container architecture:

1. **Host (librarian CLI)** prepares explicit commands to execute
2. **Container** receives `/commands/commands.json` and executes each command
3. **Multiple invocations** per library generation:
   - Phase 1: Code generation (run protoc/generator)
   - Phase 2: Formatting and build
   - Phase 3: Testing and validation

Benefits:
- Container is language-agnostic (simple command executor)
- Commands are explicit and debuggable
- Easy to add new phases without changing container code

See language-specific docs for details:
- [go.md](go.md) - Go generation details
- [python.md](python.md) - Python generation details
- [rust.md](rust.md) - Rust generation details

## Disabling Broken Libraries

If a library's generation is broken, you can temporarily disable it:

```yaml
# Disabled: Generator fails on proto field validation
# See: https://github.com/org/repo/issues/123
libraries:
  - name: aiplatform
    disabled: true
    generate:
      apis:
        - path: google/cloud/aiplatform/v1
```

Behavior:
- `librarian generate --all` - Skips aiplatform with a warning
- `librarian generate aiplatform` - Returns error with issue link
- `librarian release aiplatform` - Still works (only generation is disabled)

Requirements:
- Must include a comment with issue link explaining why it's disabled

## Workflows

### Creating a New Library

```bash
# 1. Create library with initial API
librarian create secretmanager --apis google/cloud/secretmanager/v1

# 2. (Optional) Mark files to protect from regeneration
# Edit librarian.yaml to add 'keep' section:
# keep:
#   - secretmanager/client.go

# 3. Regenerate to verify keep rules work
librarian generate secretmanager

# 4. Release when ready
librarian release secretmanager --execute
```

### Adding API Versions

```bash
# 1. Add new API version
librarian add secretmanager google/cloud/secretmanager/v1beta2

# 2. Regenerate code
librarian generate secretmanager

# 3. Release
librarian release secretmanager --execute
```

### Updating googleapis

```bash
# 1. Update googleapis reference
librarian update --googleapis

# 2. Regenerate all libraries to test
librarian generate --all

# 3. Review changes
git diff

# 4. Commit if everything looks good
git add librarian.yaml
git commit -m "chore: update googleapis to latest"
```

### Converting from Handwritten to Hybrid

```bash
# 1. Currently handwritten library exists in librarian.yaml:
#    - name: pubsub
#      path: pubsub/

# 2. Add APIs and 'keep' section
librarian add pubsub google/cloud/pubsub/v1

# 3. Edit librarian.yaml to add keep rules for handwritten files:
#    keep:
#      - pubsub/client.go
#      - pubsub/custom/

# 4. Generate (will preserve keep files)
librarian generate pubsub
```

## Common Patterns

### Monorepo (Go)

```yaml
generate:
  dir: ./  # Libraries at repository root

libraries:
  - name: secretmanager
    path: secretmanager/
  - name: pubsub
    path: pubsub/
```

### Package Directory (Python)

```yaml
generate:
  dir: packages/  # All libraries in packages/

libraries:
  - name: google-cloud-secret-manager
    path: packages/google-cloud-secret-manager/
  - name: google-cloud-pubsub
    path: packages/google-cloud-pubsub/
```

### API-Versioned Paths (Rust)

```yaml
generate:
  dir: src/generated/

libraries:
  - name: cloud-secretmanager-v1
    path: src/generated/cloud/secretmanager/v1/
```

## Error Handling

Librarian provides clear, actionable error messages:

```
Error: Library 'secretmanager' not found in librarian.yaml

Available libraries:
  - pubsub
  - storage

To create a new library: librarian create secretmanager --apis <api-path>
```

```
Error: API path 'google/cloud/invalid/v1' not found in googleapis

Verify the API path at:
  https://github.com/googleapis/googleapis/tree/main/google/cloud
```

```
Error: Generation failed for secretmanager
  → Container exited with code 1
  → See logs above for details

To debug:
  1. Check the BUILD.bazel file for the API
  2. Verify googleapis sources are up to date: librarian update --googleapis
  3. Check container logs for protoc errors
```

## Next Steps

- Read [config.md](config.md) for complete configuration reference
- Read language-specific guides:
  - [go.md](go.md) for Go-specific features
  - [python.md](python.md) for Python-specific features
  - [rust.md](rust.md) for Rust-specific features
- Read [alternatives.md](alternatives.md) for design decisions and alternatives considered
