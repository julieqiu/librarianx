# Comprehensive librarian.yaml Design

This document describes a complete `librarian.yaml` schema that captures all data from the old `.librarian` format without dropping anything.

## Schema Overview

```yaml
version: v1
language: go

# Container configuration
container:
  image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go
  tag: latest

# External sources
sources:
  googleapis:
    url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
    sha256: abc123...

# Global repository settings
global:
  files_allowlist:
    - path: "internal/generated/snippets/go.mod"
      permissions: read-write
    - path: "CHANGES.md"
      permissions: read-write

# Default settings
defaults:
  generate:
    output: '{name}/'

# Release configuration
release:
  tag_format: '{name}/v{version}'

# Libraries
libraries:
  - name: secretmanager
    version: 1.15.0

    # Module configuration (optional - only for v2+ modules)
    module_path_version: v2

    # Source directories
    source_roots:
      - secretmanager
      - internal/generated/snippets/secretmanager

    # Release configuration (optional)
    release:
      disabled: true  # Prevents automatic releases (for handwritten libs)

    # Generation configuration
    generate:
      # API definitions
      apis:
        - path: google/cloud/secretmanager/v1
          # Optional per-API overrides:
          client_directory: apiv1           # Custom client location
          disable_gapic: false              # Disable GAPIC generation
          proto_package: google.cloud.sm.v1 # Custom proto package
          nested_protos:                    # Additional nested protos
            - nested/extra.proto

        - path: google/cloud/secretmanager/v1beta2

      # Files to preserve during generation (regex patterns)
      keep:
        - ^secretmanager/CHANGES.md$
        - ^secretmanager/apiv1/iam\.go$

      # Paths to delete from output (post-generation cleanup)
      delete_output_paths:
        - internal/generated/snippets/secretmanager/internal
```

## Complete Type Definitions

```go
// Config represents the complete librarian.yaml configuration file.
type Config struct {
    Version   string     `yaml:"version"`
    Language  string     `yaml:"language,omitempty"`
    Container *Container `yaml:"container,omitempty"`
    Sources   Sources    `yaml:"sources,omitempty"`
    Global    *Global    `yaml:"global,omitempty"`
    Defaults  *Defaults  `yaml:"defaults,omitempty"`
    Release   *Release   `yaml:"release,omitempty"`
    Libraries []Library  `yaml:"libraries,omitempty"`
}

// Container contains the container image configuration.
type Container struct {
    Image string `yaml:"image"`
    Tag   string `yaml:"tag"`
}

// Sources contains references to external source repositories.
type Sources struct {
    Googleapis *Source `yaml:"googleapis,omitempty"`
}

// Source represents an external source repository.
type Source struct {
    URL    string `yaml:"url"`
    SHA256 string `yaml:"sha256"`
}

// Global contains global repository settings.
type Global struct {
    FilesAllowlist []FileAllowlist `yaml:"files_allowlist,omitempty"`
}

// FileAllowlist represents a file that can be modified globally.
type FileAllowlist struct {
    Path        string `yaml:"path"`
    Permissions string `yaml:"permissions"`
}

// Defaults contains default settings.
type Defaults struct {
    Generate *DefaultsGenerate `yaml:"generate,omitempty"`
}

// DefaultsGenerate contains default generation settings.
type DefaultsGenerate struct {
    Output string `yaml:"output,omitempty"`
}

// Release contains release configuration.
type Release struct {
    TagFormat string `yaml:"tag_format,omitempty"`
}

// Library represents a library.
type Library struct {
    Name                 string           `yaml:"name"`
    Version              string           `yaml:"version,omitempty"`
    ModulePathVersion    string           `yaml:"module_path_version,omitempty"`
    LastGeneratedCommit  string           `yaml:"last_generated_commit,omitempty"`
    SourceRoots          []string         `yaml:"source_roots,omitempty"`
    Release              *LibraryRelease  `yaml:"release,omitempty"`
    Generate             *LibraryGenerate `yaml:"generate,omitempty"`
}

// LibraryRelease contains per-library release configuration.
type LibraryRelease struct {
    Disabled bool `yaml:"disabled,omitempty"`
}

// LibraryGenerate contains generation configuration for a library.
type LibraryGenerate struct {
    APIs              []API    `yaml:"apis,omitempty"`
    Keep              []string `yaml:"keep,omitempty"`
    Remove            []string `yaml:"remove,omitempty"`
    DeleteOutputPaths []string `yaml:"delete_output_paths,omitempty"`
}

// API represents an API configuration.
type API struct {
    Path            string   `yaml:"path"`
    ServiceConfig   string   `yaml:"service_config,omitempty"`
    ClientDirectory string   `yaml:"client_directory,omitempty"`
    DisableGapic    bool     `yaml:"disable_gapic,omitempty"`
    ProtoPackage    string   `yaml:"proto_package,omitempty"`
    NestedProtos    []string `yaml:"nested_protos,omitempty"`
}
```

