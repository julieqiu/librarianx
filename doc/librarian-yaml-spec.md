# librarian.yaml Specification

This document describes the `librarian.yaml` configuration file format and
how Librarian uses it to generate client libraries.

## Table of Contents

- [Overview](#overview)
- [File Structure](#file-structure)
- [Configuration Sections](#configuration-sections)
  - [Top-Level Fields](#top-level-fields)
  - [Default Configuration](#default-configuration)
  - [Library Configuration](#library-configuration)
  - [Language-Specific Options](#language-specific-options)
- [Auto-Discovery Mode](#auto-discovery-mode)
- [Name Overrides](#name-overrides)
- [Service Config Overrides](#service-config-overrides)
- [Examples](#examples)

## Overview

The `librarian.yaml` file is the main configuration file that controls how
Librarian generates client libraries from protocol buffer definitions. It defines:

- Which language to generate (Python, Rust, Go, etc.)
- Which APIs to generate and how to package them
- Default settings for all libraries
- Per-library overrides and customizations
- Version numbers for releases

## File Structure

```yaml
version: "1.0"
language: python
repo: googleapis/google-cloud-python

default:
  output: "packages/{name}/"
  generate:
    all: true
    one_library_per: api
    transport: grpc+rest
    rest_numeric_enums: true
    release_level: stable

name_overrides:
  google/api/apikeys/v2: google-cloud-api-keys

libraries:
  - name: google-cloud-secret-manager
    apis:
      - google/cloud/secretmanager/v1
      - google/cloud/secretmanager/v1beta2
    python:
      opt_args:
        - warehouse-package-name=google-cloud-secret-manager

versions:
  google-cloud-secret-manager: 2.25.0
```

## Configuration Sections

### Top-Level Fields

#### `version` (required)
The version of the librarian configuration format.

```yaml
version: "1.0"
```

#### `language` (required)
The target programming language. Supported values:
- `python`
- `rust`
- `go`
- `dart`

```yaml
language: python
```

#### `repo` (optional)
The GitHub repository in the format `owner/repo`. Used for generating documentation links and metadata.

```yaml
repo: googleapis/google-cloud-python
```

### Default Configuration

The `default` section defines settings that apply to all libraries unless overridden.

#### `default.output`
Directory pattern for generated code. The `{name}` placeholder is replaced with the library name.

```yaml
default:
  output: "packages/{name}/"
```

#### `default.generate`
Controls code generation behavior.

**`all` (boolean)**
When `true`, enables auto-discovery mode.
Librarian will automatically generate libraries for all APIs found in googleapis, unless excluded.

```yaml
default:
  generate:
    all: true
```

**`one_library_per` (string)**
Packaging strategy:
- `"api"`: Bundle all versions of a product into one library (default for Python, Go)
- `"version"`: Create separate library per version (default for Rust, Dart)

```yaml
default:
  generate:
    one_library_per: api
```

**`transport` (string)**
Default transport protocol:
- `"grpc"`: gRPC only
- `"rest"`: REST only
- `"grpc+rest"`: Both gRPC and REST

```yaml
default:
  generate:
    transport: grpc+rest
```

**`rest_numeric_enums` (boolean)**
Whether to use numeric enums in REST transport (instead of string enums).

```yaml
default:
  generate:
    rest_numeric_enums: true
```

**`release_level` (string)**
Default release stability:
- `"stable"`: Production-ready
- `"preview"`: Beta/preview quality

```yaml
default:
  generate:
    release_level: stable
```

**`exclude_apis` (list)**
API path patterns to exclude from auto-discovery. Supports wildcards.

```yaml
default:
  generate:
    exclude_apis:
      - "google/ads/*"
      - "google/cloud/aiplatform/v1beta1"
```

### Library Configuration

The `libraries` section defines per-library overrides and configurations.
Only include libraries that need special handling beyond defaults.

#### Basic Library Definition

```yaml
libraries:
  - name: google-cloud-secret-manager
    apis:
      - google/cloud/secretmanager/v1
      - google/cloud/secretmanager/v1beta2
      - google/cloud/secrets/v1beta1
```

**`name` (required)**
The library name. For Python, this is the PyPI package name. For Rust, the crate name.

**`apis` (optional)**
List of API paths to include in this library. Used for multi-version libraries.

**`api` (optional)**
Single API path. Alternative to `apis` for single-version libraries.

#### Common Library Options

**`path` (optional)**
Override the output directory. If not specified, uses `default.output` pattern.

```yaml
libraries:
  - name: google-cloud-storage
    path: packages/google-cloud-storage/
```

**`transport` (optional)**
Override default transport for this library.

```yaml
libraries:
  - name: google-cloud-firestore
    transport: grpc
```

**`rest_numeric_enums` (optional)**
Override default rest_numeric_enums setting.

```yaml
libraries:
  - name: google-cloud-compute
    rest_numeric_enums: false
```

**`release_level` (optional)**
Override default release level.

```yaml
libraries:
  - name: google-cloud-aiplatform
    release_level: preview
```

**`keep` (optional)**
Files/directories to preserve during regeneration. Supports glob patterns.

```yaml
libraries:
  - name: google-cloud-bigquery-storage
    keep:
      - docs/.*/library.rst
      - google/cloud/bigquery_storage_v1/client.py
      - google/cloud/bigquery_storage_v1/reader.py
      - tests/system/
```

**`generate.disabled` (optional)**
Disable generation for this library.

```yaml
libraries:
  - name: google-cloud-legacy-api
    generate:
      disabled: true
```

### Language-Specific Options

Each library can have language-specific configuration under the language name.

#### Python Options

**`python.opt_args`**
Additional options passed to protoc-gen-python_gapic.

```yaml
libraries:
  - name: google-cloud-secret-manager
    python:
      opt_args:
        - python-gapic-name=secretmanager
        - python-gapic-namespace=google.cloud
        - warehouse-package-name=google-cloud-secret-manager
```

Common opt_args:
- `python-gapic-name=<name>`: Override Python module name (e.g., `secretmanager`)
- `python-gapic-namespace=<namespace>`: Override namespace (e.g., `google.cloud`)
- `warehouse-package-name=<name>`: Override PyPI package name
- `proto-plus-deps=<package>`: Add proto-plus dependencies

**`python.is_proto_only`**
Generate only proto files, not GAPIC client.

```yaml
libraries:
  - name: google-cloud-common-protos
    python:
      is_proto_only: true
```

**`python.api_description`**
Override API description in `.repo-metadata.json`.

```yaml
libraries:
  - name: google-cloud-secret-manager
    python:
      api_description: "Stores, manages, and secures access to application secrets."
```

#### Rust Options

**`rust.crate_name`**
Override Rust crate name.

```yaml
libraries:
  - name: google-cloud-storage
    rust:
      crate_name: gcp-storage
```

**`rust.documentation_overrides`**
Fix documentation strings in generated code.

```yaml
libraries:
  - name: google-cloud-storage
    rust:
      documentation_overrides:
        - id: .google.storage.v2.Bucket.name
          match: "regsitry"
          replace: "registry"
```

### Versions

The `versions` section defines version numbers for all libraries. This is the source of truth for releases.

```yaml
versions:
  google-cloud-secret-manager: 2.25.0
  google-cloud-storage: 2.20.0
  google-cloud-firestore: 2.19.0
```

## Auto-Discovery Mode

When `default.generate.all: true` is set, Librarian operates in auto-discovery mode:

### How Auto-Discovery Works

1. **Scans googleapis directory** for all API paths matching the pattern:
   ```
   google/**/v*
   google/**/v*beta*
   google/**/v*alpha*
   ```

2. **Groups APIs by service** (when `one_library_per: api`):
   ```
   google/cloud/secretmanager/v1
   google/cloud/secretmanager/v1beta2
   google/cloud/secrets/v1beta1
   ```
   All grouped into library: `google-cloud-secretmanager`

3. **Derives library names** from API paths:
   ```
   google/cloud/storage/v1 → google-cloud-storage
   google/api/apikeys/v2 → google-api-apikeys-v2
   ```

4. **Discovers service configurations** automatically:
   ```
   google/cloud/secretmanager/v1/*.yaml → secretmanager_v1.yaml
   ```

5. **Applies exclusions** from `default.generate.exclude_apis`

6. **Checks name_overrides** to use custom library names

7. **Merges with explicit library configs** from `libraries` section

### When to Use Auto-Discovery

**Use auto-discovery when:**
- You want to generate all or most APIs
- APIs follow standard naming conventions
- You want minimal configuration

**Don't use auto-discovery when:**
- You only need a few specific APIs
- APIs have complex custom requirements
- You need tight control over what's generated

### Overriding Auto-Discovered Settings

Even with `all: true`, you can override individual libraries:

```yaml
default:
  generate:
    all: true
    transport: grpc+rest

libraries:
  # Override transport for this specific library
  - name: google-cloud-firestore
    transport: grpc

  # Disable auto-discovered library
  - name: google-cloud-legacy-api
    generate:
      disabled: true
```

## Name Overrides

The `name_overrides` section customizes library names when auto-discovered
names don't match existing packages or conventions.

### Syntax

```yaml
name_overrides:
  <api-path>: <library-name>
```

### Examples

```yaml
name_overrides:
  # API path has different structure than package name
  google/cloud/secretmanager: google-cloud-secret-manager

  # Simplify multi-segment paths
  google/devtools/artifactregistry: google-cloud-artifact-registry

  # Version-specific override
  google/api/apikeys/v2: google-cloud-api-keys
```

### How Name Overrides Work

1. During auto-discovery, Librarian derives a name from the API path
2. Before creating the library, it checks `name_overrides`
3. If a match is found, it uses the override name instead
4. The override name is used for:
   - Output directory: `packages/{override-name}/`
   - Package name (Python: PyPI name, Rust: crate name)
   - Matching with explicit `libraries` entries

### Name Override vs Library Name

```yaml
# These are equivalent:
name_overrides:
  google/cloud/secretmanager: google-cloud-secret-manager

# ...and...
libraries:
  - name: google-cloud-secret-manager
    api: google/cloud/secretmanager
```

Use `name_overrides` for simple renaming. Use `libraries` when you need additional configuration.

## Service Config Overrides

Service config overrides are defined in a separate file: `internal/config/service_config_overrides.yaml`

This file handles special cases where service YAML files don't follow the standard naming pattern `*_<version>.yaml`.

### Standard Auto-Discovery

By default, Librarian finds service configs using this algorithm:

```
For API: google/cloud/secretmanager/v1
Pattern: google/cloud/secretmanager/v1/*_v1.yaml
Excludes: *_gapic.yaml
Finds: secretmanager_v1.yaml
```

### When You Need Overrides

Override when:
- Service YAML has no version suffix (e.g., `monitoring.yaml` not `monitoring_v3.yaml`)
- Service YAML is shared across multiple API paths
- API path has no version (e.g., `google/api`, `google/type`)

### Example service_config_overrides.yaml

```yaml
service_configs:
  # Paths without version numbers
  google/cloud/location: cloud.yaml
  google/api: serviceconfig.yaml

  # Service YAML without version suffix
  google/monitoring/v3: monitoring.yaml
  google/iam/admin/v1: iam.yaml

  # Shared service YAML
  google/monitoring/dashboard/v1: monitoring.yaml
  google/monitoring/metricsscope/v1: monitoring.yaml

excluded_apis:
  all: []
  rust:
    - google/ads/*
    - google/cloud/compute/v1
  python:
    - google/internal/*
```

## Examples

### Minimal Configuration (Auto-Discovery)

Generate all APIs with defaults:

```yaml
version: "1.0"
language: python
repo: googleapis/google-cloud-python

default:
  output: "packages/{name}/"
  generate:
    all: true
    transport: grpc+rest
    rest_numeric_enums: true

versions:
  google-cloud-storage: 2.20.0
  google-cloud-firestore: 2.19.0
```

### Explicit Library List

Only generate specific libraries:

```yaml
version: "1.0"
language: python
repo: googleapis/google-cloud-python

default:
  output: "packages/{name}/"
  generate:
    transport: grpc+rest

libraries:
  - name: google-cloud-storage
    api: google/cloud/storage/v1

  - name: google-cloud-firestore
    api: google/cloud/firestore/v1
    transport: grpc  # Override default

versions:
  google-cloud-storage: 2.20.0
  google-cloud-firestore: 2.19.0
```

### Multi-Version Library

Bundle multiple API versions:

```yaml
version: "1.0"
language: python

default:
  output: "packages/{name}/"

libraries:
  - name: google-cloud-secret-manager
    apis:
      - google/cloud/secretmanager/v1
      - google/cloud/secretmanager/v1beta2
      - google/cloud/secrets/v1beta1
    python:
      opt_args:
        - python-gapic-name=secretmanager
        - python-gapic-namespace=google.cloud
        - warehouse-package-name=google-cloud-secret-manager
      api_description: "Stores, manages, and secures access to application secrets."

versions:
  google-cloud-secret-manager: 2.25.0
```

### Auto-Discovery with Overrides

Generate all, but customize specific libraries:

```yaml
version: "1.0"
language: python
repo: googleapis/google-cloud-python

default:
  output: "packages/{name}/"
  generate:
    all: true
    transport: grpc+rest
    rest_numeric_enums: true
    exclude_apis:
      - "google/ads/*"
      - "google/cloud/*/v*beta*"

name_overrides:
  google/cloud/secretmanager: google-cloud-secret-manager
  google/devtools/artifactregistry: google-cloud-artifact-registry

libraries:
  # Custom configuration for secret manager
  - name: google-cloud-secret-manager
    python:
      opt_args:
        - python-gapic-name=secretmanager
      api_description: "Stores, manages, and secures access to application secrets."

  # Keep handwritten files
  - name: google-cloud-bigquery-storage
    keep:
      - google/cloud/bigquery_storage_v1/client.py
      - google/cloud/bigquery_storage_v1/reader.py

versions:
  google-cloud-storage: 2.20.0
  google-cloud-secret-manager: 2.25.0
```

### Proto-Only Library

```yaml
version: "1.0"
language: python

libraries:
  - name: google-cloud-common-protos
    apis:
      - google/api
      - google/type
      - google/rpc
    transport: grpc
    python:
      is_proto_only: true

versions:
  google-cloud-common-protos: 1.65.0
```

### Rust Multi-Crate Repository

```yaml
version: "1.0"
language: rust
repo: googleapis/google-cloud-rust

default:
  output: "src/generated/{name}/"
  generate:
    all: true
    one_library_per: channel
    transport: grpc

name_overrides:
  google/cloud/storage/v1: google-cloud-storage-v1
  google/cloud/storage/v2: google-cloud-storage-v2

libraries:
  - name: google-cloud-storage-v1
    rust:
      crate_name: gcp-storage-v1

  - name: google-cloud-storage-v2
    rust:
      crate_name: gcp-storage-v2

versions:
  google-cloud-storage-v1: 0.1.0
  google-cloud-storage-v2: 0.2.0
```

## Best Practices

### 1. Use Auto-Discovery for New Repositories

Start with auto-discovery and add overrides as needed:

```yaml
default:
  generate:
    all: true
```

### 2. Keep Libraries Section Minimal

Only add libraries that need custom configuration. Auto-discovery handles the rest.

### 3. Use Name Overrides for Simple Renames

Use `name_overrides` for simple name changes. Use `libraries` when you need additional config.

### 4. Document Custom Settings

Add comments to explain why custom settings are needed:

```yaml
libraries:
  # Uses grpc-only because REST transport has compatibility issues
  - name: google-cloud-firestore
    transport: grpc
```

### 5. Group Related Versions

For multi-version libraries, list APIs in order from stable to beta:

```yaml
libraries:
  - name: google-cloud-aiplatform
    apis:
      - google/cloud/aiplatform/v1      # Stable
      - google/cloud/aiplatform/v1beta1  # Beta
```

### 6. Maintain Versions Separately

Keep version numbers in the `versions` section, not in library configs. This makes version bumps easier.

## Troubleshooting

### Library Not Being Generated

1. Check if it's excluded: `default.generate.exclude_apis`
2. Verify the API path exists in googleapis
3. Check if `generate.disabled: true` is set
4. Ensure service config can be found (check logs)

### Wrong Library Name

1. Check `name_overrides` for matching entry
2. Verify `one_library_per` setting (service vs version)
3. Look for explicit `name` in `libraries` section

### Missing Files in Output

1. Check `keep` list - files may be backed up and not restored
2. Verify protoc ran successfully (check logs)
3. Ensure formatters (black, isort) didn't fail

### Configuration Not Applied

1. Ensure library `name` matches exactly (case-sensitive)
2. Check YAML syntax is valid
3. Verify library-specific config overrides defaults
4. Check load order: defaults → name_overrides → libraries
