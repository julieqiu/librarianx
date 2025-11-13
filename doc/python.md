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

The Python generator can optionally run synthtool to post-process generated code.
This is currently disabled by default.

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

The generator automatically removes all `*_pb2.py` and `*_pb2.pyi` files after generation.
These are protobuf-compiled files that should not be included in GAPIC-generated libraries.
No configuration is needed for this behavior.

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

Python releases use the `librarian release` command to prepare releases:

1. **`librarian release`** - Prepares release (updates files, commits, tags, pushes)

**Note:** Python libraries do NOT use `librarian publish`.
Publishing to PyPI is handled separately (typically via CI/CD automation).

### Current Implementation Status

**Implemented (in `internal/release/python/release.go`):**
- ✅ Version file updates (gapic_version.py, version.py, pyproject.toml, setup.py)
- ✅ Package CHANGELOG.md update

**TODO:**
- ⏳ Run tests (nox) before release
- ⏳ Update docs/CHANGELOG.md (duplicate)
- ⏳ Update global CHANGELOG.md (monorepo)
- ⏳ Update snippet metadata JSON files
- ⏳ PEP 440 version normalization (1.16.0-rc.1 → 1.16.0rc1)

### Python-Specific Files Updated

When releasing a Python library, these files are updated:

```
packages/google-cloud-secret-manager/
├── CHANGELOG.md                               # Package changelog
├── docs/
│   └── CHANGELOG.md                           # Duplicate for docs
├── google/cloud/secretmanager/
│   ├── gapic_version.py                       # Version constant
│   └── version.py                             # Version constant (if exists)
├── pyproject.toml                             # Version in [project]
└── setup.py                                   # Version (if exists, legacy)

# Global changelog (if exists)
CHANGELOG.md                                    # Monorepo changelog
```

### Current Implementation Analysis

#### What's Working

The current `internal/release/python/release.go` implements:

**1. Version File Updates (`updateVersionFiles`)**
- ✅ Finds and updates all version files using glob patterns
- ✅ Handles `**/gapic_version.py` (recursive search)
- ✅ Handles `**/version.py` (recursive search)
- ✅ Handles `pyproject.toml`
- ✅ Handles `setup.py` (legacy)
- ✅ Uses regex replacement with correct formatting per file type

**2. Changelog Updates (`updateChangelog`)**
- ✅ Creates CHANGELOG.md if doesn't exist
- ✅ Appends to existing CHANGELOG.md
- ✅ Inserts new entries at top (after header)
- ✅ Groups changes by type

**Strengths:**
- Simple, focused implementation
- Good file pattern matching with `**` support
- Proper handling of missing files

#### What's Missing

Comparing to the full design, here's what needs to be added:

**1. Testing (`runPythonTests` - Not Implemented)**
```go
// TODO: Add before updateVersionFiles()
if err := runPythonTests(ctx, lib.Path); err != nil {
    return fmt.Errorf("tests failed: %w", err)
}
```

**2. docs/CHANGELOG.md Update (Not Implemented)**
- Current: Only updates `packages/lib/CHANGELOG.md`
- Needed: Also update `packages/lib/docs/CHANGELOG.md`
- Solution: Add after main changelog update

**3. Global CHANGELOG.md Update (Not Implemented)**
- Current: No global changelog support
- Needed: Update monorepo `CHANGELOG.md` at repository root
- Python monorepos maintain a global changelog with sections per library

**4. PEP 440 Version Normalization (Not Implemented)**
- Current: Uses version string as-is
- Needed: Normalize pre-release versions
  - `1.16.0-rc.1` → `1.16.0rc1`
  - `1.16.0-alpha.1` → `1.16.0a1`
  - `1.16.0-beta.1` → `1.16.0b1`

#### Changelog Format Improvements

**Current format:**
```markdown
## 1.16.0

* feat: add rotation
* fix: handle nil
```

**Google Cloud Python format (should match):**
```markdown
## [1.16.0](https://github.com/googleapis/google-cloud-python/releases/tag/google-cloud-secret-manager-v1.16.0) (2025-11-12)

### Features

* add rotation ([abc1234](https://github.com/googleapis/google-cloud-python/commit/abc1234))

### Bug Fixes

* handle nil ([def5678](https://github.com/googleapis/google-cloud-python/commit/def5678))
```

**Differences:**
- Missing: Version links to GitHub releases
- Missing: Date in heading
- Missing: Grouped sections (Features, Bug Fixes, Documentation)
- Missing: Commit links

