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
  generate: all

release:
  tag_format: '{name}/v{version}'

libraries:
  # Add settings for specific libraries
  - name: google-cloud-bigquery-storage
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
  generate: explicit

release:
  tag_format: '{name}/v{version}'

libraries:
  # Explicitly list what to generate
  - name: google_api
    api: google/api
  - name: google_iam_v1
    api: google/iam/v1
  - name: google_cloud_secretmanager_v1
    api: google/cloud/secretmanager/v1
  - name: google_cloud_functions_v2
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
- `generate` (string) - Generation mode: `all` or `explicit`
- `transport` (string) - Default transport: `grpc`, `rest`, `grpc+rest`
- `rest_numeric_enums` (boolean) - Use numeric enums in REST

```yaml
defaults:
  output: packages/
  one_library_per: service    # Bundle all versions into one library (Python/Go)
  generate: all               # Auto-discover and generate all APIs
  transport: grpc+rest
  rest_numeric_enums: true
```

#### `generate` Explained

**`generate: all`** (default for large repos)
- Auto-discovers all APIs from googleapis
- Generates all discovered APIs using language-specific naming conventions
- Libraries can still be configured in the `libraries` section for additional settings

**`generate: explicit`**
- Only generates libraries explicitly listed in the `libraries` section
- Each library must specify its `api` field
- Useful for smaller repos or when you want tight control over what's generated

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

### `name_overrides`

**Type:** array
**Required:** No

Override the auto-derived library names for specific APIs.

**Fields:**
- `api` (string, required) - The googleapis API path
- `name` (string, required) - The library name to use instead of the auto-derived name

```yaml
name_overrides:
  - api: google/api/apikeys/v2
    name: google-api-keys  # Instead of auto-derived "google-api-apikeys"

  - api: google/cloud/bigquery/storage/v1
    name: google-cloud-bigquery-storage  # Instead of "google-cloud-bigquerystorage"
```

**When to use:**
- The auto-derived name doesn't match your existing package names
- You want more readable or conventional names
- You're migrating from another system and need to maintain compatibility

**How it works:**
1. Librarian discovers `google/api/apikeys/v2`
2. Derives default name → `google-api-apikeys`
3. Checks `name_overrides` for matching `api` entry
4. Finds override → uses `google-api-keys` instead
5. Looks up `google-api-keys` in `libraries` section for additional settings

---

### `libraries`

**Type:** array
**Required:** No

Library configurations. Each library entry specifies the library name and optional settings.

**With `generate: all` mode:**
```yaml
defaults:
  generate: all

libraries:
  # Add settings for auto-discovered libraries
  - name: google-cloud-vision
    keep:
      - google/cloud/vision_v1/helpers.py

  # Handwritten libraries (no api field)
  - name: pubsub
    path: pubsub/
```

**With `generate: explicit` mode:**
```yaml
defaults:
  generate: explicit

libraries:
  # Must specify api for each generated library
  - name: google-cloud-secretmanager
    api: google/cloud/secretmanager/v1

  - name: google-cloud-vision
    api: google/cloud/vision/v1

  - name: google-cloud-translate
    api: google/cloud/translate/v3
```

---

## Library Configuration

### Identifying Libraries

Each library entry has a `name` field (the package name) and optionally an `api` field (the googleapis API path).

**Generated libraries** (from googleapis):
```yaml
- name: google-cloud-vision
  api: google/cloud/vision/v1

- name: secretmanager
  api: google/cloud/secretmanager/v1

- name: google-cloud-vision-v1
  api: google/cloud/vision/v1
```

**Handwritten libraries** (no googleapis API):

Handwritten libraries have no `api` field and must specify a `path`.

```yaml
- name: pubsub
  path: pubsub

- name: auth
  path: auth

- name: compute-metadata
  path: compute/metadata
```

### Minimal Syntax

**With `generate: all`**, libraries that follow defaults need no configuration. Only add entries for libraries that need settings:

```yaml
defaults:
  generate: all

libraries:
  # Only list libraries that need settings
  - name: google-cloud-vision
    keep:
      - google/cloud/vision_v1/helpers.py
```

**With `generate: explicit`**, you must list every library:

```yaml
defaults:
  generate: explicit

libraries:
  - name: google-cloud-secretmanager
    api: google/cloud/secretmanager/v1

  - name: google-cloud-vision
    api: google/cloud/vision/v1
```

### With Additional Settings

Add configuration to libraries as needed:

