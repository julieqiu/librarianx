# Rust Generation

This document describes Rust-specific features and configuration for Librarian.

## Prerequisites

Rust generation requires:

- Rust 1.75 or later toolchain
- `cargo` (Rust build tool)
- Sidekick code generator (embedded in container)

These are included in the container, so no manual installation is required when using containers.

## Configuration

Rust libraries use the standard configuration format with Rust-specific extensions.

### Basic Example

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
  dir: src/generated/  # Rust uses src/generated/ directory

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: google-cloud-secretmanager-v1
    path: src/generated/cloud/secretmanager/v1/
    version: 1.1.0
    generate:
      specification_format: protobuf
      apis:
        - path: google/cloud/secretmanager/v1
```

### Rust-Specific API Fields

Rust supports both Protobuf and Discovery-based APIs:

**Protobuf-based API:**
```yaml
libraries:
  - name: google-cloud-secretmanager-v1
    generate:
      specification_format: protobuf
      apis:
        - path: google/cloud/secretmanager/v1
          service_config: secretmanager_v1.yaml
```

**Discovery-based API:**
```yaml
libraries:
  - name: google-cloud-compute-v1
    generate:
      specification_format: disco
      apis:
        - path: discoveries/compute.v1.json
          service_config: google/cloud/compute/v1/compute_v1.yaml
```

### Rust-Specific Library Fields

Libraries can include Rust-specific configuration:

```yaml
libraries:
  - name: google-cloud-secretmanager-v1
    rust:
      codec:
        copyright_year: "2025"
        per_service_features: true
        disabled_rustdoc_warnings: bare_urls,broken_intra_doc_links
        disabled_clippy_warnings: doc_lazy_continuation
```

**Common fields:**
- `rust.codec.copyright_year` (string) - Copyright year for generated files
- `rust.codec.per_service_features` (boolean) - Enable per-service Cargo features
- `rust.codec.disabled_rustdoc_warnings` (string) - Comma-separated list of rustdoc warnings to disable
- `rust.codec.disabled_clippy_warnings` (string) - Comma-separated list of clippy warnings to disable

For comprehensive Rust configuration options, see [sidekick-how-to-guide.md](sidekick-how-to-guide.md).

### Library Naming Conventions

Rust library names follow these conventions:

- **API-versioned crates**: `google-cloud-{service}-{version}`
  - Example: `google/cloud/secretmanager/v1` → `google-cloud-secretmanager-v1`
  - Example: `google/bigtable/admin/v2` → `google-bigtable-admin-v2`

### Directory Structure

Rust repositories typically use this structure:

```
repository/
├── librarian.yaml
├── Cargo.toml          # Workspace Cargo.toml
└── src/
    └── generated/
        └── cloud/
            └── secretmanager/
                └── v1/
                    ├── Cargo.toml
                    ├── README.md
                    └── src/
                        ├── lib.rs
                        └── generated/
```

## Generation Process

Rust generation uses Sidekick, a Rust code generator maintained in `internal/sidekick`.

### Phase 1: Code Generation

The container receives commands to run Sidekick:

```json
{
  "commands": [
    {
      "command": "sidekick",
      "args": [
        "generate",
        "--api-path=/source/google/cloud/secretmanager/v1",
        "--service-config=/source/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
        "--output=/output"
      ]
    }
  ]
}
```

**Generated files:**
- Complete Rust crate including:
  - `Cargo.toml` with dependencies
  - `src/lib.rs` and generated modules
  - `README.md`
  - Tests

### Phase 2: Formatting

```json
{
  "commands": [
    {
      "command": "cargo",
      "args": ["fmt"]
    },
    {
      "command": "taplo",
      "args": ["fmt", "Cargo.toml"]
    }
  ]
}
```

### Phase 3: Testing and Validation

```json
{
  "commands": [
    {
      "command": "cargo",
      "args": ["test"]
    },
    {
      "command": "cargo",
      "args": ["clippy"]
    },
    {
      "command": "cargo",
      "args": ["doc"]
    },
    {
      "command": "typos",
      "args": ["."]
    }
  ]
}
```

### Host Responsibilities

After the container exits, the librarian CLI:

1. Applies any file filtering rules (Rust typically doesn't need filtering)
2. Copies generated code to the library path

## File Filtering

Unlike Go and Python, Rust generation typically doesn't require file filtering. Sidekick generates a clean output directory without needing explicit `keep` or `remove` rules.

## Scaffolding Files

Rust generation creates a complete crate in one step. Sidekick generates:

- `Cargo.toml` with package metadata and dependencies
- `README.md` with usage examples
- `src/lib.rs` and all module files
- Tests

No separate scaffolding step is needed.

## Release Process

Rust releases follow the standard librarian release workflow with crates.io-specific steps.

### Version Files

Librarian updates `Cargo.toml` during release:

```toml
[package]
name = "google-cloud-secretmanager-v1"
version = "1.1.0"  # Updated by librarian release
```

### Tag Format

Rust uses crate-name-based tags:

```yaml
release:
  tag_format: '{name}/v{version}'
