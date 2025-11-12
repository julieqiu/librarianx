# Configuration System

This document describes the configuration system for Librarian, which uses a
single `librarian.yaml` file instead of flags for all configuration.

## Overview

The configuration system uses a single file at the repository root:

- **`librarian.yaml`** - Defines repository-wide settings (language, container images,
  googleapis references) and all library configurations

This design eliminates the need for command-line flags and makes all configuration
transparent and version-controlled.

## Configuration Structure

The configuration file lives at `librarian.yaml` at the repository root.

### Example: Release-only repository

```yaml
version: v0.5.0
language: go

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: custom-tool
    version: null
```

**What this enables:**
- `librarian add <name>` - Track handwritten code for release
- `librarian release <name>` - Release libraries (dry-run by default)
- `librarian release <name> --execute` - Actually perform the release

### Example: Repository with code generation

```yaml
version: v0.5.0
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
  discovery:
    url: https://github.com/googleapis/discovery-artifact-manager/archive/f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0.tar.gz
    sha256: 867048ec8f0850a4d77ad836319e4c0a0c624928611af8a900cd77e676164e8e

container:
  image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator
  tag: latest

generate:
  output_dir: packages/
  defaults:
    transport: grpc+rest
    rest_numeric_enums: true
    release_level: stable

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: google-cloud-secret-manager
    version: 2.20.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          name_pretty: "Secret Manager"
          product_documentation: "https://cloud.google.com/secret-manager/docs"
          release_level: stable
```

**What this enables:**
- `librarian add <name> <api>` - Add APIs to library
- `librarian create <name> <api>` - Create library and generate code
- `librarian generate <name>` - Regenerate existing code
- `librarian test <name>` - Run tests for a library
- `librarian update --googleapis` - Update source references
- `librarian release <name>` - Release libraries (dry-run by default)
- `librarian release <name> --execute` - Actually perform the release

### Configuration Fields

#### Top-level fields

- `version` - Version of librarian that created this config
- `language` - Repository language (`go`, `python`, `rust`)
- `libraries` - Array of library configurations (see Library Configuration below)

#### `sources` section (optional)

When present, defines external source repositories used for code generation. This section is managed by the `librarian update` command.

- `googleapis.url` - URL to googleapis tarball (e.g., `https://github.com/googleapis/googleapis/archive/{commit}.tar.gz`)
- `googleapis.sha256` - SHA256 hash for integrity verification
- `discovery.url` - URL to discovery-artifact-manager tarball
- `discovery.sha256` - SHA256 hash for integrity verification

**Design rationale**:
- **Immutable references** - URLs with commit SHAs ensure reproducible builds
- **Caching** - Downloads are cached by SHA256 in `~/Library/Caches/librarian/downloads/` to avoid repeated downloads
- **No race conditions** - Multiple concurrent generations verify the same immutable tarball
- **Single source of truth** - All libraries use the same googleapis/discovery versions from the repository config
- **Separate concerns** - Source versions are separate from generation infrastructure

#### `container` section (optional)

When present, defines the container image used for code generation.

- `image` - Container registry path (without tag)
- `tag` - Container image tag (e.g., `latest`, `v1.0.0`)

#### `generate` section (optional)

When present, enables code generation commands. This section defines generation settings.

- `output_dir` - Directory where generated code is written (relative to repository root)
- `defaults` - Default values applied to all libraries (see GenerateDefaults below)

#### `release` section (optional)

When present, enables release commands.

- `tag_format` - Template for git tags (e.g., `'{name}/v{version}'` or `'{name}-v{version}'`)
  - Supported placeholders: `{name}` and `{version}`
  - The global default is `'{name}/v{version}'` for Go repositories
  - Some modules require custom formats to avoid double version paths. These exceptions should be handled in code:
    - `bigquery/v2` uses `bigquery/v{version}` (instead of `bigquery/v2/v{version}`)
    - `pubsub/v2` uses `pubsub/v{version}` (instead of `pubsub/v2/v{version}`)
    - `root-module` uses `v{version}` (no id prefix)

## Library Configuration

Each library is defined in the `libraries` array in `librarian.yaml`.

### Example: Handwritten code (release-only)

```yaml
libraries:
  - name: custom-tool
    version: null
```

This library only has a `version` field, so it can be released but not regenerated.