```yaml
libraries:
  # Generated with handwritten additions
  - name: google-cloud-vision
    keep:
      - google/cloud/vision_v1/helpers.py
      - tests/unit/test_helpers.py

  # Generated with path override
  - name: google-cloud-firestore
    path: src/firestore/src/generated/gapic
    rust:
      template_override: templates/grpc-client

  # Handwritten
  - name: pubsub
    path: pubsub
    release:
      disabled: true
```

---

## Library Fields

Each library entry is an object with the following fields:

```yaml
- name: google-cloud-secretmanager  # Python: package name
- name: secretmanager               # Go: module name
- name: google-cloud-vision-v1      # Rust: crate name
- name: google_cloud_functions_v2   # Dart: package name
```

---

### `name`

**Type:** string
**Required:** Yes

The library name (package name published to registries like PyPI, crates.io, pub.dev).

```yaml
- name: google-cloud-secretmanager
```

---

### `api`

**Type:** string or object
**Required:** Yes (for generated libraries in explicit mode), No (for auto-discovered libraries)

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

**When required:**
- With `generate: explicit` - Required for all generated libraries
- With `generate: all` - Optional, only needed if you want to explicitly list a library for additional settings

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
- name: google-cloud-broken
  disabled: true
  reason: "Missing required BUILD.bazel service_config"
```

---

## Complete Examples

### Python with Generate All

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
  generate: all

release:
  tag_format: '{name}/v{version}'

name_overrides:
  - api: google/cloud/bigquery/storage/v1
    name: google-cloud-bigquery-storage

libraries:
  # Add settings for specific libraries
  - name: google-cloud-vision
    keep:
      - google/cloud/vision_v1/helpers.py

  - name: google-cloud-bigquery-storage
    keep:
      - google/cloud/bigquery_storage_v1/client.py

  # Handwritten libraries
  - name: pubsub
    path: pubsub/

  - name: auth
    path: auth/
```

---

### Go with Generate All

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
  generate: all

release:
  tag_format: '{name}/v{version}'

libraries:
  # Add settings for specific libraries
  - name: batch
    keep:
      - ^batch/apiv1/iam_policy_client\.go$

  # Handwritten libraries
  - name: pubsub
    path: pubsub/

  - name: storage
    path: storage/
```

---

### Rust with Generate All

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
  generate: all

release:
  tag_format: '{name}/v{version}'

libraries:
  # Add settings for specific libraries
  - name: google-cloud-aiplatform-v1beta1
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
  generate: explicit

release:
  tag_format: '{name}/v{version}'

libraries:
  # Common protos
  - name: google_api
    api: google/api

  - name: google_iam_v1
    api: google/iam/v1

  # Service APIs
  - name: google_cloud_secretmanager_v1
    api: google/cloud/secretmanager/v1

  - name: google_cloud_functions_v2
    api: google/cloud/functions/v2
```

---

## How Discovery Works

When `defaults.generate: all`:

1. **Scan googleapis** - Find all directories matching version patterns (`v1`, `v1beta`, etc.)
2. **Derive library names** - Use language-specific conventions
3. **Check name overrides** - Apply any `name_overrides` entries
4. **Compute output paths** - Apply `output` template
5. **Match library configs** - Apply any matching `libraries` entries
6. **Generate** - Create libraries at computed paths

**Example for Python:**
```
Discovered API: google/cloud/secretmanager/v1
Derive name:    google-cloud-secretmanager
Check overrides: None found
Compute path:   packages/google-cloud-secretmanager
Check config:   No matching library entry
Generate:       Create packages/google-cloud-secretmanager/
```

**Example with name override:**
```
Discovered API: google/api/apikeys/v2
Derive name:    google-api-apikeys
Check overrides: Found! Use google-api-keys instead
Compute path:   packages/google-api-keys
Check config:   Match found! Apply keep settings
Generate:       Create packages/google-api-keys/ with keep rules
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

### 1. Use Generate All for Large Repos

```yaml
defaults:
  generate: all

# Only list libraries that need settings
libraries:
  - name: google-cloud-vision
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
- name: google-cloud-broken
  disabled: true
  reason: "BUILD.bazel missing required service_config"
```

### 5. Use Name Overrides Sparingly

Only use `name_overrides` when the auto-derived name doesn't match your existing packages:

```yaml
name_overrides:
  - api: google/api/apikeys/v2
    name: google-api-keys  # Better than auto-derived "google-api-apikeys"
```

---

## See Also

- [README.md](../README.md) - User guide and workflows
- [alternatives.md](alternatives.md) - Design alternatives
