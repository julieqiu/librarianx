# Design: Extended librarian.yaml Schema

## Objective

Extend the `librarian.yaml` schema to support all fields currently in sidekick's `.sidekick.toml` configuration, enabling a complete migration from sidekick to librarian for Rust (and future languages).

## Background

The current `librarian.yaml` schema supports basic library configuration but lacks support for:
- Multiple specification formats (Discovery, OpenAPI)
- Service configuration files
- Source filtering (included/skipped IDs)
- Language-specific generation options (Rust package names, features, templates)
- Internal package dependencies
- Discovery-specific LRO configuration
- Per-library publication settings

These fields are critical for the Rust SDK and prevent a complete migration from sidekick to librarian.

## Design Principles

1. **Language-agnostic core**: Keep top-level structure language-neutral
2. **Language-specific extensions**: Allow language-specific config where needed
3. **Backward compatible**: Existing configs continue to work
4. **Progressive disclosure**: Simple cases remain simple, complex cases are possible
5. **Clear semantics**: Nested structure shows relationships between fields

## Proposed Schema

### Root-Level Changes

#### 1. Extend `sources` to support multiple source types

**Current:**
```yaml
sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/ABC.tar.gz
    sha256: 81e6057...
```

**Proposed:**
```yaml
sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/ABC.tar.gz
    sha256: 81e6057...
    extracted_name: googleapis-ABC  # optional
  showcase:
    url: https://github.com/googleapis/gapic-showcase/archive/v0.36.2.tar.gz
    sha256: 0914bdb...
    extracted_name: gapic-showcase-0.36.2
  discovery:
    url: https://github.com/googleapis/discovery-artifact-manager/archive/ABC.tar.gz
    sha256: 867048e...
    extracted_name: discovery-artifact-manager-ABC
  protobuf_src:
    url: https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protobuf-29.3.tar.gz
    sha256: 008a11c...
    extracted_name: protobuf-29.3
    subdir: src  # optional subdirectory to use
```

**Rationale**: Rust generator needs multiple source repositories. Each source has URL, SHA256, and optional extracted name/subdir.

#### 2. Extend `defaults` with language-specific options

**Current:**
```yaml
defaults:
  release_level: stable
```

**Proposed:**
```yaml
defaults:
  release_level: stable
  transport: grpc+rest  # or "grpc", "rest"
  rest_numeric_enums: false

  # Language-specific defaults (Rust)
  rust:
    disabled_rustdoc_warnings:
      - redundant_explicit_links
      - broken_intra_doc_links
    disabled_clippy_warnings:
      - doc_lazy_continuation

    # Default package dependencies (Rust monorepo)
    package_dependencies:
      - name: bytes
        package: bytes
        force_used: true
      - name: serde
        package: serde
        force_used: true
      - name: gax
        package: google-cloud-gax
        used_if: services
      - name: wkt
        package: google-cloud-wkt
        source: google.protobuf
        force_used: true

    # Rust-specific release configuration
    release:
      tools:
        cargo:
          - name: cargo-semver-checks
            version: 0.44.0
          - name: cargo-workspaces
            version: 0.4.0
      pre_installed:
        cargo: cargo
        git: git
```

**Rationale**: Avoids repeating common options across 200+ libraries. Language-specific options go under `defaults.rust` namespace. Rust-specific release tools also go here since they're defaults.

#### 3. Extend `release` configuration

**Current:**
```yaml
release:
  tag_format: '{name}/v{version}'
```

**Proposed:**
```yaml
release:
  tag_format: '{name}/v{version}'
  remote: upstream
  branch: main
  ignored_changes:
    - .repo-metadata.json
    - .sidekick.toml
    - .librarian.yaml
```

**Rationale**: Release process needs git configuration. Language-specific release tools moved to `defaults.rust.release`.

### Library-Level Changes

#### 1. Make `Library.Apis` more expressive

**Current:**
```yaml
libraries:
  - name: secretmanager
    apis:
      - google/cloud/secretmanager/v1
```

**Proposed (supports both formats):**
```yaml
libraries:
  - name: secretmanager
    # Simple format (backward compatible)
    apis:
      - google/cloud/secretmanager/v1

  - name: secretmanager-full
    # Extended format
    apis:
      - path: google/cloud/secretmanager/v1
        specification_format: protobuf  # or disco, openapi, none
```

**Rationale**: Specification format varies per library (most are protobuf, but Compute is disco). Service config customizations are handled by `service_config_overrides.yaml` files.

#### 2. Add library metadata fields

