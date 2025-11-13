# Go Generation

This document describes Go-specific features and configuration for Librarian.

## Prerequisites

Go generation requires:

- Go 1.23 or later
- `protoc` (Protocol Buffer compiler)
- `protoc-gen-go` (Go protocol buffer plugin)
- `protoc-gen-go-grpc` (Go gRPC plugin)
- `protoc-gen-go_gapic` (Google API client generator for Go)

These are included in the container, so no manual installation is required when using containers.

## Configuration

Go libraries use the standard configuration format with Go-specific extensions.

### Basic Example

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
  dir: ./  # Go uses repository root

defaults:
  transport: grpc+rest

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: secretmanager
    path: secretmanager/
    version: 1.2.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
```

### Go-Specific API Fields

API configurations can include Go-specific fields extracted from BUILD.bazel:

```yaml
libraries:
  - name: secretmanager
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          service_config: secretmanager_v1.yaml
          grpc_service_config: secretmanager_grpc_service_config.json
          transport: grpc+rest
          rest_numeric_enums: true
          importpath: cloud.google.com/go/secretmanager/apiv1;secretmanager
          release_level: ga
```

**Fields:**
- `importpath` (string) - Go import path with package name
  - Format: `{module_path}/{api_version};{package_name}`
  - Example: `cloud.google.com/go/secretmanager/apiv1;secretmanager`
- `release_level` (string) - Release level (`ga`, `beta`, `alpha`)

### Go-Specific Library Fields

Libraries can include Go-specific configuration:

```yaml
libraries:
  - name: secretmanager
    go:
      source_roots:
        - secretmanager
        - internal/generated/snippets/secretmanager
      keep:
        - secretmanager/apiv1/iam_policy_client.go
      remove_regex:
        - ^internal/generated/snippets/secretmanager/
        - ^secretmanager/apiv1/[^/]*_client\.go$
        - ^secretmanager/apiv1/doc\.go$
      module_path_version: /v2
```

**Fields:**
- `go.source_roots` (array) - Directories containing source code
- `go.keep` (array) - Files to preserve during generation (explicit file list)
- `go.remove_regex` (array) - Files to remove after generation (regex patterns)
- `go.module_path_version` (string) - Module version suffix (e.g., `/v2`)

### Library Naming Conventions

Go library names follow these conventions:

- **google.cloud APIs**: Use service name
  - Example: `google/cloud/secretmanager/v1` → `secretmanager`
- **Other APIs**: Use second-to-last path component
  - Example: `google/bigtable/admin/v2` → `admin`

### Directory Structure

Go repositories typically use a monorepo structure at the root:

```
repository/
├── librarian.yaml
├── secretmanager/
│   ├── apiv1/
│   │   ├── secret_manager_service_client.go
│   │   ├── doc.go
│   │   └── secretmanagerpb/
│   ├── go.mod
│   ├── go.sum
│   ├── README.md
│   ├── CHANGES.md
│   └── internal/
│       └── version.go
├── pubsub/
│   ├── apiv1/
│   ├── go.mod
│   └── ...
└── internal/
    └── generated/
        └── snippets/
            ├── secretmanager/
            └── pubsub/
