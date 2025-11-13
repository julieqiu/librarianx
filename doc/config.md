# Configuration Reference

This document describes the `librarian.yaml` configuration schema.

## Overview

Librarian uses a **minimal configuration** approach. It automatically discovers APIs from the googleapis repository and generates libraries using language-specific conventions.

Configuration defines:
- Repository-wide settings (language, sources)
- Default generation settings
- Release configuration
- Library-specific overrides

## Quick Start

### Minimal Python Configuration

```yaml
version: v1
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  output: packages/
  one_library_per: service
  transport: grpc+rest

release:
  tag_format: '{name}/v{version}'

libraries:
  - '*'  # Generate all discovered APIs

  # Exception: has handwritten code
  - google-cloud-bigquery-storage:
      api: google/cloud/bigquery/storage/v1
      keep:
        - google/cloud/bigquery_storage_v1/client.py
```

**Result**: Generates ~200+ libraries, only 1 needs explicit config.

### Selective Generation (Dart)

```yaml
version: v1
language: dart

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  output: generated/
  one_library_per: version

release:
  tag_format: '{name}/v{version}'

libraries:
  # Explicitly list what to generate
  - google_api:
      api: google/api
  - google_iam_v1:
      api: google/iam/v1
  - google_cloud_secretmanager_v1:
      api: google/cloud/secretmanager/v1
  - google_cloud_functions_v2:
      api: google/cloud/functions/v2
```

**Result**: Only generates the 4 explicitly listed libraries.

---

## Root-Level Fields

### `version`

**Type:** string
**Required:** Yes
**Value:** `v1`

Schema version. Currently `v1`.

```yaml
version: v1
```

---

### `language`

**Type:** string
**Required:** Yes
**Values:** `go`, `python`, `rust`, `dart`

Primary language for this repository.

```yaml
language: python
```

---

### `sources`

**Type:** object
**Required:** Yes

External source repositories for code generation.

#### `sources.googleapis`

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

---

### `defaults`

**Type:** object
**Required:** No

Default settings applied to all libraries.

The precendence is this:
- Library-level overrides
- Language defaults
- Defaults block

**Common fields:**
- `output` (string) - Output directory for generated code
- `one_library_per` (string) - Bundling strategy: `service` or `version`
- `transport` (string) - Default transport: `grpc`, `rest`, `grpc+rest`
- `rest_numeric_enums` (boolean) - Use numeric enums in REST

```yaml
defaults:
  output: packages/
  one_library_per: service    # Bundle all versions into one library (Python/Go)
  transport: grpc+rest
  rest_numeric_enums: true
```

#### `one_library_per` Explained

**`one_library_per: service`** (Python/Go default)
- All versions of a service → one library
- Example: `google/cloud/vision/v1` and `google/cloud/vision/v1beta` → `packages/google-cloud-vision/`

**`one_library_per: version`** (Rust/Dart default)
- Each version → separate library
- Example: `google/cloud/vision/v1` → `src/generated/google-cloud-vision-v1/`
- Example: `google/cloud/vision/v1beta` → `src/generated/google-cloud-vision-v1beta/`

---

### `release`

**Type:** object
**Required:** No (but required for releasing)

Release configuration.

**Fields:**
- `tag_format` (string) - Git tag format. Supports `{name}` and `{version}` placeholders.
- `remote` (string, optional) - Git remote name (default: `origin`)
- `branch` (string, optional) - Git branch name (default: `main`)

```yaml
release:
  tag_format: '{name}/v{version}'
  remote: upstream
  branch: main
```

---

### `libraries`

**Type:** array
**Required:** No

Library configurations. Each library is identified by its **name** (the package name published to registries).

**Two modes:**

If the libraries array contains `'*'`, librarian auto-discovers APIs from googleapis and generates them with default settings. Otherwise, only explicitly listed libraries are generated.

**1. Generate everything (wildcard mode)**
```yaml
libraries:
  - '*'  # Auto-discover and generate all APIs

  # Add config to specific libraries
  - google-cloud-vision:
      api: google/cloud/vision/v1
      keep: [...]
```