**Proposed:**
```yaml
libraries:
  - name: google-cloud-secretmanager-v1
    version: 1.1.1
    copyright_year: 2024

    # Display metadata
    title: "Secret Manager API"
    description: "Stores sensitive data such as API keys, passwords, and certificates"
```

**Rationale**: Name is the primary identifier (the package/crate name). Display metadata (title, description) is separate.

#### 3. Extend `Library.Generate` configuration

**Current:**
```yaml
libraries:
  - name: bigquery
    generate:
      apis:
        - path: google/cloud/bigquery/storage/v1
      keep:
        - client.go
```

**Proposed:**
```yaml
libraries:
  - name: google-cloud-storage
    generate:
      apis:
        - path: google/storage/v2

      # Code generation template
      template: templates/grpc-client  # or templates/gapic, templates/veneer

      # Source filtering
      source:
        roots:
          - googleapis
          - discovery
        included_ids:
          - .google.storage.v2.Storage.DeleteBucket
          - .google.storage.v2.Storage.GetBucket
          - .google.storage.v2.Storage.CreateBucket
        skipped_ids:
          - .google.storage.v2.Storage.InternalMethod
        include_list:
          - google/storage/v2/storage.proto
        title_override: "Cloud Storage JSON API"
        description_override: "Custom description"

      # Discovery-specific configuration (for disco format APIs)
      discovery:
        operation_id: .google.cloud.compute.v1.Operation
        pollers:
          - prefix: compute/v1/projects/{project}/zones/{zone}
            method_id: .google.cloud.compute.v1.zoneOperations.get
          - prefix: compute/v1/projects/{project}/regions/{region}
            method_id: .google.cloud.compute.v1.regionOperations.get

      # Language-specific code generation options (Rust)
      rust:
        # Naming overrides
        name_overrides:
          .google.storage.v2.Storage: StorageControl
          .google.storage.v2.Bucket: BucketResource

        # Module configuration
        module_path: src/custom/path
        root_name: custom_root

        # Feature flags (Cargo features)
        per_service_features: true
        default_features:
          - instances
          - projects

        # Extra modules to generate
        extra_modules:
          - errors
          - operation

        # Internal package dependencies (monorepo)
        package_dependencies:
          - name: lro
            package: google-cloud-lro
            force_used: true
          - name: location
            package: google-cloud-location
            source: google.cloud.location
          - name: iam_v1
            package: google-cloud-iam-v1
            source: google.iam.v1

        # Generation behavior flags
        has_veneer: true
        routing_required: true
        include_grpc_only_methods: true
        generate_setter_samples: true
        post_process_protos: true
        detailed_tracing_attributes: true

        # Linting configuration
        disabled_rustdoc_warnings:
          - bare_urls
          - broken_intra_doc_links
        disabled_clippy_warnings:
          - doc_lazy_continuation

      # Files to keep during regeneration (hybrid libraries)
      keep:
        - src/client.rs
        - tests/integration/
        - CHANGELOG.md

      # Files to remove after generation
      remove:
        - deprecated_file.rs
```

**Rationale**: All generation-related configuration lives under `generate`. Language-specific options are under `generate.rust` namespace to clearly separate language-agnostic from language-specific configuration.

#### 4. Add publication configuration

**Proposed:**
```yaml
libraries:
  - name: internal-test-crate
    publish:
      enabled: false  # marks crate as not-for-publication
      registry: crates.io  # or custom registry

  - name: secretmanager
    publish:
      enabled: true
      registry: crates.io
```

**Rationale**: Some internal crates should not be published to public registries.

## Complete Example

