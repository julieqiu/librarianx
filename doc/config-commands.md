# Librarian Config Commands Design

## Objective

Define a consistent CLI interface for managing librarian.yaml configuration using `librarian config` subcommands.

## Background

Currently, users must manually edit librarian.yaml to update configuration. This requires understanding YAML syntax, the config schema, and proper nesting. Common operations like updating metadata or managing file patterns are error-prone and time-consuming.

This document proposes a `librarian config` command with subcommands to manage configuration programmatically, inspired by successful patterns from npm, git, and gcloud.

**Note:** APIs are managed through `librarian add`, `librarian create`, and `librarian remove` commands, not through `librarian config`. The config command focuses on scalar values and file pattern arrays (keep/remove).

## Research: Config Command Patterns

### Go (`go env`)

**Commands:**
- `go env` - List all environment variables
- `go env GOPATH` - Get specific variable
- `go env -w GOPATH=/path` - Set (write) variable
- `go env -u GOPATH` - Unset (delete) variable

**Characteristics:**
- Simple flag-based interface (`-w` for write, `-u` for unset)
- No separate subcommands
- Stores in `os.UserConfigDir()/go/env` file
- Persistent across invocations

### npm (`npm config`)

**Commands:**
- `npm config list` - Show all config settings
- `npm config get <key>` - Get value(s)
- `npm config set <key>=<value>` - Set value
- `npm config delete <key>` - Delete key
- `npm config edit` - Open in editor

**Characteristics:**
- Explicit subcommands (list, get, set, delete, edit)
- Supports multiple keys in one invocation
- Clear separation between operations
- Most widely used pattern

### Git (`git config`)

**Commands:**
- `git config --list` - List all variables
- `git config <key>` - Get value
- `git config <key> <value>` - Set value
- `git config --unset <key>` - Remove key
- `git config --edit` - Open in editor

**Characteristics:**
- Hybrid: flags + positional arguments
- Single command with different behaviors based on arguments
- Supports `--global`, `--local`, `--system` scopes
- Advanced: `--rename-section`, `--remove-section`

### kubectl (`kubectl config`)

**Commands:**
- `kubectl config view` - Display merged config
- `kubectl config get-contexts` - List contexts
- `kubectl config set-context <name> --cluster=...` - Set context
- `kubectl config use-context <name>` - Switch context
- `kubectl config delete-context <name>` - Delete context
- `kubectl config current-context` - Show current context

**Characteristics:**
- Explicit subcommands
- Domain-specific operations (contexts, clusters, users)
- Composite operations (set multiple fields at once)
- Hierarchical config structure

### Cargo (`cargo config`) [Unstable]

**Commands (proposed):**
- `cargo config get <key>` - Get value
- `cargo config set <key> <value>` - Set value  [planned]
- `cargo config delete <key>` - Delete value  [planned]

**Characteristics:**
- Following npm/git patterns
- Comment-preserving TOML editor needed
- Still in development

## Design Decision: npm-style Subcommands with Array Support

**Recommended approach:** Explicit subcommands (like npm, gcloud, gh) with special handling for arrays

**Rationale:**
1. **Industry standard** - Used by npm, gcloud, gh, pnpm, poetry, aws cli
2. **Discoverable** - Each operation has clear verb
3. **Familiar** - Developers already know this pattern
4. **Clear intent** - `set`, `add`, `remove`, `delete` have obvious meanings
5. **Array support** - `add`/`remove` for arrays, `set`/`delete` for scalars
6. **Git-inspired arrays** - Follows git's `--add` pattern for multi-valued config

## Proposed Command Structure

### Core Subcommands

#### `librarian config set <key> <value>`

Set a scalar configuration value. Replaces existing value.

```bash
# Repository-level config
librarian config set language python
librarian config set generate.container.tag v1.0.0
librarian config set sources.googleapis.url https://github.com/googleapis/googleapis/archive/xyz789.tar.gz

# Edition-level config (requires --edition flag)
librarian config set --edition secretmanager version 0.2.0
librarian config set --edition secretmanager generate.metadata.release_level stable
librarian config set --edition secretmanager generate.metadata.name_pretty "Secret Manager"
```

**Use for:** Scalar values (strings, numbers, booleans)
**Not for:** Arrays (use `add`/`remove` instead)

#### `librarian config add <key> <value>`

Add a value to an array. Appends to existing array.

```bash
# Add to keep patterns array
librarian config add --edition secretmanager keep README.md
librarian config add --edition secretmanager keep docs/
librarian config add --edition secretmanager keep "*.md"

# Add to remove patterns array
librarian config add --edition secretmanager remove temp.txt
librarian config add --edition secretmanager remove "**/__pycache__"
```

**Use for:** Appending to arrays (keep, remove patterns)
**Behavior:** Creates array if it doesn't exist, appends if it does

