# Design: Comprehensive librarian.yaml Schema

## Overview

This design extends the `librarian.yaml` format to capture **all** data from the old `.librarian` format, ensuring zero data loss during conversion.

## Three-File Consolidation

The old format uses three separate files:

1. **`.librarian/config.yaml`** → Global settings and release blocks
2. **`.librarian/state.yaml`** → Library state and generation config
3. **`.librarian/generator-input/repo-config.yaml`** → Per-module overrides

All three are consolidated into a single `librarian.yaml` file.

## Schema Structure

```yaml
version: v1          # Schema version
language: go         # Primary language

container:           # Container image config
  image: ...
  tag: ...

sources:             # External source repositories
  googleapis:
    url: ...
    sha256: ...

global:              # Global repository settings
  files_allowlist:   # Files modifiable during configure
    - path: ...
      permissions: ...

defaults:            # Default settings
  generate:
    output: ...

release:             # Global release config
  tag_format: ...

libraries:           # Library definitions
  - name: ...
    version: ...
    module_path_version: ...    # Optional: for v2+ modules
    last_generated_commit: ...  # Optional: state tracking
    source_roots: [...]         # Source directories

    release:                    # Per-library release config
      disabled: true/false      # Prevent auto-release

    generate:                   # Generation config
      apis:                     # API definitions
        - path: ...
          client_directory: ... # Optional override
          disable_gapic: ...    # Optional override
          proto_package: ...    # Optional override
          nested_protos: [...]  # Optional nested protos

      keep: [...]                        # Preserve patterns (regex)
      delete_output_paths: [...]         # Delete after generation
```

## New Fields Added

### Top-Level

| Field | Source | Purpose |
|-------|--------|---------|
| `global.files_allowlist` | config.yaml | Files modifiable during configure |
| `sources.googleapis` | (new) | Source repository tracking |

### Per-Library

| Field | Source | Purpose |
|-------|--------|---------|
| `module_path_version` | repo-config.yaml | Module version suffix (e.g., v2) |
| `source_roots` | state.yaml | Source directories |
| `release.disabled` | config.yaml | Prevent auto-release |

### Per-Library Generate

| Field | Source | Purpose |
|-------|--------|---------|
| `generate.delete_output_paths` | repo-config.yaml | Delete after generation |

### Per-API

| Field | Source | Purpose |
|-------|--------|---------|
| `apis[].client_directory` | repo-config.yaml | Custom client location |
| `apis[].disable_gapic` | repo-config.yaml | Disable GAPIC generation |
| `apis[].proto_package` | repo-config.yaml | Custom proto package |
| `apis[].nested_protos` | repo-config.yaml | Nested proto files |

## Field Organization Rationale

### Why `source_roots` at library level?
- These define where the library's source code lives
- Used by both generation and release processes
- Not specific to generation, so doesn't belong in `generate` section

### Why `release.disabled` and `release.exclude_paths`?
- Both are release-time concerns
- Grouped together for clarity
- Separate from generation config

### Why API-level overrides under each API?
- Each API can have different generation requirements
- Keeps related config together
- More maintainable than separate override section