**2. Generate only listed libraries (explicit mode)**
```yaml
libraries:
  # Must specify api for each generated library
  - google-cloud-secretmanager:
      api: google/cloud/secretmanager/v1
  - google-cloud-vision:
      api: google/cloud/vision/v1
  - google-cloud-translate:
      api: google/cloud/translate/v3
```

**Key principle:** Wildcard discovers APIs and applies defaults. Explicit entries use the same defaults plus your overrides. Being explicit doesn't change behavior—it lets you add configuration.

---

## Library Configuration

### Identifying Libraries

Libraries use their **name** (package name) as the YAML key. The `api` field specifies which googleapis API to generate from.

**Generated libraries** (from googleapis):
```yaml
- google-cloud-vision:      # Python package name
    api: google/cloud/vision/v1    # googleapis API path

- secretmanager:            # Go module name
    api: google/cloud/secretmanager/v1

- google-cloud-vision-v1:   # Rust crate name
    api: google/cloud/vision/v1
```

**Handwritten libraries** (no googleapis API):

Handwritten libraries have no `api` field. They must always be listed explicitly.

```yaml
- pubsub:
    path: pubsub         # Source directory

- auth:
    path: auth

- compute-metadata:
    path: compute/metadata
```

### Minimal Syntax

For generated libraries that follow all defaults:

```yaml
libraries:
  - google-cloud-secretmanager:
      api: google/cloud/secretmanager/v1

  - google-cloud-vision:
      api: google/cloud/vision/v1
```

Output locations are computed from `defaults.output` + API path.

### With Overrides

Add configuration to libraries as needed:

```yaml
libraries:
  # Generated with handwritten additions
  - google-cloud-vision:
      api: google/cloud/vision/v1
      keep:
        - google/cloud/vision_v1/helpers.py
        - tests/unit/test_helpers.py

  # Generated with path override
  - google-cloud-firestore:
      api: google/cloud/firestore/v1
      path: src/firestore/src/generated/gapic
      rust:
        template_override: templates/grpc-client

  # Handwritten
  - pubsub:
      path: pubsub
      release:
        disabled: true
```

---

## Library Fields

Library entries use the **library name** as the YAML key. This is the package
name published to registries (PyPI, crates.io, pub.dev, etc.).

```yaml
- google-cloud-secretmanager:  # Python: package name
- secretmanager:               # Go: module name
- google-cloud-vision-v1:      # Rust: crate name
- google_cloud_functions_v2:   # Dart: package name
```

---

### `api`

**Type:** string or object
**Required:** Yes (for generated libraries)

The googleapis API path to generate from. Can be a simple string or an object for non-protobuf APIs.

**Simple syntax (protobuf APIs):**
```yaml
- name: google-cloud-secretmanager
  api: google/cloud/secretmanager/v1
```

**Object syntax (discovery APIs):**
```yaml
- name: google-cloud-compute-v1
  api:
    path: discoveries/compute.v1.json
    source: discovery
```

---

### `apis`

**Type:** array of strings
**Required:** No (use instead of `api` for multi-API libraries)

For libraries that bundle multiple API versions.

```yaml
- name: google-cloud-vision
  apis:
    - google/cloud/vision/v1
    - google/cloud/vision/v1p1beta1
```

---

### `path`

**Type:** string
**Required:** No

Explicit filesystem path override. If not specified, the path is computed from `defaults.output` + API path.

```yaml
# Uses defaults.output pattern
- name: google-cloud-aiplatform-v1
  api: google/cloud/aiplatform/v1
  # → generates to: src/generated/cloud/aiplatform/v1

# Explicit path override
- name: google-cloud-firestore
  api: google/cloud/firestore/v1
  path: src/firestore/src/generated/gapic
  # → generates to: src/firestore/src/generated/gapic
```

For handwritten libraries, `path` specifies the source directory:
```yaml
- name: auth
  path: auth
```

---

### `keep`

**Type:** array (strings)
**Required:** No

Files and directories to preserve during regeneration (for hybrid libraries with handwritten code).