#### `librarian config remove <key> <value>`

Remove a value from an array. Removes first matching value.

```bash
# Remove from keep patterns array
librarian config remove --edition secretmanager keep docs/

# Remove from remove patterns array
librarian config remove --edition secretmanager remove temp.txt
```

**Use for:** Removing specific items from arrays
**Behavior:** Removes the matching value, leaves array intact (even if empty)

#### `librarian config delete <key>`

Delete a configuration key entirely. Removes the key and all its values.

```bash
# Delete repository-level key
librarian config delete sources.discovery

# Delete edition-level key
librarian config delete --edition secretmanager generate.metadata.api_description

# Delete entire array
librarian config delete --edition secretmanager keep
```

**Use for:** Removing keys completely
**Behavior:** Key no longer exists in config

#### `librarian config get <key>`

Get value for a configuration key. Outputs to stdout.

```bash
# Get repository-level value
librarian config get language
# Output: go

# Get nested value
librarian config get sources.googleapis.url
# Output: https://github.com/googleapis/googleapis/archive/abc123.tar.gz

# Get edition-level value
librarian config get --edition secretmanager version
# Output: 0.2.0

# Get array values (one per line)
librarian config get --edition secretmanager keep
# Output:
# README.md
# docs/
# *.md
```

**Flags:**
- `--json` - Output as JSON

```bash
# Get as JSON
librarian config get --json sources.googleapis
# Output: {"url":"https://...","sha256":"81e6057..."}

# Get array as JSON
librarian config get --json --edition secretmanager keep
# Output: ["README.md", "docs/", "*.md"]
```

#### `librarian config list`

Display all configuration settings.

```bash
# List entire config (human-readable)
librarian config list

# List in JSON format
librarian config list --json

# List specific edition
librarian config list --edition secretmanager
```

**Output format (default):**
```
version: v0.5.0
language: go
sources.googleapis.url: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
sources.googleapis.sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
generate.container.image: us-central1-docker.pkg.dev/.../go-librarian-generator
generate.container.tag: latest

editions[0].name: secretmanager
editions[0].version: 0.2.0
editions[0].keep[0]: README.md
editions[0].keep[1]: docs/
...
```

#### `librarian config editions`

List all editions (convenience command).

```bash
# List edition names
librarian config editions

# Output:
# secretmanager
# pubsub
# storage
```

## Key Notation and Scoping

### Dot Notation for Nested Keys

Use **dot notation** to access nested configuration values:

```bash
# Top-level keys
language
version

# Nested keys
sources.googleapis.url
sources.googleapis.sha256
generate.container.image
generate.container.tag
generate.output_dir

# Edition-level nested keys (with --edition flag)
version
generate.metadata.name_pretty
generate.metadata.release_level
generate.metadata.product_documentation
```

### Configuration Scopes

Configuration has two scopes:

**1. Repository-level** (default, no flag needed)
- Top-level settings that apply to entire repository
- Examples: `version`, `language`, `sources`, `generate`, `release`

```bash
librarian config get language
librarian config set generate.container.tag latest
librarian config set sources.googleapis.url https://...
```

**2. Edition-level** (requires `--edition <name>` flag)
- Settings specific to a library/package
- Examples: `version`, `generate.metadata`, `keep`, `remove`

```bash
librarian config get --edition secretmanager version
librarian config set --edition secretmanager generate.metadata.release_level stable
librarian config add --edition secretmanager keep README.md
```

### Array vs Scalar Keys

Some keys hold **scalar values** (use `set`/`delete`):
- `language`
- `version`
- `generate.container.tag`
- `generate.metadata.release_level`

Some keys hold **arrays** (use `add`/`remove`/`delete`):
- `keep` - File patterns to preserve during generation
- `remove` - File patterns to delete after generation

**The command validates key types:**
```bash
# Error: keep is an array, use 'add' not 'set'
librarian config set --edition secretmanager keep README.md

# Correct:
librarian config add --edition secretmanager keep README.md
```

## Output Formats

Support multiple output formats via `--format` or `--json` flag:

```bash
# Default (human-readable)
librarian config list

# JSON
librarian config list --json

# YAML (for piping to tools)
librarian config get sources --format yaml
```

## Error Handling

**Clear, actionable error messages:**

```bash
$ librarian config get invalid.key
Error: Configuration key 'invalid.key' not found

Valid repository-level keys:
  - version
  - language
  - sources.googleapis.url
  - sources.googleapis.sha256
  ...

To list all keys: librarian config list
```

```bash
$ librarian config set language rust
Error: Unsupported language 'rust'

Supported languages: go, python, rust
```

```bash
$ librarian config set --edition nonexistent version 1.0.0
Error: Edition 'nonexistent' not found

Available editions:
  - secretmanager
  - pubsub

To add new edition: librarian config edition add <name>
```

