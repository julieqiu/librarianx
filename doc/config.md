# Configuration Reference

This document describes the `librarian.yaml` configuration schema with auto-discovery.

## Overview

Librarian uses a **minimal configuration** approach. Instead of explicitly listing every library, it automatically discovers APIs from the googleapis repository and derives library names using language-specific conventions.

Configuration defines:
- Repository-wide settings (language, sources)
- Default generation settings
- Release configuration
- **Exceptions only** - libraries that deviate from standard patterns

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
  generate_dir: packages/
  transport: grpc+rest
  rest_numeric_enums: true

release:
  tag_format: '{name}/v{version}'

# Auto-discover all APIs from googleapis
auto_discover: true

# Only list exceptions
libraries:
  # Exception: has handwritten code
  - google/cloud/bigquery/storage/v1:
      keep:
        - google/cloud/bigquery_storage_v1/client.py
```

**Result**: Automatically generates ~200+ libraries by scanning googleapis, only 1 exception needs explicit config.

### Minimal Dart Configuration (Explicit Mode)

```yaml
version: v1
language: dart

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  generate_dir: generated/
  packaging: version

release:
  tag_format: '{name}/v{version}'

# No auto-discovery - explicitly list what to generate
auto_discover: false

libraries:
  # Common protos
  - google/api
  - google/iam/v1
  - google/cloud/location

  # Service APIs
  - google/cloud/secretmanager/v1
  - google/cloud/functions/v2
```

**Result**: Only generates the 5 explicitly listed APIs.

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

#### `sources.protobuf` (Dart only)

**Fields:**
- `url` (string, required) - URL to protobuf tarball
- `sha256` (string, required) - SHA256 hash
- `extracted_name` (string, optional) - Directory name after extraction
- `subdir` (string, optional) - Subdirectory containing proto files

```yaml
sources:
  protobuf:
    url: https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protobuf-29.3.tar.gz
    sha256: 008a11cc56f9b96679b4c285fd05f46d317d685be3ab524b2a310be0fbad987e
    extracted_name: protobuf-29.3
    subdir: src
```

---

### `defaults`

**Type:** object
**Required:** No

Default settings applied to all libraries (can be overridden per-library).

**Common fields:**
- `generate_dir` (string) - Output directory for generated code
- `packaging` (string) - Packaging strategy: `service` or `version`
- `transport` (string) - Default transport: `grpc`, `rest`, `grpc+rest`
- `rest_numeric_enums` (boolean) - Use numeric enums in REST

```yaml
defaults:
  generate_dir: packages/
  packaging: service        # Bundle all versions into one library (Python/Go)
  transport: grpc+rest
  rest_numeric_enums: true
```

**Language-specific defaults:**

```yaml
# Rust
defaults:
  generate_dir: src/generated/{api.path}
  packaging: version        # Separate crate per version
  release_level: stable
  rust:
    disabled_rustdoc_warnings:
      - redundant_explicit_links

# Dart
defaults:
  generate_dir: generated/
  packaging: version        # Separate package per version
  dart:
    copyright_year: 2025
    issue_tracker_url: https://github.com/googleapis/google-cloud-dart/issues
```

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

### `auto_discover`

**Type:** boolean
**Required:** No
**Default:** `false`

Enable automatic API discovery by scanning the googleapis filesystem.

```yaml
auto_discover: true   # Scan googleapis and generate all APIs
auto_discover: false  # Only generate explicitly listed libraries
```

**How it works:**

When enabled, librarian:
1. Scans `googleapis/google/` directory tree
2. Finds all directories matching version patterns (`v1`, `v1beta`, etc.)
3. Checks for `BUILD.bazel` or `service.proto` to confirm it's an API
4. Derives API ID from path: `google/cloud/secretmanager/v1` → `google.cloud.secretmanager.v1`
5. Groups by service (Python/Go) or keeps separate (Rust/Dart) based on `packaging` setting

**Example discovery:**

```
googleapis/google/cloud/secretmanager/
├── v1/              → Found: google.cloud.secretmanager.v1
└── v1beta2/         → Found: google.cloud.secretmanager.v1beta2

