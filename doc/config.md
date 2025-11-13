# Configuration Reference

This document describes the complete `librarian.yaml` schema.

## Overview

All configuration lives in a single `librarian.yaml` file at the repository root. This file contains:

- Repository-wide settings (language, container, sources)
- Default generation settings
- Release configuration
- Per-library configuration

## Root-Level Structure

```yaml
version: v1                    # Schema version
language: go                   # Primary language (go, python, rust)

sources:                       # External source repositories
  googleapis:
    url: https://...
    sha256: ...

container:                     # Container image configuration
  image: ...
  tag: ...

generate:                      # Generation configuration
  dir: ./                      # Default output directory

defaults:                      # Default settings for all libraries
  release_level: stable
  transport: grpc+rest

release:                       # Release configuration
  tag_format: '{name}/v{version}'

libraries:                     # Library definitions
  - name: secretmanager
    path: secretmanager/
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
```

## Field Reference

### `version`

**Type:** string
**Required:** Yes

Schema version. Currently `v1`.

```yaml
version: v1
```

### `language`

**Type:** string
**Required:** Yes
**Values:** `go`, `python`, `rust`

Primary language for this repository.

```yaml
language: go
```

### `sources`

**Type:** object
**Required:** No (but required for generation)

External source repositories used for code generation.

#### `sources.googleapis`

Configuration for googleapis repository.

**Fields:**
- `url` (string, required) - URL to googleapis tarball
- `sha256` (string, required) - SHA256 hash for verification
- `extracted_name` (string, optional) - Directory name after extraction

```yaml
sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
```

#### `sources.discovery`

Configuration for discovery-artifact-manager repository (used by Rust for Discovery-based APIs).

**Fields:**
- `url` (string, required) - URL to discovery tarball
- `sha256` (string, required) - SHA256 hash for verification
- `extracted_name` (string, optional) - Directory name after extraction

```yaml
sources:
  discovery:
    url: https://github.com/googleapis/discovery-artifact-manager/archive/xyz789.tar.gz
    sha256: 867048ec8f0850a4d77ad836319e4c0a0c624928611af8a900cd77e676164e8e
```

### `container`

**Type:** object
**Required:** No (but required for generation)

Container image configuration for code generation.

**Fields:**
- `image` (string, required) - Container registry path (without tag)
- `tag` (string, required) - Container image tag

```yaml
container:
  image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go
  tag: latest
```

### `generate`

**Type:** object
**Required:** No

Global generation settings.

**Fields:**
- `dir` (string, optional) - Default output directory for generated libraries

```yaml
generate:
  dir: ./              # Go: repository root
  # dir: packages/    # Python: packages directory
  # dir: src/generated/ # Rust: src/generated directory
```

### `defaults`

**Type:** object
**Required:** No

Default settings applied to all libraries (can be overridden per-library).

**Fields:**
- `release_level` (string, optional) - Default release level (`stable`, `preview`)
- `transport` (string, optional) - Default transport (`grpc`, `rest`, `grpc+rest`)
- `rest_numeric_enums` (boolean, optional) - Default for REST numeric enums

```yaml
defaults:
  release_level: stable
  transport: grpc+rest
  rest_numeric_enums: true
```

### `release`

**Type:** object
**Required:** No (but required for releasing)

Release configuration.

**Fields:**
- `tag_format` (string, optional) - Git tag format template. Supports `{name}` and `{version}` placeholders.
- `remote` (string, optional) - Git remote name (default: `origin`)
- `branch` (string, optional) - Git branch name (default: `main`)

```yaml
release:
  tag_format: '{name}/v{version}'  # Go: secretmanager/v1.2.0
  # tag_format: '{name}-v{version}' # Alternative: secretmanager-v1.2.0
  remote: upstream
  branch: main
```

### `libraries`

**Type:** array
**Required:** Yes

Array of library configurations.

## Library Configuration

Each library entry defines how a library is managed.

### Library Types

Libraries are classified by their configuration structure:

#### 1. Fully Handwritten

Contains only `name` and `path`. No generation.

```yaml
libraries:
  - name: pubsub
    path: pubsub/
```

#### 2. Fully Generated

Contains `name`, `path`, and `generate`. Entire directory is regenerated.

```yaml
libraries:
  - name: secretmanager
    path: secretmanager/
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
```

#### 3. Hybrid (Generated + Handwritten)