```yaml
keep:
  - google/cloud/bigquery_storage_v1/client.py     # Specific file
  - google/cloud/bigquery_storage_v1/samples/      # Directory
  - tests/unit/test_*.py                           # Pattern
```

### `disabled`

**Type:** boolean
**Required:** No

Disable generation for this library.

```yaml
- packages/google-cloud-broken:
    disabled: true
    reason: "Missing required BUILD.bazel service_config"
```

---

## Complete Examples

### Python with Wildcard

```yaml
version: v1
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  output: packages/
  one_library_per: service
  transport: grpc+rest

release:
  tag_format: '{name}/v{version}'

libraries:
  - '*'

  # Exception: handwritten code
  - packages/google-cloud-vision:
      keep:
        - google/cloud/vision_v1/helpers.py

  - packages/google-cloud-bigquery-storage:
      keep:
        - google/cloud/bigquery_storage_v1/client.py

  # Handwritten libraries
  - pubsub/
  - auth/
```

---

### Go with Wildcard

```yaml
version: v1
language: go

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  output: ./
  one_library_per: service
  transport: grpc+rest

release:
  tag_format: '{name}/v{version}'

libraries:
  - '*'

  # Exception: handwritten IAM client
  - batch:
      keep:
        - ^batch/apiv1/iam_policy_client\.go$

  # Handwritten libraries
  - pubsub/
  - storage/
```

---

### Rust with Version-Level Packaging

```yaml
version: v1
language: rust

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  output: src/generated/
  one_library_per: version
  release_level: stable

release:
  tag_format: '{name}/v{version}'

libraries:
  - '*'

  # Exception: special config
  - src/generated/google-cloud-aiplatform-v1beta1:
      rust:
        per_service_features: true
```

---

### Dart Explicit Mode

```yaml
version: v1
language: dart

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  output: generated/
  one_library_per: version

release:
  tag_format: '{name}/v{version}'

libraries:
  # Common protos
  - generated/google_api
  - generated/google_iam_v1

  # Service APIs
  - generated/google_cloud_secretmanager_v1
  - generated/google_cloud_functions_v2
```

---

## How Discovery Works

When `libraries` contains `'*'`:

1. **Scan googleapis** - Find all directories matching version patterns (`v1`, `v1beta`, etc.)
2. **Derive library names** - Use language-specific conventions
3. **Compute output paths** - Apply `output` template
4. **Match configurations** - Apply any matching library configs
5. **Generate** - Create libraries at computed paths

**Example for Python:**
```
Discovered: google/cloud/secretmanager/v1
Derive name: google-cloud-secretmanager
Compute path: packages/google-cloud-secretmanager
Check config: Match found? Apply settings.
Generate: Create packages/google-cloud-secretmanager/
```

---

## Name Derivation Rules

Library names are derived from API paths using language conventions:

### Python (one_library_per: service)

```
API path: google/cloud/secretmanager/v1
Name:     google-cloud-secretmanager
Path:     packages/google-cloud-secretmanager/
```

### Go (one_library_per: service)

```
API path: google/cloud/secretmanager/v1
Name:     secretmanager
Path:     secretmanager/
```

### Rust (one_library_per: version)

```
API path: google/cloud/secretmanager/v1
Name:     google-cloud-secretmanager-v1
Path:     src/generated/google-cloud-secretmanager-v1/
```

### Dart (one_library_per: version)

```
API path: google/cloud/secretmanager/v1
Name:     google_cloud_secretmanager_v1
Path:     generated/google_cloud_secretmanager_v1/
```

---

## Configuration Best Practices

### 1. Use Wildcard for Large Repos

```yaml
# Good: generate everything
libraries:
  - '*'

# Only list exceptions
  - packages/google-cloud-vision:
      keep: [...]
```

### 2. Use Immutable Source References

```yaml
sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123def456.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
```

### 3. Minimal Keep Lists

Only add `keep` entries when you've actually added handwritten code.

### 4. Document Disabled Libraries

```yaml
- packages/google-cloud-broken:
    disabled: true
    reason: "BUILD.bazel missing required service_config"
```

---

## See Also

- [README.md](../README.md) - User guide and workflows
- [alternatives.md](alternatives.md) - Design alternatives