## Implementation Considerations

### Config API

Build on existing `internal/config/config.go`:

```go
// Get retrieves a config value by dot-notation key
func (c *Config) Get(key string) (interface{}, error)

// Set updates a config value by dot-notation key
func (c *Config) Set(key string, value interface{}) error

// Delete removes a config key
func (c *Config) Delete(key string) error

// GetEdition retrieves edition-specific config
func (c *Config) GetEdition(name, key string) (interface{}, error)

// SetEdition updates edition-specific config
func (c *Config) SetEdition(name, key string, value interface{}) error

// AddEdition creates a new edition
func (c *Config) AddEdition(name string, apis []string) error

// DeleteEdition removes an edition
func (c *Config) DeleteEdition(name string) error

// Validate checks config for errors
func (c *Config) Validate() []error
```

### Dot Notation Parser

Implement key parser for dot notation:

```go
// ParseKey converts "sources.googleapis.url" to nested access
func ParseKey(key string) []string {
    return strings.Split(key, ".")
}

// GetNestedValue traverses config struct using parsed key
func GetNestedValue(cfg interface{}, parts []string) (interface{}, error)

// SetNestedValue updates config struct using parsed key
func SetNestedValue(cfg interface{}, parts []string, value interface{}) error
```

### YAML Preservation

**Challenge:** Preserving comments and formatting when updating YAML

**Solutions:**
1. Use comment-preserving YAML library (e.g., `gopkg.in/yaml.v3`)
2. For `edit` command, just open editor (no parsing needed)
3. For `set/delete`, use targeted updates instead of full rewrites

### Validation

Implement schema-based validation:

```go
type Schema struct {
    Fields map[string]FieldSpec
}

type FieldSpec struct {
    Type     string   // "string", "int", "bool", "array", "object"
    Required bool
    Enum     []string // For restricted values
    Pattern  string   // Regex for validation
}

func (s *Schema) Validate(config *Config) []error
```

## API Management: Why Not in Config?

APIs are intentionally **NOT** managed through `librarian config` commands. Instead:

```bash
# Create new edition with initial API(s) (syntactic sugar for add + generate)
librarian create secretmanager google/cloud/secretmanager/v1

# Later, add more APIs to existing edition (incremental)
librarian add secretmanager google/cloud/secretmanager/v1beta2
librarian generate secretmanager

# Add edition without APIs (release-only)
librarian add custom-tool

# Remove APIs
librarian remove secretmanager google/cloud/secretmanager/v1beta2
```

**Rationale:**
1. **Complex operation** - Adding an API requires parsing BUILD.bazel, extracting metadata, validation
2. **Code generation** - `create` generates code immediately; `add` updates config for later generation
3. **Domain-specific** - APIs are the primary entity, deserve dedicated commands
4. **Clear semantics** - `create` (initial edition creation) vs `add` (incremental additions) vs config `add` (array operations)

**Config focuses on:**
- Scalar values (metadata, versions, settings)
- File patterns (keep/remove arrays)
- Repository settings (sources, container images)

## Alternatives Considered

### Alternative 1: go mod edit Style (Rejected)

```bash
librarian config -set language=go -edition secretmanager -add-keep README.md
```

**Rejected because:**
- Confusing flag ordering and context switching
- Not the industry standard (only go mod edit uses this)
- Designed for tools, not humans
- Less discoverable than explicit subcommands

### Alternative 2: Git-style Hybrid (Rejected)

```bash
librarian config language        # Get
librarian config language go     # Set
librarian config --unset language # Delete
```

**Rejected because:**
- Ambiguous (is 2nd arg a value or subcommand?)
- Harder to parse and validate
- Less clear intent

### Alternative 3: Using `set` for Arrays (Rejected)

```bash
librarian config set --add --edition secretmanager keep README.md
librarian config set --remove --edition secretmanager keep docs/
```

**Rejected because:**
- `set --add` is verbose and confusing
- Flags modifying subcommand behavior is unclear
- Less intuitive than dedicated `add`/`remove` subcommands

### Alternative 4: Array Notation in Keys (Rejected)

```bash
librarian config set --edition secretmanager 'keep[]' README.md
librarian config delete --edition secretmanager 'keep[README.md]'
```

**Rejected because:**
- Weird syntax requiring quotes
- Not used by other tools
- Less clear than `add`/`remove` verbs

### Alternative 5: Opening Editor (Rejected)

```bash
librarian config edit  # Opens $EDITOR
```

**Rejected because:**
- Not scriptable
- Requires interactive session
- Hard to automate
- Manual YAML editing error-prone
- Counter to goal of command-line editing

### Alternative 6: Environment Variables (Rejected)

```bash
LIBRARIAN_LANGUAGE=go librarian generate
```

**Rejected because:**
- Not persistent
- Hard to discover available options
- Doesn't work well for complex nested config
- Use case: per-command overrides (could add later)