Python/Go (packaging: service):
  → ONE library: google-cloud-secretmanager (contains both versions)

Rust/Dart (packaging: version):
  → TWO libraries: google-cloud-secretmanager-v1, google-cloud-secretmanager-v1beta2
```

---

### `libraries`

**Type:** array
**Required:** No (when `auto_discover: true`)

Library configurations.

**With auto-discovery:** Only list exceptions (handwritten code, name overrides, disabled APIs)
**Without auto-discovery:** Must list all libraries explicitly

---

## Library Configuration

### Short Syntax (API Path Only)

For libraries that follow all defaults:

```yaml
libraries:
  - google/cloud/secretmanager/v1
  - google/api
  - google/iam/v1
```

Librarian automatically:
- Derives library name from API path
- Uses default settings from `defaults` section
- Discovers service config files

---

### Extended Syntax (With Overrides)

For libraries that need customization:

```yaml
libraries:
  # Override name
  - google/cloud/bigquery/storage/v1:
      name: google-cloud-bigquerystorage

  # Add handwritten code preservation
  - google/cloud/bigquery/storage/v1:
      keep:
        - google/cloud/bigquery_storage_v1/client.py
        - google/cloud/bigquery_storage_v1/reader.py

  # Disable API
  - google/cloud/broken/v1:
      disabled: true
      reason: "Missing BUILD.bazel configuration"

  # Language-specific config
  - google/cloud/aiplatform/v1beta1:
      dart:
        api_keys_environment_variables: GOOGLE_API_KEY
        dev_dependencies:
          - googleapis_auth
```

---

### Library Fields

#### `name`

**Type:** string
**Required:** No (derived from API path)

Override the default library name.

**Default derivation:**

| Language | API Path | Derived Name |
|----------|----------|--------------|
| Python | `google/cloud/secretmanager/v1` | `google-cloud-secretmanager` |
| Go | `google/cloud/secretmanager/v1` | `secretmanager` |
| Rust | `google/cloud/secretmanager/v1` | `google-cloud-secretmanager-v1` |
| Dart | `google/cloud/secretmanager/v1` | `google_cloud_secretmanager_v1` |

**Override:**

```yaml
- google/cloud/bigquery/storage/v1:
    name: google-cloud-bigquerystorage