Contains `name`, `path`, `generate`, and `keep`. Specific files are protected.

```yaml
libraries:
  - name: bigquery
    path: bigquery/
    generate:
      apis:
        - path: google/cloud/bigquery/storage/v1
    keep:
      - bigquery/client.go
      - bigquery/samples/
```

### Library Fields

#### `name`

**Type:** string
**Required:** Yes

Library name. This is the package/module name used by the language ecosystem.

**Examples:**
- Go: `secretmanager`
- Python: `google-cloud-secret-manager`
- Rust: `google-cloud-secretmanager-v1`

```yaml
name: secretmanager
```

#### `path`

**Type:** string
**Required:** No (derived from `name` and `generate.dir` if not specified)

Filesystem path to the library directory (relative to repository root).

```yaml
path: secretmanager/                           # Go
# path: packages/google-cloud-secret-manager/ # Python
# path: src/generated/cloud/secretmanager/v1/ # Rust
```

#### `version`

**Type:** string
**Required:** No

Current version of the library. Updated by `librarian release`.

```yaml
version: 1.2.0
```

#### `disabled`

**Type:** boolean
**Required:** No

Temporarily disable generation for this library. Must include a comment with an issue link.

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

**Behavior:**
- `librarian generate --all` - Skips with warning
- `librarian generate aiplatform` - Returns error with issue link
- `librarian release aiplatform` - Still works (only generation is disabled)

#### `generate`

**Type:** object
**Required:** No (but required for code generation)

Generation configuration for this library.

##### `generate.apis`

**Type:** array
**Required:** Yes (if `generate` is present)

Array of API configurations. Each API represents one proto service to generate.

**API Fields:**

- `path` (string, required) - API path relative to googleapis root
  - Example: `google/cloud/secretmanager/v1`
- `service_config` (string, optional) - Service YAML filename
  - Example: `secretmanager_v1.yaml`
- `grpc_service_config` (string, optional) - gRPC retry config JSON filename
  - Example: `secretmanager_grpc_service_config.json`
- `transport` (string, optional) - Transport protocol
  - Values: `grpc`, `rest`, `grpc+rest`
- `rest_numeric_enums` (boolean, optional) - Use numeric enums in REST
- Additional language-specific fields (see language docs)

**Simple format (string):**
```yaml
generate:
  apis:
    - google/cloud/secretmanager/v1
```

**Extended format (object):**
```yaml
generate:
  apis:
    - path: google/cloud/secretmanager/v1
      service_config: secretmanager_v1.yaml
      grpc_service_config: secretmanager_grpc_service_config.json
      transport: grpc+rest
      rest_numeric_enums: true
```

**Multiple APIs:**
```yaml
generate:
  apis:
    - path: google/cloud/secretmanager/v1
    - path: google/cloud/secretmanager/v1beta2
```

##### `generate.specification_format`

**Type:** string
**Required:** No
**Values:** `protobuf`, `disco`, `openapi`, `none`
**Default:** `protobuf`

API specification format. Most APIs use `protobuf`. Some (like Compute) use `disco` (Discovery).

```yaml
generate:
  specification_format: protobuf  # Default
  # specification_format: disco   # For Discovery-based APIs
```

#### `keep`

**Type:** array (strings)
**Required:** No

Files and directories to preserve during regeneration (for hybrid libraries).

**Format:** Array of file paths or directory paths (relative to repository root).

```yaml
keep:
  - secretmanager/client.go          # Specific file (full path from repo root)
  - secretmanager/samples/           # Directory (full path from repo root)
  - secretmanager/custom/*.go        # Pattern (if supported)
```

**Use cases:**
- Handwritten client wrappers
- Custom helper functions
- Integration tests
- Documentation

## Language-Specific Configuration

