# Python Generation

This document describes Python-specific features and configuration for Librarian.

## Prerequisites

Before generating Python libraries, install these dependencies:

### 1. Install pandoc

Required by the Python GAPIC generator for documentation processing:

```bash
# macOS (using Homebrew)
brew install pandoc

# Ubuntu/Debian
sudo apt-get install pandoc

# Other platforms
# See: https://pandoc.org/installing.html
```

### 2. Install the Python GAPIC generator

```bash
# Using pipx (recommended)
pipx install gapic-generator

# Or using pip with --user flag
pip3 install --user gapic-generator
```

This installs the `protoc-gen-python_gapic` plugin that protoc uses to generate Python client libraries.

### 3. (Optional) Install synthtool

The Python generator can optionally run synthtool to post-process generated code. This is currently disabled by default.

```bash
# Install synthtool
pip3 install --user --break-system-packages gcp-synthtool

# Note: synthtool is no longer actively maintained
```

The post-processor is disabled by default because synthtool is deprecated.

## Configuration

Python libraries use the standard configuration format with Python-specific extensions.

### Basic Example

```yaml
version: v1
language: python

sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98

container:
  image: us-central1-docker.pkg.dev/.../librarian-python
  tag: latest

generate:
  dir: packages/  # Python uses packages/ directory

defaults:
  transport: grpc+rest
  rest_numeric_enums: true

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: google-cloud-secret-manager
    path: packages/google-cloud-secret-manager/
    version: 2.20.0
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
```

### Python-Specific API Fields

API configurations can include Python-specific fields extracted from BUILD.bazel:

```yaml
libraries:
  - name: google-cloud-secret-manager
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          service_config: secretmanager_v1.yaml
          grpc_service_config: secretmanager_grpc_service_config.json
          transport: grpc+rest
          rest_numeric_enums: true
          opt_args:
            - warehouse-package-name=google-cloud-secret-manager
            - python-gapic-namespace=google.cloud
```

**Fields:**
- `opt_args` (array) - Additional generator options passed to protoc-gen-python_gapic
  - `warehouse-package-name` - Package name for PyPI
  - `python-gapic-namespace` - Python namespace (e.g., google.cloud)

### Python-Specific Library Fields

Libraries can include Python-specific configuration:

```yaml
libraries:
  - name: google-cloud-secret-manager
    python:
      remove:
        - packages/google-cloud-secret-manager/google/cloud/secretmanager.py
```

**Fields:**
- `python.remove` (array) - Files to delete after generation (explicit file list, no regex)

### Library Naming Conventions

Python library names follow these conventions:

- **google.cloud APIs**: `google-cloud-{service}`
  - Example: `google/cloud/secretmanager/v1` → `google-cloud-secret-manager`
- **Other APIs**: `google-{service}`
  - Example: `google/bigtable/admin/v2` → `google-bigtable-admin`
- **Override**: Use `warehouse-package-name` in `opt_args` to specify exact name

### Directory Structure

Python repositories typically use this structure:

```
repository/
├── librarian.yaml
├── CHANGELOG.md           # Global changelog
└── packages/
    ├── google-cloud-secret-manager/
    │   ├── google/
    │   │   └── cloud/
    │   │       └── secretmanager/
    │   ├── setup.py
    │   ├── pyproject.toml
    │   ├── README.rst
    │   ├── CHANGELOG.md
    │   ├── docs/
    │   │   └── CHANGELOG.md
    │   └── noxfile.py
    └── google-cloud-pubsub/
        ├── google/
        ├── setup.py
        └── ...
```

## Generation Process

Python generation follows this workflow:

### Phase 1: Code Generation

The container receives commands to run protoc with the Python GAPIC plugin:

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
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

**Generated files:**
- Client library code (`google/cloud/servicename/`)
- `setup.py` with package metadata
- Basic `README.rst` with placeholders
- `noxfile.py` for testing
- Version files (`gapic_version.py`)

### Phase 2: Post-Processing (Optional)

If synthtool is enabled, the container runs post-processing:

```json
{
  "commands": [
    {
      "command": "python3",
      "args": ["-m", "synthtool", "--templates", "/source/google/cloud/secretmanager/v1"]
    }
  ]
}
```

**Post-processing steps:**
- Applies templates to populate README.rst
- Runs formatters (black, isort)
- Copies to final location

### Phase 3: Testing

The container runs nox to validate the generated code:

```json
{
  "commands": [
    {
      "command": "nox",
      "args": ["-s", "unit"]
    }
  ]
}
```

### Host Responsibilities

After the container exits, the librarian CLI:

1. Applies `python.remove` file filtering rules
2. Applies `keep` rules (for hybrid libraries)
3. Copies generated code to the library path

## File Filtering

### Automatic Cleanup

The generator automatically removes all `*_pb2.py` and `*_pb2.pyi` files after generation. These are protobuf-compiled files that should not be included in GAPIC-generated libraries. No configuration is needed for this behavior.

### Manual Removal

For additional files to remove, use `python.remove`:

```yaml
libraries:
  - name: google-cloud-secret-manager
    python:
      remove:
        - packages/google-cloud-secret-manager/google/cloud/secretmanager.py
        - packages/google-cloud-secret-manager/google/cloud/secretmanager_v1/__init__.py
```

**Format:** Explicit file paths (no regex support for simplicity).

### Preserving Files (Hybrid Libraries)

For hybrid libraries with handwritten code, use `keep`:

```yaml
libraries:
  - name: google-cloud-storage
    path: packages/google-cloud-storage/
    generate:
      apis:
        - path: google/storage/v2
    keep:
      - packages/google-cloud-storage/google/cloud/storage/client.py
      - packages/google-cloud-storage/tests/integration/
```

## Scaffolding Files