```

#### `keep`

**Type:** array (strings)
**Required:** No

Files and directories to preserve during regeneration (for hybrid libraries with handwritten code).

**Format:** Array of file paths relative to repository root.

```yaml
keep:
  - google/cloud/bigquery_storage_v1/client.py          # Specific file
  - google/cloud/bigquery_storage_v1/samples/           # Directory
  - google/cloud/bigquery_storage_v1/custom/*.go        # Pattern
```

#### `disabled`

**Type:** boolean
**Required:** No

Disable generation for this API. Must include a `reason` field.

```yaml
- google/cloud/broken/v1:
    disabled: true
    reason: "Missing required BUILD.bazel service_config"
```

#### `versions`

**Type:** array (strings)
**Required:** No

For multi-version APIs, specify which versions to include. By default, all discovered versions are included.

```yaml
# API has v1, v1alpha, v1beta, v1beta2, v1beta3
# Only generate v1 and v1beta3
- google/ai/generativelanguage:
    versions: [v1, v1beta3]
```

#### Language-Specific Fields

See language-specific documentation:
- [go.md](go.md) - Go-specific fields
- [python.md](python.md) - Python-specific fields
- [rust.md](rust.md) - Rust-specific fields

---

## Name Derivation Rules

Library names are derived from API paths using language conventions:

### Python (service-level packaging)

```
API path: google/cloud/secretmanager/v1
Service:  secretmanager
Namespace: cloud
Name:     google-cloud-secretmanager

API path: google/ai/generativelanguage/v1
Service:  generativelanguage
Namespace: ai
Name:     google-ai-generativelanguage
```

### Go (service-level packaging)

```
API path: google/cloud/secretmanager/v1
Service:  secretmanager
Name:     secretmanager

API path: google/analytics/admin/v1alpha
Service:  admin (from google/analytics/admin)
Name:     analytics
```

### Rust (version-level packaging)

```
API path: google/cloud/secretmanager/v1
Name:     google-cloud-secretmanager-v1

API path: google/cloud/secretmanager/v1beta2
Name:     google-cloud-secretmanager-v1beta2
```

### Dart (version-level packaging)

```
API path: google/cloud/secretmanager/v1
Name:     google_cloud_secretmanager_v1

API path: google/ai/generativelanguage/v1beta
Name:     google_cloud_ai_generativelanguage_v1beta
```

---

## Multi-Version Handling

### Service-Level Packaging (Python, Go)

All versions of a service are bundled into one library:

```yaml
# Discovered APIs:
# - google/ai/generativelanguage/v1
# - google/ai/generativelanguage/v1alpha
# - google/ai/generativelanguage/v1beta
# - google/ai/generativelanguage/v1beta2
# - google/ai/generativelanguage/v1beta3

# Result: ONE library with all 5 versions
# Name: google-ai-generativelanguage (Python)
# Name: ai (Go)
```

**To generate only specific versions:**

```yaml
- google/ai/generativelanguage:
    versions: [v1, v1beta3]  # Skip alpha, beta, beta2
```

### Version-Level Packaging (Rust, Dart)

Each version becomes a separate library:

```yaml
# Discovered APIs:
# - google/cloud/aiplatform/v1
# - google/cloud/aiplatform/v1beta1

# Result: TWO libraries
# Rust: google-cloud-aiplatform-v1, google-cloud-aiplatform-v1beta1
# Dart: google_cloud_aiplatform_v1, google_cloud_aiplatform_v1beta1
```

**To generate only v1:**

```yaml
- google/cloud/aiplatform/v1  # Only list the one you want
```

---

## Complete Examples

### Python with Auto-Discovery

```yaml
version: v1
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  generate_dir: packages/
  packaging: service
  transport: grpc+rest
  rest_numeric_enums: true

release:
  tag_format: '{name}/v{version}'

auto_discover: true

libraries:
  # Exception: handwritten code
  - google/cloud/bigquery/storage/v1:
      keep:
        - google/cloud/bigquery_storage_v1/client.py
        - google/cloud/bigquery_storage_v1/reader.py
        - google/cloud/bigquery_storage_v1/writer.py

  - google/cloud/automl/v1beta1:
      keep:
        - docs/automl_v1beta1/tables.rst
        - google/cloud/automl_v1beta1/services/tables

  # Exception: custom name
  - google/cloud/translate:
      keep:
        - google/cloud/translate_v2
        - tests/unit/v2
```

**Result**: ~200+ libraries generated, only 3 need explicit config.

---

### Go with Auto-Discovery

```yaml
version: v1
language: go

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  generate_dir: '{name}/'
  packaging: service
  transport: grpc+rest

release:
  tag_format: '{name}/v{version}'

auto_discover: true

libraries:
  # Exception: handwritten IAM client
  - google/cloud/batch/v1:
      keep:
        - ^batch/apiv1/iam_policy_client\.go$

  - google/cloud/vmmigration/v1:
      keep:
        - ^vmmigration/apiv1/iam_policy_client\.go$

  # Exception: handwritten library (no generation)
  - pubsub:
      auto_discover: false  # Don't generate, fully handwritten

  - storage:
      auto_discover: false
```

---

### Rust with Auto-Discovery

```yaml
version: v1
language: rust

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH
  discovery:
    url: https://github.com/googleapis/discovery-artifact-manager/archive/COMMIT.tar.gz
    sha256: HASH

defaults:
  generate_dir: src/generated/{api.path}
  packaging: version
  release_level: stable
  rust:
    disabled_rustdoc_warnings:
      - redundant_explicit_links
      - broken_intra_doc_links

release:
  tag_format: '{name}/v{version}'

auto_discover: true

# Filter: only stable versions
discover_filter:
  exclude: [*.v1alpha, *.v1beta*]

libraries:
  # Exception: include beta version for this service
  - google/cloud/aiplatform/v1beta1:
      rust:
        per_service_features: true

  # Exception: special Compute config (Discovery-based)
  - google/cloud/compute/v1:
      specification_format: disco
      rust:
        per_service_features: true
        default_features:
          - instances
          - projects
```

---

### Dart without Auto-Discovery

```yaml
version: v1
language: dart

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/COMMIT.tar.gz
    sha256: HASH
  protobuf:
    url: https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protobuf-29.3.tar.gz
    sha256: HASH
    extracted_name: protobuf-29.3
    subdir: src

defaults:
  generate_dir: generated/
  packaging: version
  dart:
    copyright_year: 2025

release:
  tag_format: '{name}/v{version}'

auto_discover: false

libraries:
  # Common protos
  - google/api
  - google/iam/v1
  - google/cloud/location

  # Service APIs
  - google/cloud/secretmanager/v1
  - google/cloud/functions/v2

  # With overrides
  - google/cloud/aiplatform/v1beta1:
      dart:
        api_keys_environment_variables: GOOGLE_API_KEY
```

---

## Configuration Best Practices

### 1. Use Auto-Discovery

Enable auto-discovery to minimize configuration:

```yaml
# Good: auto-discover everything
auto_discover: true

# Only list exceptions
libraries:
  - google/cloud/bigquery/storage/v1:
      keep: [...]
```

### 2. Use Immutable Source References

Always use commit-specific URLs with SHA256 hashes:

```yaml
sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123def456.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
```

### 3. Use Defaults

Put common settings in `defaults` to avoid repetition:

```yaml
defaults:
  generate_dir: packages/
  transport: grpc+rest
  rest_numeric_enums: true
```

### 4. Minimal Keep Lists

Only add `keep` entries when you've actually added handwritten code:

```yaml
# Good: minimal keep list
keep:
  - google/cloud/bigquery_storage_v1/client.py

# Bad: preemptive keep list
keep:
  - google/cloud/bigquery_storage_v1/client.py
  - google/cloud/bigquery_storage_v1/helpers.py  # Doesn't exist yet
```

### 5. Document Disabled APIs

Always include a reason when disabling:

```yaml
- google/cloud/broken/v1:
    disabled: true
    reason: "BUILD.bazel missing required service_config"
```

---

## Migration from Explicit Config

To migrate from an explicit config (listing all libraries) to auto-discovery:

1. **Add auto_discover: true**
2. **Remove all standard libraries** that follow naming conventions
3. **Keep only exceptions** (handwritten code, custom names, disabled)
4. **Test**: `librarian generate --all --dry-run`

**Example migration:**

**Before (1,612 lines):**
```yaml
libraries:
  - name: google-cloud-secret-manager
    version: 2.20.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
  - name: google-cloud-storage
    version: 2.10.0
    generate:
      apis:
        - path: google/storage/v2
  # ... 229 more libraries
```

**After (99 lines):**
```yaml
auto_discover: true

libraries:
  # Only exceptions remain
  - google/cloud/storage:
      keep:
        - google/cloud/storage/client.py
```

---

## See Also

- [alternatives.md](alternatives.md) - Design alternatives and why they were rejected
- [go.md](go.md) - Go-specific configuration
- [python.md](python.md) - Python-specific configuration
- [rust.md](rust.md) - Rust-specific configuration