```

## Generation Process

Go generation follows this workflow:

### Phase 1: Code Generation

The container receives commands to run protoc with Go plugins:

```json
{
  "commands": [
    {
      "command": "protoc",
      "args": [
        "--proto_path=/source",
        "--go_out=/output",
        "--go-grpc_out=/output",
        "--go_gapic_out=/output",
        "--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
        "--go_gapic_opt=grpc-service-config=/source/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
        "--go_gapic_opt=api-service-config=/source/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
        "--go_gapic_opt=transport=grpc+rest",
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

**Generated files:**
- Client code (`*_client.go`)
- Protocol buffer types (`secretmanagerpb/*.pb.go`)
- gRPC service definitions
- Documentation (`doc.go`)
- Examples (`*_example_test.go`)
- Metadata (`gapic_metadata.json`)

### Phase 2: Formatting and Build

```json
{
  "commands": [
    {
      "command": "goimports",
      "args": ["-w", "."]
    },
    {
      "command": "go",
      "args": ["mod", "init", "cloud.google.com/go/secretmanager"]
    },
    {
      "command": "go",
      "args": ["mod", "tidy"]
    }
  ]
}
```

### Phase 3: Testing

```json
{
  "commands": [
    {
      "command": "go",
      "args": ["build", "./..."]
    },
    {
      "command": "go",
      "args": ["test", "./...", "-short"]
    }
  ]
}
```

### Host Responsibilities

After the container exits, the librarian CLI:

1. Applies `go.remove_regex` file filtering patterns
2. Applies `go.keep` file preservation rules
3. Copies generated code to the library path

## File Filtering

### Automatic Removal Patterns

Most Go libraries follow a consistent pattern for files to remove after generation. These are specified in `go.remove_regex`:

**Common patterns:**
```yaml
go:
  remove_regex:
    # Snippets directory
    - ^internal/generated/snippets/secretmanager/

    # Generated client files
    - ^secretmanager/apiv1/[^/]*_client\.go$
    - ^secretmanager/apiv1/[^/]*_client_example_go123_test\.go$
    - ^secretmanager/apiv1/[^/]*_client_example_test\.go$

    # Auxiliary files
    - ^secretmanager/apiv1/auxiliary\.go$
    - ^secretmanager/apiv1/auxiliary_go123\.go$

    # Documentation and metadata
    - ^secretmanager/apiv1/doc\.go$
    - ^secretmanager/apiv1/gapic_metadata\.json$
    - ^secretmanager/apiv1/helpers\.go$

    # Protocol buffer generated files
    - ^secretmanager/apiv1/secretmanagerpb/.*$

    # Metadata
    - ^secretmanager/apiv1/\.repo-metadata\.json$
```

### Preserving Files (Hybrid Libraries)

For hybrid libraries with handwritten code, use `go.keep`:

```yaml
libraries:
  - name: batch
    go:
      keep:
        - batch/apiv1/iam_policy_client.go  # Handwritten IAM wrapper
```

## Scaffolding Files

On first generation, librarian creates these scaffolding files:

### 1. README.md

```markdown
# Secret Manager API

[Product Documentation](https://cloud.google.com/secret-manager)

## Installation

\```bash
go get cloud.google.com/go/secretmanager
\```
```

### 2. CHANGES.md

```markdown
# Changes
```

### 3. internal/version.go

```go
package internal

// Version is the current version of the secretmanager client library.
const Version = "0.0.0"
```

### 4. {clientDir}/version.go

```go
package apiv1

import "cloud.google.com/go/secretmanager/internal"

// version is the version of this client library.
var version = internal.Version
```

### 5. internal/generated/snippets/go.mod

Updates with replace directive:

```go
replace cloud.google.com/go/secretmanager => ../../../secretmanager
```

## Release Process

Go releases follow the standard librarian release workflow with Go-specific implementation details.

### Overview

Go uses a two-command release process:

1. **`librarian release`** - Prepares files, creates commit, tags, and pushes
2. **`librarian publish`** - Verifies pkg.go.dev indexing (optional)

### Version Files

Librarian updates these files during release:

- `internal/version.go` - Module-level version constant
- `CHANGES.md` - Changelog with new version entries
- `internal/generated/snippets/{library}/apiv*/snippet_metadata.*.json` - Snippet metadata files

### Go-Specific Implementation

**For the complete release workflow, see [doc/release.md](release.md#implementation-architecture).**

Go-specific logic is implemented in `internal/release/golang.go`:

```go
// Release performs Go-specific release preparation
func Release(ctx context.Context, lib *config.Library, version string, changes []*Change) error {
    // 1. Run Go tests
    if err := runGoTests(ctx, lib.Path); err != nil {
        return err
    }

    // 2. Update CHANGES.md with Google Cloud Go format
    if err := updateChangelog(lib, version, changes); err != nil {
        return err
    }

    // 3. Update internal/version.go
    if err := updateVersionFile(lib, version); err != nil {
        return err
    }

    // 4. Update snippet metadata JSON files
    if err := updateSnippetMetadata(lib, version); err != nil {
        return err
    }

    return nil
}

// Publish verifies pkg.go.dev indexing
func Publish(ctx context.Context, lib *config.Library) error {
    return verifyPkgGoDev(lib)
}
```

#### 1. Running Tests

```go
func runGoTests(ctx context.Context, libPath string) error {
    cmd := exec.CommandContext(ctx, "go", "test", "./...")
    cmd.Dir = libPath
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("tests failed:\n%s", output)
    }
    return nil
}
```

#### 2. Updating CHANGES.md

Groups changes by type and generates Google Cloud Go changelog format (see [Changelog Format](#changelog-format) below).

#### 3. Updating internal/version.go

Generates version file from template:

```go
package internal

// Version is the current version of this client.
const Version = "1.16.0"
```

#### 4. Updating Snippet Metadata

Finds and updates all `snippet_metadata.*.json` files, replacing the version field

### Tag Format

Go uses module-path-based tags:

```yaml
release:
  tag_format: '{name}/v{version}'
```

Examples:
- `secretmanager/v1.16.0`
- `pubsub/v2.5.1`
- `spanner/v4.0.0`
- `secretmanager/v1.16.0-rc.1` (pre-release)

### Module Versioning (v2+)

For v2+ modules, specify `module_path_version`:

```yaml
libraries:
  - name: bigquery
    go:
      module_path_version: /v2
```

This creates tags like `bigquery/v2.0.0` and uses import path `cloud.google.com/go/bigquery/v2`.

### Changelog Format

Go uses Google Cloud Go changelog style:

```markdown
# Changes

## [1.16.0](https://github.com/googleapis/google-cloud-go/releases/tag/secretmanager%2Fv1.16.0) (2025-11-12)

### Features

* add Secret rotation support ([abc1234](https://github.com/googleapis/google-cloud-go/commit/abc1234...))
* another feature ([def5678](https://github.com/googleapis/google-cloud-go/commit/def5678...))

### Bug Fixes

* handle nil pointers correctly ([123456a](https://github.com/googleapis/google-cloud-go/commit/123456a...))

### Performance Improvements

* optimize client creation ([789abcd](https://github.com/googleapis/google-cloud-go/commit/789abcd...))

### Documentation

* update README examples ([fedcba9](https://github.com/googleapis/google-cloud-go/commit/fedcba9...))

## [1.15.0]...
```

**Features:**
- Release URL in section header
- Date in ISO format
- Grouped by type (Features, Bug Fixes, Performance Improvements, Reverts, Documentation)
- Commit links with shortened hash
- Alphabetically sorted within sections

### Publishing

Go libraries are automatically indexed by pkg.go.dev when tags are pushed. No manual publishing step is required.

```bash
# Release creates and pushes tag
librarian release secretmanager --execute

# Verify indexing (optional)
librarian publish secretmanager --execute
```

**What publish does for Go:**

1. Finds latest tag: `secretmanager/v1.16.0`
2. Verifies tag exists in remote
3. Checks pkg.go.dev indexing status
4. Prints tracking URL

**Output:**
```
Publishing secretmanager...

✓ Found tag: secretmanager/v1.16.0
✓ Tag exists in remote
✓ Published to pkg.go.dev (auto-indexed)

Track: https://pkg.go.dev/cloud.google.com/go/secretmanager/apiv1@v1.16.0

Note: pkg.go.dev indexes new tags within a few minutes.
```

### Pre-Release Versions

Go supports pre-release versions in semver format:

```bash
# Create release candidate
librarian release secretmanager --execute --version 1.16.0-rc.1

# Tag: secretmanager/v1.16.0-rc.1
# Import: go get cloud.google.com/go/secretmanager@v1.16.0-rc.1
```

Pre-release versions require explicit version in `go.mod`:
```bash
go get cloud.google.com/go/secretmanager/apiv1@v1.16.0-rc.1
```

### Root Module Libraries

For single-module repositories (like gapic-generator-go), use special handling:

```yaml
libraries:
  - name: root-module  # Special ID
    path: ./
    go:
      source_roots: ["."]
```

Files are placed in repository root:
- `CHANGES.md` (not `{library}/CHANGES.md`)
- `internal/version.go` (not `{library}/internal/version.go`)

## Container Architecture

The Go container includes:

- Go 1.23
- `protoc` (Protocol Buffer compiler)
- `protoc-gen-go` (Go protocol buffer plugin)
- `protoc-gen-go-grpc` (Go gRPC plugin)
- `protoc-gen-go_gapic` (Google API client generator for Go)
- `goimports` (Go import formatter)

The container is a simple command executor that reads `/commands/commands.json` and executes each command sequentially.

## Common Workflows

### Creating a New Go Library

```bash
# 1. Initialize Go repository (if not already done)
librarian init go

# 2. Create library with initial API
librarian create secretmanager --apis google/cloud/secretmanager/v1

# 3. Verify generation worked
ls secretmanager/
```

### Adding a New API Version

```bash
# 1. Add new API version to existing library
librarian add secretmanager google/cloud/secretmanager/v1beta2

# 2. Regenerate code
librarian generate secretmanager
```

### Creating a Hybrid Library

```bash
# 1. Create library
librarian create batch --apis google/cloud/batch/v1

# 2. Add handwritten code
# Edit batch/apiv1/iam_policy_client.go

# 3. Add keep rules to librarian.yaml
# go:
#   keep:
#     - batch/apiv1/iam_policy_client.go

# 4. Regenerate to verify keep rules work
librarian generate batch
```

## Troubleshooting

### protoc-gen-go_gapic not found

```
Error: protoc-gen-go_gapic: program not found or is not executable
```

**Solution:** Install gapic-generator-go:
```bash
go install github.com/googleapis/gapic-generator-go/cmd/protoc-gen-go_gapic@latest
```

### go.mod conflicts

```
Error: go: finding module for package cloud.google.com/go/secretmanager/apiv1
```

**Solution:** Run `go mod tidy` in the library directory:
```bash
cd secretmanager/
go mod tidy
```

### Import cycle detected

```
Error: import cycle not allowed
```

**Possible causes:**
1. Handwritten code imports generated code that imports handwritten code
2. Multiple API versions with circular dependencies

**Solution:** Restructure imports to break the cycle, or use internal packages.

## Best Practices

### 1. Use Consistent Naming

Follow Go module naming conventions:

```yaml
# Good: matches go.mod module path
name: secretmanager

# Bad: inconsistent with module path
name: secret-manager
```

### 2. Minimal keep Lists

Only add `go.keep` entries for actual handwritten code:

```yaml
# Good: only handwritten files
go:
  keep:
    - batch/apiv1/iam_policy_client.go

# Bad: protecting generated files
go:
  keep:
    - batch/apiv1/*.go
```

### 3. Use remove_regex Patterns

Go remove patterns follow a consistent structure. Let the generator handle these automatically when possible.

### 4. Test Before Releasing

Always run tests before releasing:

```bash
# Test locally
librarian generate secretmanager
cd secretmanager
go test ./...

# Then release
librarian release secretmanager --execute
```

## Next Steps

- Read [overview.md](overview.md) for general CLI usage
- Read [config.md](config.md) for complete configuration reference
- Read [alternatives.md](alternatives.md) for design decisions