On first generation, librarian creates these scaffolding files:

### 1. CHANGELOG.md (Package-Level)

```markdown
# Changelog

[PyPI History][1]

[1]: https://pypi.org/project/google-cloud-secret-manager/#history
```

### 2. docs/CHANGELOG.md

Duplicate of package-level CHANGELOG.md for documentation generation.

### 3. CHANGELOG.md (Global)

If the repository uses a global CHANGELOG, librarian adds an entry for the new library:

```markdown
# Changelog

## google-cloud-secret-manager

### [1.0.0] - 2025-01-15

- Initial release

## google-cloud-pubsub
...
```

## Release Process

Python releases follow the standard librarian release workflow with PyPI-specific steps.

### Version Files

Librarian updates these files during release:

- `google/cloud/servicename/gapic_version.py`
- `google/cloud/servicename/version.py`
- `setup.py`
- `pyproject.toml`
- Snippet metadata JSON files

### Changelog Format

Python changelogs use the Keep a Changelog format:

```markdown
# Changelog

## [1.2.0] - 2025-01-15

### Added
- Add Secret rotation support

### Fixed
- Handle nil pointers correctly in GetSecret

## [1.1.0] - 2025-01-01
...
```

### Publishing to PyPI

The release command publishes to PyPI after creating tags:

```bash
# Dry-run (shows what would happen)
librarian release google-cloud-secret-manager

# Actually release
librarian release google-cloud-secret-manager --execute
```

**Steps:**
1. Analyze conventional commits
2. Update version files
3. Update changelogs
4. Create commit
5. Create git tag (`google-cloud-secret-manager/v1.2.0`)
6. Push tag
7. Build package (sdist + wheel)
8. Publish to PyPI

## Container Architecture

The Python container includes:

- Python 3.14
- `protoc` (Protocol Buffer compiler)
- `grpc-tools` (includes `protoc-gen-python` and `protoc-gen-grpc-python`)
- `gapic-generator-python` (Google API client generator)
- `synthtool` (Google's synthesis tool)
- `nox` (Testing framework)

The container is a simple command executor that reads `/commands/commands.json` and executes each command sequentially.

## Common Workflows

### Creating a New Python Library

```bash
# 1. Initialize Python repository (if not already done)
librarian init python

# 2. Create library with initial API
librarian create google-cloud-secret-manager --apis google/cloud/secretmanager/v1

# 3. Verify generation worked
ls packages/google-cloud-secret-manager/
```

### Adding a New API Version

```bash
# 1. Add new API version to existing library
librarian add google-cloud-secret-manager google/cloud/secretmanager/v1beta2

# 2. Regenerate code
librarian generate google-cloud-secret-manager
```

### Creating a Hybrid Library

```bash
# 1. Create library
librarian create google-cloud-storage --apis google/storage/v2

# 2. Add handwritten code
# Edit packages/google-cloud-storage/google/cloud/storage/client.py

# 3. Add keep rules to librarian.yaml
# keep:
#   - packages/google-cloud-storage/google/cloud/storage/client.py

# 4. Regenerate to verify keep rules work
librarian generate google-cloud-storage
```

### Updating to Latest googleapis

```bash
# 1. Update googleapis reference
librarian update --googleapis

# 2. Regenerate all Python libraries
librarian generate --all

# 3. Review changes
git diff

# 4. Commit if everything looks good
git add librarian.yaml packages/
git commit -m "chore: update to latest googleapis"
```

## Troubleshooting

### protoc-gen-python_gapic not found

```
Error: protoc-gen-python_gapic: program not found or is not executable
```

**Solution:** Install gapic-generator-python:
```bash
pipx install gapic-generator
```

### pandoc not found

```
Error: FileNotFoundError: [Errno 2] No such file or directory: 'pandoc'
```

**Solution:** Install pandoc:
```bash
brew install pandoc  # macOS
sudo apt-get install pandoc  # Ubuntu/Debian
```

### nox session failed

```
Error: nox > Command ['python', '-m', 'pytest'] failed with exit code 1
```

**Solutions:**
1. Check test failures in the output
2. Skip tests during development: `librarian generate --skip-tests secretmanager`
3. Fix test failures before releasing

### Import errors after generation

```
ImportError: cannot import name 'SecretManagerServiceClient' from 'google.cloud.secretmanager'
```

**Possible causes:**
1. Missing `__init__.py` files
2. Incorrect namespace in opt_args
3. synthtool post-processing failed

**Solution:** Check generation logs for errors, verify opt_args configuration.

## Best Practices

### 1. Use Consistent Naming

Follow Python package naming conventions:

```yaml
# Good: matches PyPI conventions
name: google-cloud-secret-manager

# Bad: inconsistent casing
name: Google-Cloud-SecretManager
```

### 2. Minimal keep Lists

Only add `keep` entries for actual handwritten code:

```yaml
# Good: only handwritten files
keep:
  - packages/google-cloud-storage/google/cloud/storage/client.py

# Bad: protecting generated files
keep:
  - packages/google-cloud-storage/google/cloud/storage/
```

### 3. Test Before Releasing

Always run tests before releasing:

```bash
# Test locally
librarian generate google-cloud-secret-manager

# Verify tests pass
cd packages/google-cloud-secret-manager
nox -s unit

# Then release
librarian release google-cloud-secret-manager --execute
```

### 4. Use Defaults

Put common settings in `defaults` to avoid repetition:

```yaml
defaults:
  transport: grpc+rest
  rest_numeric_enums: true

libraries:
  - name: google-cloud-secret-manager
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
          # Inherits transport and rest_numeric_enums
```

## Next Steps

- Read [overview.md](overview.md) for general CLI usage
- Read [config.md](config.md) for complete configuration reference
- Read [alternatives.md](alternatives.md) for design decisions