Each language can have additional fields. See:
- [go.md](go.md#configuration) - Go-specific fields
- [python.md](python.md#configuration) - Python-specific fields
- [rust.md](rust.md#configuration) - Rust-specific fields

## Complete Examples

### Go Example

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
  # Fully generated library
  - name: secretmanager
    path: secretmanager/
    version: 1.2.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
        - path: google/cloud/secretmanager/v1beta2

  # Hybrid library (generated + handwritten)
  - name: bigquery
    path: bigquery/
    version: 2.5.0
    generate:
      apis:
        - path: google/cloud/bigquery/storage/v1
    keep:
      - bigquery/client.go

  # Handwritten library
  - name: pubsub
    path: pubsub/
    version: 3.0.0
```

### Python Example

```yaml
version: v1
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98

container:
  image: us-central1-docker.pkg.dev/.../librarian-python
  tag: latest

generate:
  dir: packages/

defaults:
  transport: grpc+rest
  rest_numeric_enums: true

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: google-cloud-secret-manager
    path: packages/google-cloud-secret-manager/
    version: 2.20.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          transport: grpc+rest
```

### Rust Example

```yaml
version: v1
language: rust

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
  discovery:
    url: https://github.com/googleapis/discovery-artifact-manager/archive/xyz789.tar.gz
    sha256: 867048ec8f0850a4d77ad836319e4c0a0c624928611af8a900cd77e676164e8e

container:
  image: us-central1-docker.pkg.dev/.../librarian-rust
  tag: latest

generate:
  dir: src/generated/

release:
  tag_format: '{name}/v{version}'

libraries:
  # Protobuf-based API
  - name: google-cloud-secretmanager-v1
    path: src/generated/cloud/secretmanager/v1/
    version: 1.1.0
    generate:
      specification_format: protobuf
      apis:
        - path: google/cloud/secretmanager/v1

  # Discovery-based API
  - name: google-cloud-compute-v1
    path: src/generated/cloud/compute/v1/
    version: 0.2.1
    generate:
      specification_format: disco
      apis:
        - path: discoveries/compute.v1.json
```

## Configuration Best Practices

### 1. Use Immutable Source References

Always use commit-specific URLs with SHA256 hashes:

```yaml
sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123def456.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
```

Don't use branch names like `main`:
```yaml
# Bad: not reproducible
url: https://github.com/googleapis/googleapis/archive/main.tar.gz
```

### 2. Minimize `keep` Patterns

Only add `keep` entries when you've actually added handwritten code:

```yaml
# Good: minimal keep list
keep:
  - secretmanager/client.go

# Bad: preemptive keep list
keep:
  - secretmanager/client.go
  - secretmanager/helpers.go
  - secretmanager/custom/
```

### 3. Use Defaults for Common Settings

Put common settings in `defaults` to avoid repetition:

```yaml
defaults:
  transport: grpc+rest
  rest_numeric_enums: true

libraries:
  - name: secretmanager
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          # Inherits transport and rest_numeric_enums from defaults
```

### 4. Document Disabled Libraries

Always include a comment with an issue link when disabling a library:

```yaml
# Disabled: BUILD.bazel missing required service_config
# See: https://github.com/googleapis/googleapis/issues/12345
libraries:
  - name: broken-api
    disabled: true
    generate:
      apis:
        - path: google/cloud/broken/v1
```

### 5. Keep Configuration Close to Code

The `librarian.yaml` file should live at the repository root alongside the code it configures.

### 6. CLI Flags vs YAML Fields

CLI flags use dashes while YAML fields use underscores:

```bash
# CLI flag (with dashes)
librarian create mylib --apis google/cloud/foo/v1 --specification-format protobuf

# YAML field (with underscores)
generate:
  specification_format: protobuf
```

This is a standard convention across most CLI tools (e.g., `--dry-run` flag becomes `dry_run` in config files).

## Configuration Migration

### From Old .librarian Format

The old format used three separate files:
- `.librarian/config.yaml`
- `.librarian/state.yaml`
- `.librarian/generator-input/repo-config.yaml`

The new format consolidates everything into `librarian.yaml`.

**Migration command:**
```bash
librarian convert --input .librarian/ --output librarian.yaml
```

### From Sidekick (.sidekick.toml)

Rust projects using Sidekick can migrate to librarian:

```bash
librarian convert --input .sidekick.toml --output librarian.yaml
```

## Schema Validation

Librarian validates the configuration file on every command. Common errors:

**Missing required fields:**
```
Error: librarian.yaml is missing required field 'language'
```

**Invalid language:**
```
Error: Unsupported language 'javascript'. Supported: go, python, rust
```

**Invalid API path:**
```
Error: API path 'google/cloud/invalid/v1' not found in googleapis
```

**Disabled library:**
```
Error: Library 'aiplatform' is disabled
See: https://github.com/org/repo/issues/123
```

## Next Steps

- Read [overview.md](overview.md) for CLI commands and workflows
- Read language-specific docs:
  - [go.md](go.md)
  - [python.md](python.md)
  - [rust.md](rust.md)
