# Data Dropped During Conversion

This document lists all data from the old `.librarian` format that is **not** converted to the new `librarian.yaml` format.

## Summary

The current converter only captures a **minimal subset** of the old format. Significant data is being dropped from three sources:

1. `.librarian/config.yaml`
2. `.librarian/state.yaml`
3. `.librarian/generator-input/repo-config.yaml`

---

## From `.librarian/config.yaml`

### ❌ `global_files_allowlist`
**Purpose**: Specifies which files can be modified during the configure step.

**Example**:
```yaml
global_files_allowlist:
  - path: "internal/generated/snippets/go.mod"
    permissions: "read-write"
  - path: "CHANGES.md"
    permissions: "read-write"
```

**Status**: Not captured in new format.

### ❌ `libraries[].release_blocked`
**Purpose**: Marks libraries (typically handwritten or hybrid) that should not be automatically released.

**Example**:
```yaml
libraries:
  - id: "auth"
    release_blocked: true
  - id: "bigtable"
    release_blocked: true
```

**Count**: 18 libraries with `release_blocked: true`

**Status**: Maps to `release.disabled` in new format.

---

## From `.librarian/state.yaml`

### ✅ `apis[].service_config`
**Purpose**: Specifies the service config file for each API.

**Example**:
```yaml
apis:
  - path: google/cloud/accessapproval/v1
    service_config: accessapproval_v1.yaml
```