## Examples

### Complete Workflow: Managing File Patterns

```bash
# 1. View current keep patterns
librarian config get --edition secretmanager keep
# (empty or shows existing patterns)

# 2. Add patterns to preserve during regeneration
librarian config add --edition secretmanager keep README.md
librarian config add --edition secretmanager keep docs/
librarian config add --edition secretmanager keep "custom_*.go"

# 3. Verify
librarian config get --edition secretmanager keep
# README.md
# docs/
# custom_*.go

# 4. Remove a pattern
librarian config remove --edition secretmanager keep docs/

# 5. Regenerate code (patterns are applied)
librarian generate secretmanager
```

### Complete Workflow: Updating Repository Config

```bash
# 1. Check current settings
librarian config get language
librarian config get generate.container.tag

# 2. Update settings
librarian config set language python
librarian config set generate.container.tag v2.0.0

# 3. Update googleapis source
librarian config set sources.googleapis.url https://github.com/googleapis/googleapis/archive/xyz789.tar.gz
librarian config set sources.googleapis.sha256 867048ec8f0850a4d77ad836319e4c0a0c624928611af8a900cd77e676164e8e

# 4. Verify changes
librarian config list
```

### Complete Workflow: Updating Edition Metadata

```bash
# 1. Check current metadata
librarian config get --edition secretmanager generate.metadata.release_level

# 2. Update metadata fields
librarian config set --edition secretmanager generate.metadata.release_level stable
librarian config set --edition secretmanager generate.metadata.name_pretty "Secret Manager"
librarian config set --edition secretmanager generate.metadata.product_documentation https://cloud.google.com/secret-manager/docs

# 3. View all metadata as JSON
librarian config get --json --edition secretmanager generate.metadata
```

### Complete Workflow: Creating New Edition with Config

```bash
# 1. Use 'librarian create' to create edition with initial APIs
librarian create secretmanager google/cloud/secretmanager/v1

# 2. Configure file patterns
librarian config add --edition secretmanager keep README.md
librarian config add --edition secretmanager keep docs/

# 3. Update metadata
librarian config set --edition secretmanager generate.metadata.release_level stable

# 4. Later, add more APIs
librarian add secretmanager google/cloud/secretmanager/v1beta2
librarian generate secretmanager
```

### Scripting Example: Batch Updates

```bash
# Update release level for multiple editions
for edition in secretmanager pubsub storage; do
  librarian config set --edition $edition generate.metadata.release_level stable
done

# Add standard keep patterns to all editions
for edition in $(librarian config editions); do
  librarian config add --edition $edition keep README.md
  librarian config add --edition $edition keep CHANGES.md
done

# Conditional updates
if librarian config get --edition secretmanager version | grep -q "null"; then
  librarian config set --edition secretmanager version 0.1.0
fi
```

## Summary

The `librarian config` command provides a clear, scriptable interface for managing configuration, following industry-standard patterns from npm, gcloud, and git:

**Subcommands:**

**Scalar operations:**
- `set <key> <value>` - Set/update scalar value
- `get <key>` - Retrieve value
- `delete <key>` - Remove key entirely
- `list` - Show all configuration

**Array operations:**
- `add <key> <value>` - Append to array
- `remove <key> <value>` - Remove from array

**Query operations:**
- `editions` - List all editions (convenience)

**Flags:**
- `--edition <name>` - Scope operations to specific edition
- `--json` - Output as JSON (for `get`/`list`)

**Key Design Principles:**

1. **Clear verbs** - Each subcommand has obvious meaning
   - `set` = replace scalar
   - `add` = append to array
   - `remove` = remove from array
   - `delete` = delete key

2. **Industry standard** - Follows patterns from npm, gcloud, gh, git
   - Familiar to developers
   - Discoverable through `--help`
   - Matches existing mental models

3. **Type-aware** - Commands validate key types
   - Can't `set` an array (use `add`)
   - Can't `add` to scalar (use `set`)
   - Clear error messages

4. **Separation of concerns** - Config handles settings, not APIs
   - `librarian create` - Create new editions with initial APIs
   - `librarian add` - Add more APIs to existing editions (incremental)
   - `librarian remove` - Remove editions and APIs
   - `librarian config` - Manage settings and file patterns

5. **Scriptable** - Perfect for automation
   - Consistent exit codes
   - Machine-readable JSON output
   - Composable in shell scripts

**What this handles:**
- ✅ Repository settings (language, sources, container images)
- ✅ Edition metadata (release_level, documentation URLs)
- ✅ Version management
- ✅ File patterns (keep/remove arrays)

**What this does NOT handle:**
- ❌ API management (use `librarian add`/`librarian create`/`librarian remove`)
- ❌ Code generation (use `librarian generate`)
- ❌ Releases (use `librarian release`)