### Why both `keep` and `remove`?
- `keep` = preserve during generation (don't overwrite)
- `remove` = delete after generation (cleanup)
- Different purposes, both needed

## Comparison with Current Schema

### Current Schema (Minimal)

```yaml
version: v1
language: go
container:
  image: ...
  tag: ...
generate:
  output: ...
release:
  tag_format: ...
libraries:
  - name: ...
    version: ...
    generate:
      apis:
        - path: ...
      keep: [...]
```

**Captures**: ~15% of old format data

### Comprehensive Schema

```yaml
version: v1
language: go
container:
  image: ...
  tag: ...
sources:
  googleapis:
    url: ...
    sha256: ...
global:
  files_allowlist: [...]
defaults:
  generate:
    output: ...
release:
  tag_format: ...
libraries:
  - name: ...
    version: ...
    module_path_version: ...
    source_roots: [...]
    release:
      disabled: ...
    generate:
      apis:
        - path: ...
          client_directory: ...
          disable_gapic: ...
          proto_package: ...
          nested_protos: [...]
      keep: [...]
      remove: [...]
      delete_output_paths: [...]
```

**Captures**: 100% of old format data

## Implementation Changes Required

### 1. Update `internal/config/config.go`

Add new types:

```go
// Global repository settings
type Global struct {
    FilesAllowlist []FileAllowlist `yaml:"files_allowlist,omitempty"`
}

type FileAllowlist struct {
    Path        string `yaml:"path"`
    Permissions string `yaml:"permissions"`
}

// Defaults
type Defaults struct {
    Generate *DefaultsGenerate `yaml:"generate,omitempty"`
}

type DefaultsGenerate struct {
    Output string `yaml:"output,omitempty"`
}

// Extend Library
type Library struct {
    Name              string           `yaml:"name"`
    Version           string           `yaml:"version,omitempty"`
    ModulePathVersion string           `yaml:"module_path_version,omitempty"` // NEW
    SourceRoots       []string         `yaml:"source_roots,omitempty"`        // NEW
    Release           *LibraryRelease  `yaml:"release,omitempty"`             // NEW
    Generate          *LibraryGenerate `yaml:"generate,omitempty"`
}

// NEW: Per-library release config
type LibraryRelease struct {
    Disabled bool `yaml:"disabled,omitempty"`
}

// Extend LibraryGenerate
type LibraryGenerate struct {
    APIs              []API    `yaml:"apis,omitempty"`
    Keep              []string `yaml:"keep,omitempty"`
    DeleteOutputPaths []string `yaml:"delete_output_paths,omitempty"` // NEW
}

// Extend API
type API struct {
    Path            string   `yaml:"path"`
    ClientDirectory string   `yaml:"client_directory,omitempty"` // NEW
    DisableGapic    bool     `yaml:"disable_gapic,omitempty"`    // NEW
    ProtoPackage    string   `yaml:"proto_package,omitempty"`    // NEW
    NestedProtos    []string `yaml:"nested_protos,omitempty"`    // NEW
}

// Extend Config
type Config struct {
    Version   string     `yaml:"version"`
    Language  string     `yaml:"language,omitempty"`
    Container *Container `yaml:"container,omitempty"`
    Sources   Sources    `yaml:"sources,omitempty"`
    Global    *Global    `yaml:"global,omitempty"`    // NEW
    Defaults  *Defaults  `yaml:"defaults,omitempty"`  // NEW
    Release   *Release   `yaml:"release,omitempty"`
    Libraries []Library  `yaml:"libraries,omitempty"`
}
```

### 2. Update `internal/convert/convert.go`

Extend converter to read all three files:

```go
func Convert(inputDir, outputFile string) error {
    // Read config.yaml
    config := readConfigYAML(inputDir)

    // Read state.yaml
    state := readStateYAML(inputDir)

    // Read generator-input/repo-config.yaml
    repoConfig := readRepoConfigYAML(inputDir)

    // Merge all three into new Config
    newConfig := mergeAllSources(config, state, repoConfig)

    // Write output
    return newConfig.Write(outputFile)
}
```

### 3. Update Test Data

Create comprehensive test fixtures showing:
- Libraries with all field combinations
- Edge cases (empty lists, missing optional fields)
- Round-trip conversion testing

## Migration Strategy

### Phase 1: Schema Definition (Current)
- ✅ Document comprehensive schema
- ✅ Define Go types
- ✅ Create examples

### Phase 2: Implementation
- Update `config.go` with new types
- Update converter to read all three files
- Update converter to populate all fields
- Add comprehensive tests

### Phase 3: Validation
- Convert entire google-cloud-go repository
- Validate no data loss
- Verify all 183 libraries
- Check all edge cases

### Phase 4: Tooling
- Update librarian commands to use new fields
- Add validation for new fields
- Update documentation

## Benefits

1. **Zero Data Loss**: Every field preserved
2. **Single Source of Truth**: One file instead of three
3. **Better Organization**: Logical grouping of related fields
4. **Extensible**: Easy to add new fields
5. **Language Agnostic**: Works for Go, Python, Rust
6. **Backward Compatible**: Old minimal format still works (optional fields)

## Backward Compatibility

Libraries that don't need the new fields can use the minimal format:

```yaml
libraries:
  - name: simple
    version: 1.0.0
    generate:
      apis:
        - path: google/cloud/simple/v1
```

Libraries that need the new fields can specify them:

```yaml
libraries:
  - name: complex
    version: 2.0.0
    module_path_version: v2
    source_roots: [complex, internal/snippets/complex]
    release:
      disabled: true
    generate:
      apis:
        - path: google/cloud/complex/v1
          disable_gapic: true
      keep: [^complex/handwritten/.*$]
      delete_output_paths: [internal/snippets/complex/internal]
```

## Files Created

- ✅ `doc/design-librarian-yaml-comprehensive.md` - Complete schema definition
- ✅ `doc/librarian-yaml-example.yaml` - Real-world examples
- ✅ `doc/DESIGN_COMPREHENSIVE_SCHEMA.md` - This file

## Next Steps

1. Review and approve schema design
2. Implement new types in `internal/config/config.go`
3. Update converter in `internal/convert/convert.go`
4. Add comprehensive tests
5. Convert google-cloud-go and validate