**Status**: Not needed in new format - already captured in `service_config_overrides.yaml` (see [alternatives-considered.md](../../doc/alternatives-considered.md#service-config-in-api-configuration)).

### ✅ `source_roots`
**Purpose**: Lists the source directories for each library.

**Example**:
```yaml
source_roots:
  - accessapproval
  - internal/generated/snippets/accessapproval
```

**Analysis**: 175 out of 183 libraries (95.6%) follow the exact same pattern: `[{name}, internal/generated/snippets/{name}]`. This would add ~350 lines of duplicated configuration.

**Status**: Not needed in new format - handled by tooling defaults with variable substitution. Only 8 libraries with non-standard patterns need explicit configuration (see [alternatives-considered.md](../../doc/alternatives-considered.md#source-roots-in-library-configuration)).

### ✅ `remove_regex`
**Purpose**: Specifies files/directories to remove after generation (for cleanup).

**Example**:
```yaml
remove_regex:
  - ^internal/generated/snippets/accessapproval/
  - ^accessapproval/apiv1/[^/]*_client\.go$
  - ^accessapproval/apiv1/[^/]*_client_example_go123_test\.go$
  - ^accessapproval/apiv1/[^/]*_client_example_test\.go$
  - ^accessapproval/apiv1/auxiliary\.go$
  - ^accessapproval/apiv1/auxiliary_go123\.go$
  - ^accessapproval/apiv1/doc\.go$
  - ^accessapproval/apiv1/gapic_metadata\.json$
  - ^accessapproval/apiv1/helpers\.go$
  - ^accessapproval/apiv1/accessapprovalpb/.*$
  - ^accessapproval/apiv1/\.repo-metadata\.json$
```

**Analysis**: These patterns are nearly identical for all 183 libraries - just with the library name substituted. This would add ~1,830 lines of duplicated regex to librarian.yaml.

**Status**: Not needed in new format - handled by generator defaults with variable substitution (see [alternatives-considered.md](../../doc/alternatives-considered.md#remove-patterns-in-library-configuration)).

### ✅ `release_exclude_paths`
**Purpose**: Specifies paths to exclude from releases.

**Example**:
```yaml
release_exclude_paths:
  - internal/generated/snippets/accessapproval/
```

**Analysis**: 175 out of 183 libraries have this field, and ALL follow the exact same pattern: `internal/generated/snippets/{library}/`

**Status**: Not needed in new format - handled by release tool defaults with variable substitution (see [alternatives-considered.md](../../doc/alternatives-considered.md#release-exclude-paths-in-library-configuration)).

### ✅ `last_generated_commit`
**Purpose**: Tracks the last commit hash when the library was generated.

**Example**:
```yaml
last_generated_commit: c288189b43c016dd3cf1ec73ce3cadee8b732f07
```

**Status**: Not needed in new format - this is runtime state, not configuration. Git history tracks when files were last modified (see [alternatives-considered.md](../../doc/alternatives-considered.md#last-generated-commit-in-library-configuration)).

---

## From `.librarian/generator-input/repo-config.yaml`

This file contains module-specific overrides for generation. **All of this data is being dropped.**

### ❌ `modules[].module_path_version`
**Purpose**: Specifies the major version suffix for the module path.

**Example**:
```yaml
modules:
  - name: dataproc
    module_path_version: v2
```

**Count**: 3 modules (dataproc, recaptchaenterprise, vision)

**Status**: Not captured in new format.

### ❌ `modules[].apis[].client_directory`
**Purpose**: Specifies custom client directory location (overrides default).

**Example**:
```yaml
modules:
  - name: monitoring
    apis:
      - path: google/monitoring/v3
        client_directory: apiv3/v2
```

**Count**: ~15 APIs with custom client directories

**Status**: Not captured in new format.

### ❌ `modules[].apis[].disable_gapic`
**Purpose**: Disables GAPIC generation for an API (for handwritten clients).

**Example**:
```yaml
modules:
  - name: bigtable
    apis:
      - path: google/bigtable/v2
        disable_gapic: true
```

**Count**: 3 APIs (bigtable/v2, bigtable/admin/v2, datastore/v1)

**Status**: Not captured in new format.

### ❌ `modules[].apis[].nested_protos`
**Purpose**: Includes nested proto files in generation.

**Example**:
```yaml
modules:
  - name: containeranalysis
    apis:
      - path: google/devtools/containeranalysis/v1beta1
        nested_protos:
          - grafeas/grafeas.proto
```

**Count**: 1 API (containeranalysis)

**Status**: Not captured in new format.

### ❌ `modules[].apis[].proto_package`
**Purpose**: Specifies custom protobuf package name (when it differs from path).

**Example**:
```yaml
modules:
  - name: maps
    apis:
      - path: google/maps/fleetengine/v1
        proto_package: maps.fleetengine.v1
```

**Count**: 3 APIs (maps fleetengine, translate)

**Status**: Not captured in new format.

### ❌ `modules[].delete_generation_output_paths`
**Purpose**: Specifies paths to delete after generation.

**Example**:
```yaml
modules:
  - name: storage
    delete_generation_output_paths:
      - internal/generated/snippets/storage/internal
```

**Count**: 1 module (storage)

**Status**: Not captured in new format.

---

## What IS Captured

For comparison, here's what the converter DOES capture:

### ✅ From state.yaml:
- `image` → `container.image` + `container.tag`
- `libraries[].id` → `libraries[].name`
- `libraries[].version` → `libraries[].version`
- `libraries[].source_roots` → `libraries[].source_roots` (only for 8 non-standard libraries)
- `libraries[].apis[].path` → `libraries[].generate.apis[].path`
- `libraries[].preserve_regex` → `libraries[].generate.keep`
- `libraries[].tag_format` → `release.tag_format` (global)

### ✅ Hardcoded:
- `version: v1`
- `language: go`
- `generate.output: '{name}/'`

---

## Impact Assessment

### Critical Data Loss
None - all essential data is either captured or handled by tooling defaults.

### Important Data Loss
These fields affect release and module configuration:
- `release_blocked` - Handwritten libraries may be auto-released incorrectly
- `modules[].module_path_version` - Module paths will be incorrect for v2+ modules

### Generator-Specific Data Loss
All data in `generator-input/repo-config.yaml` is lost:
- Custom client directories
- Disabled GAPIC generation flags
- Custom proto packages
- Nested proto includes
- Custom deletion paths

### Intentionally Excluded (Not Data Loss)
- `last_generated_commit` - Runtime state tracked by git (see alternatives-considered.md #12)
- `apis[].service_config` - Already in service_config_overrides.yaml (see alternatives-considered.md #11)
- `source_roots` - Handled by tooling defaults; ~350 lines of duplicated patterns for 175 libraries (see alternatives-considered.md #15)
- `remove_regex` - Handled by generator defaults; ~1,830 lines of identical patterns (see alternatives-considered.md #13)
- `release_exclude_paths` - Handled by release tool defaults; 175 identical patterns (see alternatives-considered.md #14)

---

## Recommendation

The current converter is a **proof of concept** only. To create a production-ready converter, you need to:

1. **Extend the new `librarian.yaml` schema** to support all the fields from the old format
2. **Update the converter** to capture all data from all three files
3. **Decide on mapping strategy** for fields that don't have direct equivalents

Alternatively, if the new format intentionally excludes some of these fields, document which workflows will break and how users should work around them.