### Example: Generated code

```yaml
libraries:
  - name: google-cloud-secret-manager
    version: 2.20.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          grpc_service_config: secretmanager_grpc_service_config.json
          service_yaml: secretmanager_v1.yaml
          transport: grpc+rest
          rest_numeric_enums: true
          name_pretty: "Secret Manager"
          product_documentation: "https://cloud.google.com/secret-manager/docs"
          release_level: stable
          opt_args:
            - warehouse-package-name=google-cloud-secret-manager
        - path: google/cloud/secretmanager/v1beta2
          grpc_service_config: secretmanager_grpc_service_config.json
          service_yaml: secretmanager_v1beta2.yaml
          transport: grpc+rest
          rest_numeric_enums: true
          release_level: preview
          opt_args:
            - warehouse-package-name=google-cloud-secret-manager
      keep:
        - README.md
        - docs/
      remove:
        - temp.txt
```

### Example: Disabled library

When a library's generation is broken (e.g., due to BUILD.bazel issues, API changes, or generator bugs), you can temporarily disable it while preserving its configuration:

```yaml
libraries:
  # Disabled: Generator failing on optional field handling
  # See: https://github.com/googleapis/google-cloud-go/issues/12345
  - name: aiplatform
    version: 1.2.3
    disabled: true
    generate:
      apis:
        - path: google/cloud/aiplatform/v1
          name_pretty: "Vertex AI"
          product_documentation: "https://cloud.google.com/vertex-ai/docs"
```

**Behavior:**
- `librarian generate --all` - Skips aiplatform with a warning
- `librarian generate aiplatform` - Returns error: "aiplatform is disabled. See issue link in comment for details."
- `librarian release aiplatform` - Still works (only generation is disabled)
- Configuration is preserved for when the issue is resolved

**Requirements:**
- The `disabled` field MUST be accompanied by a comment with an issue link
- The comment should explain why the library is disabled
- This ensures the team can track and resolve the underlying issue

### Library Configuration Fields

#### Library fields

- `name` - Name of the library
- `path` - Directory path relative to repository root (optional, derived from name and generate.output_dir if empty)
- `version` - Current released version (pointer, null if never released)
- `disabled` - Skip this library during generation (optional, boolean). When set to `true`, generation is disabled. MUST include a comment with an issue link explaining why the library is disabled

#### `generate` section (optional)

When present, this library can be regenerated with `librarian generate`.

**API Configuration** (`apis` array):

Each API entry contains configuration extracted from BUILD.bazel during `librarian add` or `librarian create`:

- `path` - API path relative to googleapis root (e.g., `google/cloud/secretmanager/v1`)
- `grpc_service_config` - Retry configuration file path (relative to API directory)
- `service_yaml` - Service configuration file path
- `transport` - Transport protocol (e.g., `grpc+rest`, `grpc`)
- `rest_numeric_enums` - Whether to use numeric enums in REST
- `opt_args` - Additional generator options (array of strings)

**Metadata fields on each API**:

Library metadata used to generate documentation and configure the package:

- `name_pretty` - Human-readable name (e.g., "Secret Manager")
- `product_documentation` - URL to product documentation
- `client_documentation` - URL to client library documentation
- `issue_tracker` - URL to issue tracker
- `release_level` - Release level: `stable` or `preview`
- `library_type` - Library type: `GAPIC_AUTO` or `GAPIC_COMBO`
- `api_id` - API ID (e.g., `secretmanager.googleapis.com`)
- `api_shortname` - Short API name (e.g., `secretmanager`)
- `api_description` - Description of the API
- `default_version` - Default API version (e.g., `v1`)

**File filtering**:

- `keep` - Files/directories not overwritten during generation (array of patterns)
- `remove` - Files/directories deleted after generation (array of patterns)

**Automatic cleanup**: The generator automatically removes all `*_pb2.py` and `*_pb2.pyi` files after generation. These are protobuf-compiled files that should not be included in GAPIC-generated libraries. No configuration is needed for this behavior.

**Note**: Library configuration does NOT store googleapis/discovery URLs or SHA256 hashes. These are stored in the top-level `sources` section to ensure all libraries use the same source versions. This design:
- Prevents duplication across all library configs
- Ensures consistency - all libraries generated from the same source versions
- Prevents race conditions - no per-library source state to get out of sync
- Simplifies updates - change source versions in one place (via `librarian update`), regenerate all libraries

