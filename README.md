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
  one_library_per: api
  transport: grpc+rest
  generate: all

release:
  tag_format: '{name}/v{version}'

libraries: []
```

### 2. Generate everything

The `generate: all` setting auto-discovers and generates all APIs:

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
  - name: google-cloud-vision
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
# With generate: all, these are auto-discovered
# Only list them if you need to add settings
```

**Hybrid** - Generated + handwritten code:
```yaml
- name: google-cloud-vision
  keep:
    - google/cloud/vision_v1/helpers.py
```

**Handwritten** - Fully custom (no generation):
```yaml
- name: pubsub
  path: pubsub/

- name: auth
  path: auth/
```

### Two Modes

**Generate all mode** - Auto-discover and generate everything:
```yaml
defaults:
  generate: all

libraries:
  # Only list libraries that need settings
  - name: google-cloud-vision
    keep: [...]
```

**Explicit mode** - Generate only listed:
```yaml
defaults:
  generate: explicit

libraries:
  - name: google-cloud-secretmanager
    api: google/cloud/secretmanager/v1

  - name: google-cloud-vision
    api: google/cloud/vision/v1

  - name: google-cloud-translate
    api: google/cloud/translate/v3
```

### Bundling Strategies

**API-level** (`one_library_per: api`) - Python/Go default:
- All versions → one library
- `google/cloud/vision/v1` + `google/cloud/vision/v1beta` → `packages/google-cloud-vision/`

**Channel-level** (`one_library_per: channel`) - Rust/Dart default:
- Each channel → separate library
- `google/cloud/vision/v1` → `src/generated/google-cloud-vision-v1/`
- `google/cloud/vision/v1beta` → `src/generated/google-cloud-vision-v1beta/`

## Go Example

```yaml
version: v1
language: go

defaults:
  output: ./
  one_library_per: api
  generate: all

libraries:
  # Exception: handwritten IAM client
  - name: batch
    keep:
      - ^batch/apiv1/iam_policy_client\.go$

  # Handwritten libraries
  - name: pubsub
    path: pubsub/

  - name: storage
    path: storage/
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
one_library_per: api  # Bundle all versions (Python/Go)
one_library_per: channel  # Separate per version (Rust/Dart)
```

**`generate`** - Generation mode:
```yaml
defaults:
  generate: all      # Auto-discover all APIs
  generate: explicit # Only generate listed libraries
```

**`libraries`** - What to configure:
```yaml
# With generate: all, only list libraries that need settings
libraries:
  - name: google-cloud-vision
    keep:
      - google/cloud/vision_v1/helpers.py
      - tests/unit/test_*.py

# With generate: explicit, list all libraries
libraries:
  - name: google-cloud-secretmanager
    api: google/cloud/secretmanager/v1
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
  one_library_per: api
  transport: grpc+rest
  rest_numeric_enums: true
  generate: all

release:
  tag_format: '{name}/v{version}'

name_overrides:
  - api: google/cloud/bigquery/storage/v1
    name: google-cloud-bigquery-storage

libraries:
  # Hybrid libraries (generated + handwritten)
  - name: google-cloud-vision
    keep:
      - google/cloud/vision_v1/helpers.py
      - tests/unit/test_helpers.py

  - name: google-cloud-bigquery-storage
    keep:
      - google/cloud/bigquery_storage_v1/client.py
      - google/cloud/bigquery_storage_v1/reader.py

  # Handwritten libraries
  - name: pubsub
    path: pubsub/

  - name: auth
    path: auth/

  - name: datastore
    path: datastore/
```

**Result:** 200+ libraries generated with only 6 explicit configurations.

## Philosophy

Librarian follows these principles:

1. **Minimal configuration** - Only configure what's different
2. **Clear naming** - Field names describe what they contain
3. **Explicit fields** - Use `name:` and `api:` fields, not YAML keys
4. **Two modes** - Auto-discover everything or list explicitly
5. **Name overrides** - Separate concerns (naming vs. configuration)

## See Also

- [doc/config.md](doc/config.md) - Complete configuration reference
- [doc/alternatives.md](doc/alternatives.md) - Design decisions and alternatives

## Contributing

Librarian is actively developed. Feedback and contributions welcome!