### Implementation: internal/release/python.go

#### Release() Function - Current Implementation

The current implementation is minimal and focused:

```go
package release

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
)

func Release(ctx context.Context, lib *config.Library, version string, changes []*Change) error {
    // 1. Run tests
    if err := runPythonTests(ctx, lib.Path); err != nil {
        return fmt.Errorf("tests failed: %w", err)
    }

    // 2. Update version files
    if err := updateVersionFiles(lib.Path, version); err != nil {
        return fmt.Errorf("failed to update version files: %w", err)
    }

    // 3. Update changelogs
    if err := updateChangelogs(lib, version, changes); err != nil {
        return fmt.Errorf("failed to update changelogs: %w", err)
    }

    // 4. Update snippet metadata (if exists)
    if err := updateSnippetMetadata(lib, version); err != nil {
        return fmt.Errorf("failed to update snippet metadata: %w", err)
    }

    return nil
}
```

#### 1. Running Tests

```go
func runPythonTests(ctx context.Context, libPath string) error {
    // Run nox test session
    cmd := exec.CommandContext(ctx, "nox", "-s", "unit")
    cmd.Dir = libPath
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("nox unit tests failed: %w", err)
    }

    return nil
}
```

**Testing strategy:**
- Run `nox -s unit` (fast unit tests)
- Skip slow integration tests (run in CI instead)
- User can skip with `--skip-tests` flag

#### 2. Updating Version Files

```go
func updateVersionFiles(libPath, version string) error {
    // Update each version file
    files := []struct {
        path    string
        updater func(string, string) error
    }{
        {
            path:    filepath.Join(libPath, "pyproject.toml"),
            updater: updatePyprojectToml,
        },
        {
            path:    filepath.Join(libPath, "setup.py"),
            updater: updateSetupPy,
        },
        {
            path:    findGapicVersionFile(libPath),
            updater: updateGapicVersion,
        },
        {
            path:    findVersionFile(libPath),
            updater: updateVersionPy,
        },
    }

    for _, f := range files {
        if f.path == "" {
            continue // File doesn't exist
        }

        if err := f.updater(f.path, version); err != nil {
            return fmt.Errorf("failed to update %s: %w", f.path, err)
        }
    }

    return nil
}
```

##### pyproject.toml Update

```go
func updatePyprojectToml(path, version string) error {
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    // Replace version in [project] section
    // version = "1.15.0" -> version = "1.16.0"
    re := regexp.MustCompile(`(?m)^version\s*=\s*"[^"]*"`)
    updated := re.ReplaceAllString(string(content), fmt.Sprintf(`version = "%s"`, version))

    return os.WriteFile(path, []byte(updated), 0644)
}
```

##### setup.py Update (Legacy)

```go
func updateSetupPy(path, version string) error {
    content, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // setup.py is optional (legacy)
        }
        return err
    }

    // Replace version = "1.15.0" -> version = "1.16.0"
    re := regexp.MustCompile(`version\s*=\s*"[^"]*"`)
    updated := re.ReplaceAllString(string(content), fmt.Sprintf(`version = "%s"`, version))

    return os.WriteFile(path, []byte(updated), 0644)
}
```

##### gapic_version.py Update

```go
func updateGapicVersion(path, version string) error {
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    // Replace __version__ = "1.15.0" -> __version__ = "1.16.0"
    re := regexp.MustCompile(`__version__\s*=\s*"[^"]*"`)
    updated := re.ReplaceAllString(string(content), fmt.Sprintf(`__version__ = "%s"`, version))

    return os.WriteFile(path, []byte(updated), 0644)
}
```

##### Finding Version Files

```go
func findGapicVersionFile(libPath string) string {
    // Search for google/cloud/*/gapic_version.py
    pattern := filepath.Join(libPath, "google", "cloud", "*", "gapic_version.py")
    matches, err := filepath.Glob(pattern)
    if err != nil || len(matches) == 0 {
        return ""
    }
    return matches[0]
}

