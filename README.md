# Librarian

Librarian is a tool for managing Google Cloud client libraries across multiple languages. It handles code generation from API definitions, version management, and publishing to package registries.

## Installation

```bash
$ go install github.com/julieqiu/librarian/cmd/librarian@latest
```

## Quick Start: Python

Let's build Python client libraries for Google Cloud APIs.

### 1. Initialize

```bash
$ mkdir my-python-libs && cd my-python-libs
$ librarian init python
Created librarian.yaml
```

This creates a minimal configuration:

```yaml
version: v1
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/main.tar.gz
    sha256: ...

defaults:
  output: packages/
  one_library_per: service
  transport: grpc+rest

release:
  tag_format: '{name}/v{version}'

libraries: []
```

### 2. Generate everything

Add the wildcard to generate all APIs:

```bash
$ vim librarian.yaml
# Add under libraries:
libraries:
  - '*'
```

```bash
$ librarian generate --all
Discovering APIs from googleapis...
Found 237 APIs, generating 182 libraries...
  ✓ packages/google-cloud-secretmanager/
  ✓ packages/google-cloud-vision/
  ✓ packages/google-cloud-translate/
  ... 179 more ...
Done.
```

### 3. Add handwritten code

```bash
$ cd packages/google-cloud-vision/
$ vim google/cloud/vision_v1/helpers.py
# Add custom helper functions
```

### 4. Protect handwritten code

```bash
$ vim librarian.yaml
```

```yaml
libraries:
  - '*'

  - packages/google-cloud-vision:
      keep:
        - google/cloud/vision_v1/helpers.py
        - tests/unit/test_helpers.py
```

### 5. Regenerate

```bash
$ librarian generate --all
Regenerating packages/google-cloud-vision...
  Preserving: helpers.py, test_helpers.py
  ✓ Generated
```

Your handwritten code is preserved!

## How It Works

### Library Types

**Generated** - Created from googleapis APIs:
```yaml
- packages/google-cloud-vision
```

**Hybrid** - Generated + handwritten code:
```yaml
- packages/google-cloud-vision:
    keep:
      - google/cloud/vision_v1/helpers.py
```

**Handwritten** - Fully custom (no generation):
```yaml
- pubsub/
- auth/
```

### Two Modes

**Wildcard mode** - Generate everything:
```yaml
libraries:
  - '*'  # Generate all discovered APIs

  # Only list exceptions
  - packages/google-cloud-vision:
      keep: [...]
```

**Explicit mode** - Generate only listed:
```yaml
libraries:
  - packages/google-cloud-secretmanager
  - packages/google-cloud-vision
  - packages/google-cloud-translate
```

### Bundling Strategies

**Service-level** (`one_library_per: service`) - Python/Go default:
- All versions → one library
- `google/cloud/vision/v1` + `google/cloud/vision/v1beta` → `packages/google-cloud-vision/`

**Version-level** (`one_library_per: version`) - Rust/Dart default:
- Each version → separate library
- `google/cloud/vision/v1` → `src/generated/google-cloud-vision-v1/`
- `google/cloud/vision/v1beta` → `src/generated/google-cloud-vision-v1beta/`

## Go Example

```yaml
version: v1
language: go

defaults:
  output: ./
  one_library_per: service

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

## Configuration

See [doc/config.md](doc/config.md) for complete configuration reference.

### Key Fields

**`output`** - Where generated code goes:
```yaml
defaults:
  output: packages/  # Python: packages/google-cloud-*/
  output: ./         # Go: */
  output: src/generated/  # Rust: src/generated/google-cloud-*-v*/
```

**`one_library_per`** - Bundling strategy:
```yaml
one_library_per: service  # Bundle all versions (Python/Go)
one_library_per: version  # Separate per version (Rust/Dart)
```

**`libraries`** - What to generate:
```yaml
libraries:
  - '*'  # Everything (wildcard)

  # Or explicit list
  - packages/google-cloud-secretmanager
  - packages/google-cloud-vision
```

**`keep`** - Protect handwritten code:
```yaml
- packages/google-cloud-vision:
    keep:
      - google/cloud/vision_v1/helpers.py
      - tests/unit/test_*.py
```

## Commands

```bash
# Initialize repository
$ librarian init <language>

# Generate libraries
$ librarian generate --all
$ librarian generate packages/google-cloud-vision

# Release
$ librarian release packages/google-cloud-vision
```

## Real-World Example

```yaml
version: v1
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
    sha256: 81e6057...

defaults:
  output: packages/
  one_library_per: service
  transport: grpc+rest
  rest_numeric_enums: true

release:
  tag_format: '{name}/v{version}'

libraries:
  - '*'  # Generate ~200 libraries

  # Exception: hybrid libraries
  - packages/google-cloud-vision:
      keep:
        - google/cloud/vision_v1/helpers.py
        - tests/unit/test_helpers.py

  - packages/google-cloud-bigquery-storage:
      keep:
        - google/cloud/bigquery_storage_v1/client.py
        - google/cloud/bigquery_storage_v1/reader.py

  # Exception: handwritten libraries
  - pubsub/
  - auth/
  - datastore/
```

**Result:** 200+ libraries generated with only 5 explicit configurations.

## Philosophy

Librarian follows these principles:

1. **Minimal configuration** - Only configure what's different
2. **Filesystem as truth** - Reference libraries by their paths
3. **Clear naming** - Field names describe what they contain
4. **Explicit intent** - Use `*` wildcard, not boolean flags
5. **One list** - All libraries (generated and handwritten) together

## See Also

- [doc/config.md](doc/config.md) - Complete configuration reference
- [doc/alternatives.md](doc/alternatives.md) - Design decisions and alternatives

## Contributing

Librarian is actively developed. Feedback and contributions welcome!
