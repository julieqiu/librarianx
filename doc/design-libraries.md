# Design Resolution: Library paths and locations

## Resolution (2025-11-11)

**Final Design Implemented:**

### Library struct:
```go
type Library struct {
    Name     string   `yaml:"name"`               // Library name
    Apis     []string `yaml:"apis,omitempty"`     // List of googleapis paths
    Location string   `yaml:"location,omitempty"` // Optional explicit filesystem path
}
```

### Template expansion in `generate.output`:
- **`{name}`** - Library name (works with any number of APIs)
- **`{api.path}`** - API path (requires exactly 1 API, fails otherwise)

### Language-specific patterns:

**Go:**
```yaml
generate:
  output: '{name}/'
libraries:
  - name: secretmanager
    apis:
      - google/cloud/secretmanager/v1
      - google/cloud/secretmanager/v1beta2
# Generates to: secretmanager/
```

**Python:**
```yaml
generate:
  output: 'packages/{name}/'
libraries:
  - name: google-cloud-secretmanager
    apis:
      - google/cloud/secretmanager/v1
      - google/cloud/secretmanager/v1beta2
# Generates to: packages/google-cloud-secretmanager/
```

**Rust:**
```yaml
generate:
  output: 'src/generated/{api.path}/'
libraries:
  - name: google-cloud-secretmanager-v1
    apis:
      - google/cloud/secretmanager/v1
# Generates to: src/generated/google/cloud/secretmanager/v1/

  - name: google-cloud-secretmanager-v1beta2
    apis:
      - google/cloud/secretmanager/v1beta2
# Generates to: src/generated/google/cloud/secretmanager/v1beta2/
```

**Handwritten libraries:**
```yaml
libraries:
  - name: storage
    location: storage/
# No apis field = handwritten, uses explicit location
```

## Original Problem

Need to support two types of libraries:

1. **Generated libraries** (from googleapis)
   - Input: googleapis path (e.g., `google/cloud/secretmanager/v1`)
   - Output: filesystem location (language-dependent)
   - Example: `librarian add secretmanager google/cloud/secretmanager/v1`

2. **Handwritten/release-only libraries**
   - No googleapis path (no generation needed)
   - Code already exists at specific filesystem location
   - Example: `librarian add storage --location storage/`

## Language-Specific Conventions

**Go:**
- Generated: `<generate.output>/<name>/` (e.g., `generated/secretmanager/`)
- Handwritten: `<root>/<name>/` (e.g., `storage/`)
- Both at top-level, whether generated or handwritten

**Rust:**
- Generated code: `src/generated/cloud/secretmanager/v1/src/`
- Generated metadata: `src/generated/cloud/secretmanager/v1/`
- Handwritten: `src/`

**Python:**
- Generated: `packages/google-cloud-secretmanager/`
- Handwritten: `packages/storage/`

## Design Questions

1. **Field naming confusion:**
   - Currently `path` means "googleapis path"
   - But we also need "filesystem path" for where code lives
   - Options:
     - `googleapis` + `path` (path = filesystem location)
     - `source` + `path` (source = googleapis, path = filesystem)
     - `googleapis` + `location`

2. **Path inference:**
   - If `googleapis` is present → generated → infer filesystem path as `<generate.output>/<name>`
   - If explicit filesystem path is set → use that
   - Release command needs to know where code lives

3. **Command syntax:**
   ```bash
   # Generated libraries
   librarian add secretmanager google/cloud/secretmanager/v1

   # Handwritten libraries - how to specify filesystem path?
   librarian add storage --location storage/
   # OR
   librarian add storage storage/
   # OR
   librarian add storage  # infers path as <generate.output>/storage
   ```

4. **Example YAML structures:**

   **Option A:**
   ```yaml
   libraries:
     - name: secretmanager
       googleapis: google/cloud/secretmanager/v1
       # path inferred: <generate.output>/<name>

     - name: storage
       path: storage/
       # no googleapis = handwritten
   ```

   **Option B:**
   ```yaml
   libraries:
     - name: secretmanager
       source: google/cloud/secretmanager/v1  # googleapis
       location: generated/secretmanager       # optional, inferred if not set

     - name: storage
       location: storage/          # explicit for handwritten
   ```

## Implementation Details

1. **`Library.ExpandTemplate(template string) (string, error)`**
   - Expands `{name}` and `{api.path}` keywords
   - Validates that `{api.path}` is only used with exactly 1 API
   - Returns error if validation fails

2. **`Library.GeneratedLocation(generateOutput string) (string, error)`**
   - Returns explicit `Location` if set (for handwritten libraries)
   - Otherwise expands `generate.output` template
   - Returns error if template validation fails

3. **Command behavior:**
   ```bash
   # Go/Python: One library with multiple APIs
   librarian add secretmanager google/cloud/secretmanager/v1 google/cloud/secretmanager/v1beta2

   # Rust: One library per API version
   librarian add google-cloud-secretmanager-v1 google/cloud/secretmanager/v1
   librarian add google-cloud-secretmanager-v1beta2 google/cloud/secretmanager/v1beta2
   ```