func findVersionFile(libPath string) string {
    // Search for google/cloud/*/version.py
    pattern := filepath.Join(libPath, "google", "cloud", "*", "version.py")
    matches, err := filepath.Glob(pattern)
    if err != nil || len(matches) == 0 {
        return ""
    }
    return matches[0]
}
```

#### 3. Updating Changelogs

```go
func updateChangelogs(lib *config.Library, version string, changes []*Change) error {
    // 1. Update package CHANGELOG.md
    pkgChangelog := filepath.Join(lib.Path, "CHANGELOG.md")
    if err := updateChangelog(pkgChangelog, version, changes); err != nil {
        return fmt.Errorf("failed to update package changelog: %w", err)
    }

    // 2. Update docs/CHANGELOG.md (duplicate)
    docsChangelog := filepath.Join(lib.Path, "docs", "CHANGELOG.md")
    if err := updateChangelog(docsChangelog, version, changes); err != nil {
        return fmt.Errorf("failed to update docs changelog: %w", err)
    }

    // 3. Update global CHANGELOG.md (if exists)
    globalChangelog := "CHANGELOG.md"
    if fileExists(globalChangelog) {
        if err := updateGlobalChangelog(globalChangelog, lib.Name, version, changes); err != nil {
            return fmt.Errorf("failed to update global changelog: %w", err)
        }
    }

    return nil
}
```

##### Changelog Format (Package-Level)

```go
func updateChangelog(path, version string, changes []*Change) error {
    // Generate new changelog entry
    entry := formatChangelogEntry(version, changes)

    // Read existing changelog
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    // Insert new entry at top (after "# Changelog" header)
    lines := strings.Split(string(content), "\n")

    // Find insertion point (after first header)
    insertIdx := 1
    for i, line := range lines {
        if strings.HasPrefix(line, "#") {
            insertIdx = i + 1
            break
        }
    }

    // Insert new entry
    newLines := append(lines[:insertIdx], append([]string{"", entry, ""}, lines[insertIdx:]...)...)
    updated := strings.Join(newLines, "\n")

    return os.WriteFile(path, []byte(updated), 0644)
}

func formatChangelogEntry(version string, changes []*Change) string {
    today := time.Now().Format("2006-01-02")

    var buf strings.Builder
    buf.WriteString(fmt.Sprintf("## [%s](https://github.com/googleapis/google-cloud-python/releases/tag/google-cloud-secret-manager-v%s) (%s)\n\n",
        version, version, today))

    // Group changes by type
    features := filterByType(changes, "feat")
    fixes := filterByType(changes, "fix")
    docs := filterByType(changes, "docs")

    if len(features) > 0 {
        buf.WriteString("### Features\n\n")
        for _, c := range features {
            buf.WriteString(fmt.Sprintf("* %s ([%s](https://github.com/googleapis/google-cloud-python/commit/%s))\n",
                c.Subject, c.CommitHash[:7], c.CommitHash))
        }
        buf.WriteString("\n")
    }

    if len(fixes) > 0 {
        buf.WriteString("### Bug Fixes\n\n")
        for _, c := range fixes {
            buf.WriteString(fmt.Sprintf("* %s ([%s](https://github.com/googleapis/google-cloud-python/commit/%s))\n",
                c.Subject, c.CommitHash[:7], c.CommitHash))
        }
        buf.WriteString("\n")
    }

    if len(docs) > 0 {
        buf.WriteString("### Documentation\n\n")
        for _, c := range docs {
            buf.WriteString(fmt.Sprintf("* %s ([%s](https://github.com/googleapis/google-cloud-python/commit/%s))\n",
                c.Subject, c.CommitHash[:7], c.CommitHash))
        }
        buf.WriteString("\n")
    }

    return buf.String()
}
```

##### Global Changelog Update

```go
func updateGlobalChangelog(path, libName, version string, changes []*Change) error {
    // Global changelog has entries for all libraries
    // Format:
    // # Changelog
    //
    // ## google-cloud-secret-manager
    //
    // ### [1.16.0] (2025-11-12)
    // ...
    //
    // ## google-cloud-pubsub
    // ...

    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    entry := formatGlobalChangelogEntry(libName, version, changes)

    // Find section for this library, or create it
    // Insert new version entry
    lines := strings.Split(string(content), "\n")
    insertIdx := findLibrarySection(lines, libName)

    // Insert entry
    newLines := append(lines[:insertIdx], append([]string{entry}, lines[insertIdx:]...)...)
    updated := strings.Join(newLines, "\n")

    return os.WriteFile(path, []byte(updated), 0644)
}
```

#### 4. Snippet Metadata Update

```go
func updateSnippetMetadata(lib *config.Library, version string) error {
    // Find snippet metadata files
    pattern := filepath.Join("internal", "generated", "snippets", lib.Name, "**", "snippet_metadata.*.json")
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return err
    }

    for _, path := range matches {
        if err := updateSnippetMetadataFile(path, version); err != nil {
            return fmt.Errorf("failed to update %s: %w", path, err)
        }
    }

    return nil
}

