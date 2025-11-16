# Python Library Migration Tool

## Objective

Create a one-time migration tool that converts google-cloud-python's legacy `.librarian/` configuration into the new unified `librarian.yaml` format.

## Background

Currently, google-cloud-python uses multiple files to configure Librarian:
- `.librarian/state.yaml` - tracks library state (versions, APIs, source_roots, etc.)
- `.librarian/config.yaml` - repository-level configuration (global files, overrides)
- `.librarian/generator-input/` - additional language-specific generator inputs
- `BUILD.bazel` files - contain `py_gapic_library` sections with generation metadata
- `.generator/` - generator configuration that defines required fields

Users cannot easily understand the full configuration of their libraries because the information is scattered across multiple locations. They need a single, unified configuration file that is human-readable and maintainable.

This document proposes a migration tool to consolidate these files into the new `librarian.yaml` format.

## Overview

The migration tool reads all legacy configuration sources, merges and deduplicates the data, and transforms it into the new `librarian.yaml` schema. The tool outputs the result to stdout or a specified file.

## Detailed Design

### Command Structure

```
librarian-migrate [flags]

Flags:
  -repo string
        Path to the google-cloud-python repository (required)
  -output string
        Output file path (default: stdout)
  -googleapis string
        Path to googleapis repository for BUILD.bazel files
```

### Data Flow

```
┌─────────────────────────┐
│  .librarian/state.yaml  │
│  .librarian/config.yaml │
│  .librarian/generator-  │
│      input/             │
│  BUILD.bazel files      │
│  .generator/            │
└──────────┬──────────────┘
           │
           ▼
    ┌──────────────┐
    │   Parse &    │
    │   Merge      │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐
    │  Transform   │
    │  to new      │
    │  schema      │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐
    │ librarian.   │
    │ yaml         │
    └──────────────┘
```

### Package Structure

The tool lives in `cmd/librarian-migrate/` as throwaway code:

```
cmd/librarian-migrate/
├── main.go              # Entry point, CLI flag handling
├── reader.go            # Read all source files
├── merger.go            # Merge and deduplicate data
├── transformer.go       # Transform to new schema
└── types.go             # Data structures for migration
```

### Processing Steps

#### 1. Read Phase (reader.go)

Read all source configuration files:

- Read `.librarian/state.yaml` → extract libraries, versions, APIs, source_roots, preserve_regex, remove_regex, tag_format
- Read `.librarian/config.yaml` → extract global_files_allowlist, library overrides (next_version, generate_blocked, release_blocked)
- Read `.librarian/generator-input/*` → collect Python-specific configuration files
- Read `.generator/*` → understand which fields are actually used
- For each library in state.yaml:
  - Locate corresponding `BUILD.bazel` in googleapis
  - Extract `py_gapic_library` sections for transport, service_yaml, and other metadata

#### 2. Merge Phase (merger.go)

Combine data from all sources per library:

- state.yaml provides: id, version, apis, source_roots, preserve_regex, remove_regex, tag_format
- config.yaml provides: next_version, generate_blocked, release_blocked
- BUILD.bazel provides: transport settings, opt_args
- generator-input provides: additional Python-specific config

Deduplicate by:
- Extracting common patterns across all libraries to populate the `default:` section
- Removing redundant fields that match defaults
- Consolidating repeated opt_args into common patterns

#### 3. Transform Phase (transformer.go)

Map old schema to new schema:

- `state.libraries[].id` → `libraries[].name`
- `state.libraries[].apis[].path` → `libraries[].api` or `libraries[].apis`
- `state.libraries[].source_roots` → inferred from output template
- `state.libraries[].preserve_regex` → `libraries[].keep`
- `state.libraries[].tag_format` → `default.release.tag_format` or library-specific override
- BUILD.bazel transport → `libraries[].transport` or `default.generate.transport`
- BUILD.bazel opt_args → `libraries[].python.opt_args`
- generator-input configs → `libraries[].python` section

Generate output structure:
```yaml
version: v1
language: python
default:
  output: packages/{name}/
  generate:
    one_library_per: service
    transport: grpc+rest
    rest_numeric_enums: true
  release:
    tag_format: '{name}/v{version}'
libraries:
  - name: <library-name>
    api: <api-path>  # or apis: [...]
    keep: [...]
    transport: <override if different from default>
    python:
      opt_args: [...]
```

#### 4. Output Phase (main.go)

Format and output the result:
- Format as YAML using `gopkg.in/yaml.v3`
- Run `yamlfmt` if available
- Write to file or stdout

### Data Mapping Reference

| Old Location | Old Field | New Location | New Field |
|--------------|-----------|--------------|-----------|
| state.yaml | `libraries[].id` | librarian.yaml | `libraries[].name` |
| state.yaml | `libraries[].version` | librarian.yaml | (not migrated, state-specific) |
| state.yaml | `libraries[].apis[].path` | librarian.yaml | `libraries[].api` or `apis` |
| state.yaml | `libraries[].preserve_regex` | librarian.yaml | `libraries[].keep` |
| state.yaml | `libraries[].tag_format` | librarian.yaml | `default.release.tag_format` |
| config.yaml | `global_files_allowlist` | librarian.yaml | (not in new format) |
| BUILD.bazel | `py_gapic_library.transport` | librarian.yaml | `libraries[].transport` |
| BUILD.bazel | `py_gapic_library.opt_args` | librarian.yaml | `libraries[].python.opt_args` |
| generator-input | `*.yaml` | librarian.yaml | `libraries[].python.*` |

### Deduplication Strategy

1. **Transport**: If 80%+ libraries use same transport, make it default
2. **Tag Format**: If all libraries use same format, move to default
3. **Opt Args**: Group common opt_args patterns
4. **Output Path**: Extract common pattern to `default.output` template

### Error Handling

- Missing required files → clear error with path
- Invalid YAML → parse error with line number
- Missing BUILD.bazel → warning, continue without that data
- Conflicting data → warning, prefer state.yaml

### Testing Strategy

Since this is throwaway code, testing is manual:
1. Run against google-cloud-python repository
2. Validate output YAML syntax
3. Spot-check 3-5 libraries for correctness
4. Compare generated librarian.yaml with existing testdata/python/librarian.yaml

## Alternatives Considered

### Make it part of internal/

We considered placing the code in `internal/` to follow standard project structure. This was rejected because the migration tool is throwaway code only needed once. Keeping it in `cmd/` makes it clear this is a temporary utility.

### Separate commands per source

We considered creating separate commands to read each configuration source (one for state.yaml, one for config.yaml, etc.). This was rejected because the merge logic needs all sources together to properly deduplicate and identify common patterns.

### Manual migration

We considered manually migrating the configuration files. This was rejected because it is too error-prone for 50+ libraries and does not allow for systematic deduplication of common patterns.