## Field Mapping from Old Format

### From `.librarian/config.yaml`

```yaml
# Old format
global_files_allowlist:
  - path: "go.mod"
    permissions: "read-write"

libraries:
  - id: "auth"
    release_disabled: true
```

```yaml
# New format
global:
  files_allowlist:
    - path: "go.mod"
      permissions: read-write

libraries:
  - name: auth
    release:
      disabled: true
```

### From `.librarian/state.yaml`

```yaml
# Old format
image: registry/image:tag
libraries:
  - id: secretmanager
    version: 1.15.0
    apis:
      - path: google/cloud/secretmanager/v1
    source_roots:
      - secretmanager
      - internal/generated/snippets/secretmanager
    preserve_regex:
      - ^secretmanager/CHANGES.md$
    release_exclude_paths:
      - internal/generated/snippets/secretmanager/
    tag_format: '{name}/v{version}'
```

```yaml
# New format
container:
  image: registry/image
  tag: tag

release:
  tag_format: '{name}/v{version}'

libraries:
  - name: secretmanager
    version: 1.15.0
    source_roots:
      - secretmanager
      - internal/generated/snippets/secretmanager
    generate:
      apis:
        - path: google/cloud/secretmanager/v1
      keep:
        - ^secretmanager/CHANGES.md$
```

### From `.librarian/generator-input/repo-config.yaml`

```yaml
# Old format
modules:
  - name: dataproc
    module_path_version: v2
    apis:
      - path: google/cloud/dataproc/v1
        client_directory: apiv1
        disable_gapic: false
        proto_package: google.cloud.dataproc.v1
        nested_protos:
          - nested/proto.proto
    delete_generation_output_paths:
      - internal/snippets/internal
```

```yaml
# New format
libraries:
  - name: dataproc
    module_path_version: v2
    generate:
      apis:
        - path: google/cloud/dataproc/v1
          client_directory: apiv1
          disable_gapic: false
          proto_package: google.cloud.dataproc.v1
          nested_protos:
            - nested/proto.proto
      delete_output_paths:
        - internal/snippets/internal
```

## Example: Complete Library Entry

Here's what a fully-populated library entry looks like:

```yaml
libraries:
  - name: bigtable
    version: 1.40.1
    module_path_version: v2

    source_roots:
      - bigtable
      - internal/generated/snippets/bigtable

    release:
      disabled: true  # Handwritten library

    generate:
      apis:
        - path: google/bigtable/v2
          disable_gapic: true  # Handwritten client

        - path: google/bigtable/admin/v2
          disable_gapic: true  # Handwritten admin client

      keep:
        - ^bigtable/bttest/.*$
        - ^bigtable/internal/.*$

      delete_output_paths:
        - internal/generated/snippets/bigtable/internal
```

## Benefits of This Design

1. **No Data Loss**: Every field from the old format has a place in the new format
2. **Organized**: Related fields are grouped together (release, generate, etc.)
3. **Optional Fields**: Libraries only specify what they need - most fields are optional
4. **Extensible**: Easy to add new fields without breaking existing configs
5. **Language-Agnostic**: Structure works for Go, Python, Rust, etc.
6. **Self-Documenting**: Field names are clear and hierarchical

## Migration Path

The old format uses three separate files:
- `.librarian/config.yaml` → Merged into `global` and per-library `release.blocked`
- `.librarian/state.yaml` → Majority of library configuration
- `.librarian/generator-input/repo-config.yaml` → Per-library and per-API overrides

All three are consolidated into a single `librarian.yaml` file.