```

Examples:
- `google-cloud-secretmanager-v1/v1.1.0`
- `google-cloud-compute-v1/v0.2.1`

### Publishing to crates.io

The release command publishes to crates.io after creating tags:

```bash
# Dry-run (shows what would happen)
librarian release google-cloud-secretmanager-v1

# Actually release
librarian release google-cloud-secretmanager-v1 --execute
```

**Steps:**
1. Analyze conventional commits
2. Update Cargo.toml version
3. Update changelogs
4. Create commit
5. Create git tag
6. Push tag
7. Publish to crates.io

## Container Architecture

The Rust container includes:

- Rust 1.75 toolchain
- `cargo` (Rust build tool)
- `taplo-cli` (TOML formatter)
- `typos-cli` (Spell checker)
- Sidekick code generator (pure Go, embedded in container)

The container is a simple command executor that reads `/commands/commands.json` and executes each command sequentially.

## Common Workflows

### Creating a New Rust Library

```bash
# 1. Initialize Rust repository (if not already done)
librarian init rust

# 2. Create library with initial API
librarian create google-cloud-secretmanager-v1 --apis google/cloud/secretmanager/v1

# 3. Verify generation worked
ls src/generated/cloud/secretmanager/v1/
```

### Creating a Discovery-Based Library

```bash
# 1. Create Compute library (uses Discovery)
librarian create google-cloud-compute-v1 \
  --apis discoveries/compute.v1.json \
  --specification-format disco

# 2. Verify generation
ls src/generated/cloud/compute/v1/
```

### Updating to Latest googleapis

```bash
# 1. Update googleapis and discovery references
librarian update --googleapis --discovery

# 2. Regenerate all Rust libraries
librarian generate --all

# 3. Review changes
git diff

# 4. Commit if everything looks good
git add librarian.yaml src/generated/
git commit -m "chore: update to latest googleapis"
```

## Troubleshooting

### Sidekick generation failed

```
Error: sidekick generate failed with exit code 1
```

**Solutions:**
1. Check service_config path is correct
2. Verify API path exists in googleapis
3. Check Sidekick logs for specific errors

### Cargo build failed

```
Error: cargo build failed with exit code 101
```

**Solutions:**
1. Check for compilation errors in generated code
2. Verify dependencies in Cargo.toml
3. Run `cargo check` for detailed errors

### Publishing to crates.io failed

```
Error: crates.io returned 403 Forbidden
```

**Solutions:**
1. Verify you're authenticated: `cargo login`
2. Check you have publishing permissions for the crate
3. Ensure version doesn't already exist on crates.io

## Sidekick Integration

Rust generation is powered by Sidekick, a Rust code generator maintained in this repository.

**Key features:**
- Generates idiomatic Rust code from Protocol Buffers and Discovery documents
- Supports both gRPC and REST transports
- Handles long-running operations (LRO)
- Generates Cargo features for optional functionality

For detailed Sidekick documentation, see:
- [sidekick.md](sidekick.md) - Sidekick overview
- [sidekick-how-to-guide.md](sidekick-how-to-guide.md) - Detailed configuration guide
- [sidekick-merge-strategy.md](sidekick-merge-strategy.md) - Merge strategies for updates

## Best Practices

### 1. Use Per-Crate Versioning

Rust crates are typically versioned independently:

```yaml
libraries:
  - name: google-cloud-secretmanager-v1
    version: 1.1.0  # Independent version

  - name: google-cloud-pubsub-v1
    version: 2.3.0  # Different version
```

### 2. Enable Per-Service Features

For large APIs, enable per-service features:

```yaml
rust:
  codec:
    per_service_features: true
    default_features:
      - instances
      - projects
```

### 3. Test Before Releasing

Always run tests before releasing:

```bash
# Test locally
librarian generate google-cloud-secretmanager-v1
cd src/generated/cloud/secretmanager/v1
cargo test

# Then release
librarian release google-cloud-secretmanager-v1 --execute
```

### 4. Configure Linting

Disable warnings for generated code:

```yaml
rust:
  codec:
    disabled_rustdoc_warnings: bare_urls,broken_intra_doc_links
    disabled_clippy_warnings: doc_lazy_continuation
```

## Next Steps

- Read [userguide.md](userguide.md) for general CLI usage
- Read [config.md](config.md) for complete configuration reference
- Read [sidekick-how-to-guide.md](sidekick-how-to-guide.md) for detailed Rust configuration
- Read [alternatives.md](alternatives.md) for design decisions
