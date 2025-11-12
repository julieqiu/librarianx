# Unsupported Fields from sidekick.toml

This document lists the fields from `.sidekick.toml` that are **not currently supported** in `librarian.yaml`.

## Summary

Successfully converted **226 libraries** from google-cloud-rust to librarian.yaml format.

The following fields from sidekick.toml are **NOT** represented in the current librarian.yaml schema:

## General Section

### `general.service-config` (190 occurrences)
**Purpose**: Specifies the service configuration YAML file path (e.g., `google/cloud/kms/v1/cloudkms_v1.yaml`)

**Example**:
```toml
[general]
service-config = 'google/cloud/kms/v1/cloudkms_v1.yaml'
```

**Why needed**: Service config contains critical metadata like authentication scopes, service endpoints, and method-level configurations. Without this, the generator cannot properly configure service clients.

### `general.specification-format` (non-protobuf formats) (3 occurrences)
**Purpose**: Specifies alternative specification formats
- `disco` (Discovery format, used by Compute)
- `openapi` (OpenAPI format)
- `none` (No specification, handwritten code)

**Example**:
```toml
[general]
specification-format = 'disco'
specification-source = 'discoveries/compute.v1.json'
```

**Why needed**: Rust SDK supports multiple API specification formats, not just protobuf.

## Source Section

### `source.roots` (8 occurrences)
**Purpose**: Specifies which source roots to search (e.g., `'googleapis'`, `'discovery,googleapis'`)

**Example**:
```toml
[source]
roots = 'discovery,googleapis'
```

### `source.included-ids` (6 occurrences)
**Purpose**: Explicitly lists which service methods/messages to include in generation (whitelist)

**Example**:
```toml
[source]
included-ids = """\
    .google.storage.v2.Storage.DeleteBucket,\
    .google.storage.v2.Storage.GetBucket,\
    .google.storage.v2.Storage.CreateBucket\
    """
```

**Why needed**: For hybrid libraries with both generated and handwritten code, this allows precise control over what gets generated.

### `source.skipped-ids` (4 occurrences)
**Purpose**: Explicitly lists which service methods/messages to exclude from generation (blacklist)

### `source.include-list` (7 occurrences)
**Purpose**: Lists specific proto files to include in generation

### `source.title-override` (15 occurrences)
**Purpose**: Overrides the display title for the library

### `source.description-override` (1 occurrence)
**Purpose**: Overrides the description for the library

### `source.project-root` (1 occurrence)
**Purpose**: Specifies the project root directory

## Discovery Section (for Discovery/OpenAPI specs)

### `discovery.operation-id` (1 occurrence)
**Purpose**: Specifies the operation ID format for long-running operations

**Example**:
```toml
[discovery]
operation-id = '.google.cloud.compute.v1.Operation'
```

### `discovery.pollers` (1 occurrence)
**Purpose**: Configures polling methods for long-running operations

**Example**:
```toml
[[discovery.pollers]]
prefix    = 'compute/v1/projects/{project}/zones/{zone}'
method-id = '.google.cloud.compute.v1.zoneOperations.get'
```

**Why needed**: Discovery-based APIs (like Compute) need special configuration for LRO polling.

## Codec Section (Rust-specific generation options)

### `codec.package-name-override` (23 occurrences)
**Purpose**: Overrides the generated Rust package name

**Example**:
```toml
[codec]
package-name-override = 'google-cloud-compute-v1'
```

**Why needed**: Package names often don't follow a simple transformation from API paths.

### `codec.template-override` (25 occurrences)
**Purpose**: Specifies alternative code generation templates

**Example**:
```toml
[codec]
template-override = 'templates/grpc-client'
```

**Why needed**: Different library types (GAPIC vs veneer vs internal) need different templates.

### `codec.module-path` (14 occurrences)
**Purpose**: Specifies the Rust module path for the generated code

### `codec.package:*` (17 different package references)
**Purpose**: Specifies dependencies on other packages in the monorepo

**Example**:
```toml
[codec]
'package:lro' = 'force-used=true,package=google-cloud-lro'
'package:location' = 'package=google-cloud-location,source=google.cloud.location'
```

**Why needed**: Internal dependencies between generated crates need explicit configuration.

### `codec.per-service-features` (5 occurrences)
**Purpose**: Enables per-service Cargo features (for large APIs like Compute)

**Example**:
```toml
[codec]
per-service-features = 'true'
default-features = 'instances,projects'
```

**Why needed**: Large APIs need feature flags to reduce compile times.

### `codec.has-veneer` (3 occurrences)
**Purpose**: Indicates the library has a handwritten veneer layer

**Why needed**: Affects code generation strategy (minimal vs full generation).

### `codec.routing-required` (2 occurrences)
**Purpose**: Indicates routing headers are required for this API

### `codec.name-overrides` (5 occurrences)
**Purpose**: Overrides generated type/service names

**Example**:
```toml
[codec]
name-overrides = '.google.storage.v2.Storage=StorageControl'
```

### `codec.include-grpc-only-methods` (2 occurrences)
**Purpose**: Includes methods that are gRPC-only (not in REST API)

### `codec.disabled-rustdoc-warnings` (5 occurrences)
**Purpose**: Disables specific rustdoc lints

**Example**:
```toml
[codec]
disabled-rustdoc-warnings = "bare_urls,broken_intra_doc_links"
```

### `codec.disabled-clippy-warnings` (1 occurrence)
**Purpose**: Disables specific clippy lints

### `codec.generate-setter-samples` (2 occurrences)
**Purpose**: Generates example code for setter methods

### `codec.not-for-publication` (3 occurrences)
**Purpose**: Marks crates that should not be published to crates.io

### `codec.extra-modules` (1 occurrence)
**Purpose**: Specifies additional modules to generate

### `codec.default-features` (1 occurrence)
**Purpose**: Specifies default Cargo features

### `codec.root-name` (1 occurrence)
**Purpose**: Overrides the root module name

### `codec.post-process-protos` (1 occurrence)
**Purpose**: Enables post-processing of protobuf definitions

### `codec.detailed-tracing-attributes` (1 occurrence)
**Purpose**: Enables detailed tracing attributes in generated code

## Categorization by Importance

### Critical (Required for correct generation)
1. `general.service-config` - Service metadata
2. `general.specification-format` - Multi-format support
3. `codec.package-name-override` - Naming conventions
4. `codec.template-override` - Code generation strategy
5. `codec.package:*` - Internal dependencies
6. `source.included-ids` / `source.skipped-ids` - Precise control for hybrid libraries

### Important (Affects functionality)
1. `codec.per-service-features` - Build performance for large APIs
2. `codec.has-veneer` - Generation strategy
3. `codec.routing-required` - Runtime behavior
4. `codec.name-overrides` - API naming
5. `discovery.*` - Discovery API support

### Nice-to-have (Quality of life)
1. `codec.disabled-rustdoc-warnings` - Documentation quality
2. `codec.generate-setter-samples` - Developer experience
3. `source.title-override` - Display names
4. `codec.not-for-publication` - Release management

## Recommendation

The librarian.yaml schema should be extended to support these fields. The most critical ones are:

1. **Service config path** - Essential for proper client generation
2. **Specification format** - Needed for Discovery/OpenAPI support
3. **Package naming overrides** - Required for Rust naming conventions
4. **Template overrides** - Different library types need different templates
5. **Internal package dependencies** - Monorepo structure requires this
6. **Method filtering** (included-ids/skipped-ids) - Essential for hybrid libraries

These fields could be added to the `Library.Generate` section or as codec-specific configuration that gets passed through to the container.