```yaml
version: v1
language: rust

container:
  image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/rust-librarian-generator
  tag: latest

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/9fcfbea.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
  showcase:
    url: https://github.com/googleapis/gapic-showcase/archive/v0.36.2.tar.gz
    sha256: 0914bdbb088713aa087a53b355ff6631ad95e4769afd8bbd97d5d9e78d4cdf09
    extracted_name: gapic-showcase-0.36.2

defaults:
  release_level: stable
  transport: grpc+rest
  rust:
    disabled_rustdoc_warnings:
      - redundant_explicit_links
      - broken_intra_doc_links
    package_dependencies:
      - name: bytes
        package: bytes
        force_used: true
      - name: gax
        package: google-cloud-gax
        used_if: services
    release:
      tools:
        cargo:
          - name: cargo-semver-checks
            version: 0.44.0

generate:
  output: src/generated/{api.path}

release:
  tag_format: '{name}/v{version}'
  remote: upstream
  branch: main
  ignored_changes:
    - .repo-metadata.json

libraries:
  # Simple fully-generated library
  - name: google-cloud-secretmanager-v1
    version: 1.1.1
    copyright_year: 2024
    apis:
      - path: google/cloud/secretmanager/v1
    generate:
      rust:
        generate_setter_samples: true

  # Complex hybrid library with custom configuration
  - name: google-cloud-storage
    version: 2.5.0
    copyright_year: 2024
    title: "Cloud Storage"
    apis:
      - path: google/storage/v2
    path: src/storage
    generate:
      template: templates/grpc-client
      source:
        included_ids:
          - .google.storage.v2.Storage.DeleteBucket
          - .google.storage.v2.Storage.GetBucket
      rust:
        name_overrides:
          .google.storage.v2.Storage: StorageControl
        has_veneer: true
        routing_required: true
        package_dependencies:
          - name: lro
            package: google-cloud-lro
            force_used: true
      keep:
        - src/client.rs
        - src/download.rs
        - tests/

  # Discovery-based API (Compute)
  - name: google-cloud-compute-v1
    version: 0.2.1
    copyright_year: 2025
    apis:
      - path: discoveries/compute.v1.json
        specification_format: disco
    generate:
      source:
        roots:
          - discovery
          - googleapis
      discovery:
        operation_id: .google.cloud.compute.v1.Operation
        pollers:
          - prefix: compute/v1/projects/{project}/zones/{zone}
            method_id: .google.cloud.compute.v1.zoneOperations.get
      rust:
        per_service_features: true
        default_features:
          - instances
          - projects
        extra_modules:
          - errors
          - operation
        disabled_rustdoc_warnings:
          - bare_urls
          - broken_intra_doc_links
        disabled_clippy_warnings:
          - doc_lazy_continuation
        package_dependencies:
          - name: lro
            package: google-cloud-lro
            force_used: true

  # Handwritten library (no generation)
  - name: google-cloud-pubsub
    version: 1.0.0
    copyright_year: 2024
    path: src/pubsub
```

## Field Organization Strategy

**Language-agnostic configuration** lives at the top level:
```
root
├── version, language
├── container
├── sources
├── generate.output
└── release (tag_format, remote, branch, ignored_changes)
```

**Language-specific defaults** live under `defaults.<language>`:
```
defaults.rust
├── disabled_rustdoc_warnings
├── disabled_clippy_warnings
├── package_dependencies
└── release
    ├── tools (cargo subcommands)
    └── pre_installed (system binaries)
```

**Per-library generation config** lives under `library.generate`:
```
library.generate
├── template              (which template to use)
├── source                (what to include/exclude)
│   ├── roots
│   ├── included_ids
│   ├── skipped_ids
│   └── ...
├── discovery             (Discovery API specific)
│   ├── operation_id
│   └── pollers
├── rust                  (Rust-specific options)
│   ├── name_overrides
│   ├── package_dependencies
│   ├── per_service_features
│   ├── disabled_rustdoc_warnings
│   └── ...
├── keep                  (files to preserve)
└── remove                (files to delete)
```

This structure makes it clear:
- **Top level** = language-agnostic (works for any language)
- **`defaults.rust`** = Rust defaults (applies to all Rust libraries)
- **`library.generate.rust`** = Per-library Rust overrides

## Field Reference

### Root Level

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | Schema version (v1) |
| `language` | string | Yes | Primary language (rust, go, python) |
| `container` | object | No | Container image configuration |
| `sources` | object | No | External source repositories |
| `defaults` | object | No | Default settings for all libraries |
| `generate` | object | No | Generation configuration |
| `release` | object | No | Release configuration |
| `libraries` | array | Yes | List of libraries |

### `sources.*`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Download URL |
| `sha256` | string | Yes | SHA256 hash |
| `extracted_name` | string | No | Directory name after extraction |
| `subdir` | string | No | Subdirectory to use |

### `defaults`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `release_level` | string | No | stable or preview |
| `transport` | string | No | grpc, rest, or grpc+rest |
| `rest_numeric_enums` | bool | No | Use numeric enums in REST |
| `rust` | object | No | Rust-specific defaults |

### `defaults.rust`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `disabled_rustdoc_warnings` | array | No | Rustdoc warnings to disable |
| `disabled_clippy_warnings` | array | No | Clippy warnings to disable |
| `package_dependencies` | array | No | Default package dependencies |
| `release` | object | No | Rust-specific release config |

### `defaults.rust.release`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tools` | object | No | Required tools (cargo subcommands) |
| `pre_installed` | object | No | Pre-installed binaries |