## How Configuration Works

### Creating a library without APIs (release-only)

```bash
librarian add my-tool
```

This adds a library entry to `librarian.yaml`:

```yaml
libraries:
  - name: my-tool
    version: null
```

The library can be released but not regenerated.

### Creating a library with APIs (generated code)

Use `create` for initial library creation:

```bash
librarian create google-cloud-secret-manager google/cloud/secretmanager/v1
```

Later, add more APIs incrementally:

```bash
librarian add google-cloud-secret-manager google/cloud/secretmanager/v1beta2
librarian generate google-cloud-secret-manager
```

Both commands:

1. Read `librarian.yaml` to get googleapis location from `sources` section
2. Download googleapis tarball if needed (cached by SHA256)
3. For each API path:
   - Read `BUILD.bazel` file in that directory
   - Extract configuration from language-specific gapic rule (e.g., `py_gapic_library`)
   - Add API configuration to library entry
4. Add/update library entry to `librarian.yaml` with all extracted config

The difference:
- `create` creates a new library and generates code immediately (syntactic sugar for `add` + `generate`); fails if directory exists
- `add` only updates the config for existing libraries; run `librarian generate` afterward to generate code

**Key insight**: BUILD.bazel parsing happens only once during `librarian add` or `librarian create`. The
extracted configuration is saved to `librarian.yaml` and reused for all subsequent
`librarian generate` commands. This makes generation faster and ensures reproducibility
even if BUILD.bazel files change upstream.

### Generating code

```bash
librarian generate google-cloud-secret-manager
```

This:

1. Reads `librarian.yaml` (repository config with `sources`, `container`, and `generate` sections, plus library config)
2. Finds the library entry for `google-cloud-secret-manager`
3. Ensures sources are available:
   - Downloads googleapis tarball from `sources.googleapis.url`
   - Verifies SHA256 matches `sources.googleapis.sha256`
   - Caches by SHA256 in `~/Library/Caches/librarian/downloads/`
4. Builds `generate.json` from API configurations in library's `generate` section
5. Runs generator container **once** with `generate.json`
6. Applies keep/remove rules after container exits
7. Copies final output to library directory

**No generation state** is written to the library entry. The repository config's `sources` section serves as the single source of truth for what was used.

## Container Interface

The generator container is a Docker image that implements the Librarian container
contract using a **command-based architecture**.

### What the container receives

**Mounts:**

- `/commands/commands.json` - Contains commands to execute (read-only)
- `/source` - Read-only googleapis repository
- `/output` - Directory where container writes generated code

**Container execution:**

The librarian CLI invokes the container multiple times during generation, each time
with a different commands.json file. The container reads the commands and executes
them sequentially.

**Example invocation:**

```bash
docker run \
  -v /path/to/commands.json:/commands/commands.json:ro \
  -v /path/to/googleapis:/source:ro \
  -v /path/to/output:/output \
  python-generator:latest
```

### commands.json

The container reads `/commands/commands.json` which contains explicit commands to execute.

**Example (Python code generation):**