func updateSnippetMetadataFile(path, version string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    var metadata map[string]interface{}
    if err := json.Unmarshal(data, &metadata); err != nil {
        return err
    }

    // Update version field
    metadata["clientVersion"] = version

    updated, err := json.MarshalIndent(metadata, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, updated, 0644)
}
```

### Version Format Normalization

Python uses PEP 440 version format, which differs from semver for pre-releases.

#### Normalization Function

```go
func normalizePythonVersion(version string) string {
    // Normalize semver pre-release to PEP 440
    // 1.16.0-rc.1 -> 1.16.0rc1
    // 1.16.0-alpha.1 -> 1.16.0a1
    // 1.16.0-beta.1 -> 1.16.0b1

    version = strings.ReplaceAll(version, "-rc.", "rc")
    version = strings.ReplaceAll(version, "-alpha.", "a")
    version = strings.ReplaceAll(version, "-beta.", "b")

    return version
}
```

#### Usage

```go
func Release(ctx context.Context, lib *config.Library, version string, changes []*Change) error {
    // Normalize version for Python
    pythonVersion := normalizePythonVersion(version)

    // Use pythonVersion for all updates
    if err := updateVersionFiles(lib.Path, pythonVersion); err != nil {
        return err
    }

    // ...
}
```

### Complete Example

#### Release Command

```bash
librarian release google-cloud-secret-manager --execute
```

**What happens:**

1. **Git Analysis**
   ```
   ✓ Found last tag: google-cloud-secret-manager/v2.19.0
   ✓ Analyzed 5 commits since last release
   ✓ Determined version bump: 2.19.0 → 2.20.0 (minor)
   ```

2. **Python Tests**
   ```
   ✓ Running nox -s unit...
     (test output)
   ✓ Tests passed
   ```

3. **File Updates**
   ```
   ✓ Updated packages/google-cloud-secret-manager/pyproject.toml
   ✓ Updated packages/google-cloud-secret-manager/google/cloud/secretmanager/gapic_version.py
   ✓ Updated packages/google-cloud-secret-manager/CHANGELOG.md
   ✓ Updated packages/google-cloud-secret-manager/docs/CHANGELOG.md
   ✓ Updated CHANGELOG.md (global)
   ✓ Updated snippet metadata
   ```

4. **Git Operations**
   ```
   ✓ Created commit: chore(release): google-cloud-secret-manager v2.20.0
   ✓ Created tag: google-cloud-secret-manager/v2.20.0
   ✓ Pushed tag to origin
   ```

**Note:** Publishing to PyPI is handled separately via CI/CD automation.
The `librarian release` command only prepares the release files and creates the git tag.

### Error Handling

#### Test Failures

```
Error: Tests failed for google-cloud-secret-manager

--- FAIL: test_get_secret (0.00s)
    test_client.py:42: AssertionError: expected None, got error

Fix tests or use --skip-tests (not recommended).
```

#### Version File Not Found

```
Warning: gapic_version.py not found in packages/google-cloud-secret-manager/
Skipping gapic_version.py update.

This is normal for handwritten libraries.
```

### Testing the Implementation

#### Unit Tests

```go
// internal/release/python_test.go

func TestNormalizePythonVersion(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"1.16.0", "1.16.0"},
        {"1.16.0-rc.1", "1.16.0rc1"},
        {"1.16.0-alpha.1", "1.16.0a1"},
        {"1.16.0-beta.2", "1.16.0b2"},
    }

    for _, test := range tests {
        got := normalizePythonVersion(test.input)
        if got != test.want {
            t.Errorf("normalizePythonVersion(%q) = %q, want %q", test.input, got, test.want)
        }
    }
}

func TestUpdatePyprojectToml(t *testing.T) {
    content := `[project]
name = "google-cloud-secret-manager"
version = "2.19.0"
`

    want := `[project]
name = "google-cloud-secret-manager"
version = "2.20.0"
`

    tmpfile := createTempFile(t, content)
    defer os.Remove(tmpfile)

    if err := updatePyprojectToml(tmpfile, "2.20.0"); err != nil {
        t.Fatal(err)
    }

    got, err := os.ReadFile(tmpfile)
    if err != nil {
        t.Fatal(err)
    }

    if string(got) != want {
        t.Errorf("mismatch (-want +got):\n%s", diff(want, string(got)))
    }
}
```

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