### `release`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tag_format` | string | No | Git tag format template |
| `remote` | string | No | Git remote name |
| `branch` | string | No | Git branch name |
| `ignored_changes` | array | No | Files to ignore in changelogs |

### `libraries[]`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Package/crate name (e.g., google-cloud-storage) |
| `version` | string | No | Current version |
| `copyright_year` | int | No | Copyright year |
| `title` | string | No | Display title |
| `description` | string | No | Library description |
| `apis` | array | Conditional | API paths (required if `generate` present) |
| `path` | string | No | Filesystem path |
| `generate` | object | No | Generation configuration |
| `publish` | object | No | Publication settings |

### `libraries[].apis[]`

Simple format (string): `google/cloud/secretmanager/v1`

Or extended format (object):

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | API path |
| `specification_format` | string | No | protobuf, disco, openapi, none |

### `libraries[].generate`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `template` | string | No | Template directory override |
| `source` | object | No | Source filtering options |
| `discovery` | object | No | Discovery-specific config |
| `rust` | object | No | Rust-specific generation options |
| `keep` | array | No | Files to preserve |
| `remove` | array | No | Files to delete |

### `libraries[].generate.source`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `roots` | array | No | Source roots to search |
| `included_ids` | array | No | IDs to include (whitelist) |
| `skipped_ids` | array | No | IDs to skip (blacklist) |
| `include_list` | array | No | Files to include |
| `title_override` | string | No | Override title |
| `description_override` | string | No | Override description |

### `libraries[].generate.discovery`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `operation_id` | string | No | LRO operation ID format |
| `pollers` | array | No | Polling configuration |

### `libraries[].generate.discovery.pollers[]`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `prefix` | string | Yes | URL prefix pattern |
| `method_id` | string | Yes | Polling method ID |

### `libraries[].generate.rust`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name_overrides` | object | No | Type/service name overrides |
| `module_path` | string | No | Module path override |
| `root_name` | string | No | Root name override |
| `per_service_features` | bool | No | Enable per-service Cargo features |
| `default_features` | array | No | Default Cargo features list |
| `extra_modules` | array | No | Extra modules to generate |
| `package_dependencies` | array | No | Crate dependencies |
| `has_veneer` | bool | No | Has handwritten veneer |
| `routing_required` | bool | No | Routing headers required |
| `include_grpc_only_methods` | bool | No | Include gRPC-only methods |
| `generate_setter_samples` | bool | No | Generate setter examples |
| `post_process_protos` | bool | No | Post-process protos |
| `detailed_tracing_attributes` | bool | No | Detailed tracing |
| `disabled_rustdoc_warnings` | array | No | Rustdoc warnings to disable |
| `disabled_clippy_warnings` | array | No | Clippy warnings to disable |

### `libraries[].generate.rust.package_dependencies[]`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Dependency identifier |
| `package` | string | Yes | Package name |
| `source` | string | No | Proto source package |
| `force_used` | bool | No | Always include |
| `used_if` | string | No | Condition (services, lro, etc) |
| `feature` | string | No | Cargo feature to enable |

### `libraries[].publish`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable publication (default true) |
| `registry` | string | No | Package registry |

## Migration Path

1. **Phase 1**: Add new fields to schema, keep backward compatibility
2. **Phase 2**: Update converter to populate all fields
3. **Phase 3**: Update generator container to read new fields
4. **Phase 4**: Deprecate `.sidekick.toml` format

## Alternatives Considered

### Alternative 1: Flat structure with prefixed keys

```yaml
libraries:
  - name: storage
    codec_name_overrides: {".google.storage.v2.Storage": "StorageControl"}
    codec_has_veneer: true
    codec_routing_required: true
```

**Rejected**: Less readable, doesn't show relationships between fields.

### Alternative 2: Opaque codec options map

```yaml
libraries:
  - name: storage
    generate:
      codec_options:
        name-overrides: ".google.storage.v2.Storage=StorageControl"
        has-veneer: "true"
```

**Rejected**: No validation, no type safety, harder to document.

### Alternative 3: Keep codec options in container only

Move all codec options to container environment variables or JSON config.

**Rejected**: Makes config opaque, harder to review in PRs, language-specific options belong in config file.

## Summary

This design extends `librarian.yaml` to support all sidekick fields while maintaining:
- Language-agnostic core structure
- Clear separation of concerns
- Backward compatibility
- Progressive disclosure (simple cases stay simple)
- Validation and type safety

All 226 libraries from google-cloud-rust can be fully represented in this extended schema.