```json
{
  "commands": [
    {
      "command": "python3",
      "args": [
        "-m", "grpc_tools.protoc",
        "--proto_path=/source",
        "--python_gapic_out=/output",
        "--python_gapic_opt=service-config=/source/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
        "--python_gapic_opt=retry-config=/source/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
        "--python_gapic_opt=transport=grpc+rest",
        "--python_gapic_opt=rest-numeric-enums",
        "--python_gapic_opt=warehouse-package-name=google-cloud-secret-manager",
        "/source/google/cloud/secretmanager/v1/resources.proto",
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

**Example (Go code generation):**

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
        "/source/google/cloud/secretmanager/v1/resources.proto",
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

### Container responsibilities

The container must:

1. Read `/commands/commands.json`
2. Execute each command sequentially
3. Exit when all commands complete

The container does NOT:

- Parse BUILD.bazel files (already done by librarian CLI)
- Clone googleapis (already mounted at `/source`)
- Parse `.librarian.yaml` files (already done by librarian CLI - commands are pre-built)
- Apply keep/remove/exclude rules (done by librarian CLI after container exits)
- Update `.librarian.yaml` (done by librarian CLI after container exits)

### Multiple invocations

The host CLI calls the container multiple times during generation, each with different commands:

1. **Code generation** - Run protoc/generators
2. **Post-processing** - Run formatters, templates
3. **Testing** - Run tests and validation

Between invocations, the host applies file filtering rules and manages staging directories.

See [doc/generate.md](generate.md) for detailed generation flows for Python, Go, and Rust.

## Key Design Decisions

### Why parse BUILD.bazel in the CLI?

- Parsing happens once during `librarian add` or `librarian create`, not on every generation
- Configuration is saved in `librarian.yaml` for transparency and reproducibility
- Users can manually edit configuration if BUILD.bazel is incorrect
- Container remains simple - just executes protoc with provided options
- Go has excellent Bazel parsing libraries (`github.com/bazelbuild/buildtools/build`)

### Why use a separate `sources` section?

The new design stores source URLs/SHA256 hashes in a dedicated `sources` section in the repository config, not in each artifact's config. This provides:

- **Single source of truth** - One place to update source versions for all libraries (via `librarian update`)
- **No duplication** - Don't repeat the same URL/SHA256 across hundreds of library configs
- **Prevents race conditions** - No mutable shared cache or per-library source state
- **Consistent generation** - All libraries always use the same source versions
- **Simpler updates** - Change sources in one file, regenerate all libraries
- **Clear separation** - Source versions (external dependencies) are separate from generation settings (infrastructure)
- **Git history** - `git log librarian.yaml` shows when sources changed for the entire repository

This follows the same pattern as sidekick (see `.sidekick.toml` in google-cloud-rust), where the root config contains `googleapis-root` and `googleapis-sha256`, and per-library configs contain only API-specific settings.

### Why use regex patterns for keep/remove/exclude?

Regex patterns provide:

- **Flexibility** - Match patterns like `.*_test\.py` or `docs/.*\.md`
- **Precision** - Exact control over what files are affected
- **Simplicity** - Single pattern can match many files
- **Familiarity** - Developers understand regex

### Why use a command-based architecture?

The command-based architecture provides:

- **Simplicity** - Container just executes commands, no parsing/interpretation needed
- **Language-agnostic container code** - Same Go code runs in all language containers
- **Explicit control** - Host decides exactly what commands to run
- **Debuggability** - Commands are transparent and can be inspected
- **Flexibility** - Easy to add new commands or change execution order

## Migration from Old System

The old system used flags and `.repo-metadata.json` files. The new system uses
a single `librarian.yaml` file for all configuration.

**Old system:**

```bash
# Flags required for every command
librarian generate \
  --api secretmanager/v1 \
  --api-source=/path/to/googleapis \
  --library secretmanager \
  --image python-gen:latest
```

**New system:**

```bash
# All configuration in librarian.yaml
librarian generate google-cloud-secret-manager
```

**Migration steps:**

1. Run `librarian init <language>` to create `librarian.yaml`
2. For each existing library:
   - Run `librarian add <name> <apis>` to add library entry
   - Verify configuration matches old `.repo-metadata.json`
3. Delete old `.repo-metadata.json` files
4. Update CI/CD pipelines to use new commands without flags

## Completed Improvements

### ✅ Adopted command-based container architecture

The container interface uses a command-based architecture:
- `/commands/commands.json` contains **explicit commands to execute**
- Container is a simple command executor (language-agnostic)
- Multiple container invocations per library (generate → format → test)
- Librarian team maintains container (simple Go code)
- Language teams maintain generator tools (gapic-generator-python, gapic-generator-go, Sidekick)

**Key benefits:**
- **Simplicity**: Container is ~30 lines of Go, no language expertise needed
- **Explicit**: Commands are visible in commands.json, easy to debug
- **Ownership**: Librarian team owns container, language teams own generators
- **Flexibility**: Easy to add/remove/reorder commands without changing container

### ✅ Single configuration file

All configuration lives in one `librarian.yaml` file at the repository root:

```yaml
version: v0.5.0
language: go

container:
  image: go-generator
  tag: latest

generate:
  output_dir: ./
  defaults:
    transport: grpc+rest

sources:
  googleapis:
    url: https://...
    sha256: ...

libraries:
  - name: secretmanager
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
```

**Benefit**: Single source of truth, easier discovery, litmus test for configuration complexity.
