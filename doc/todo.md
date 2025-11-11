# TODO List

- [x] Add set / unset for `tag_format`, `generate.output`, etc.
- [x] Add librarian add <library> <api>... (basic implementation done)
- [] Add librarian update googleapis
- [] Add librarian update --all
- [] Add librarian generate <library>
- [] Add librarian release <library>
- [] Add librarian publish <library>

## Design Discussion: Library paths and locations

### Current Status (2025-11-11)

We have a basic `librarian add` command:
```bash
librarian add secretmanager google/cloud/secretmanager/v1 google/cloud/secretmanager/v1beta2
```

Current `Library` struct:
```go
type Library struct {
    Name string `yaml:"name"`  // e.g., "secretmanager"
    Path string `yaml:"path"`  // e.g., "google/cloud/secretmanager/v1" (googleapis path)
}
```

### The Problem

Need to support two types of librarys:

1. **Generated librarys** (from googleapis)
   - Input: googleapis path (e.g., `google/cloud/secretmanager/v1`)
   - Output: filesystem location (language-dependent)
   - Example: `librarian add secretmanager google/cloud/secretmanager/v1`

2. **Handwritten/release-only librarys**
   - No googleapis path (no generation needed)
   - Code already exists at specific filesystem location
   - Example: `librarian add gcloud-mcp` (code at `packages/gcloud-mcp/`)

### Language-Specific Conventions

**Go:**
- Generated: `<generate.output>/<name>/` (e.g., `generated/secretmanager/`)
- Handwritten: `<root>/<name>/` (e.g., `gcloud-mcp/`)
- Both at top-level, whether generated or handwritten

**Rust:**
- Generated code: `src/generated/cloud/secretmanager/v1/src/`
- Generated metadata: `src/generated/cloud/secretmanager/v1/`
- Handwritten: `src/`

**Python:**
- Generated: `packages/google-cloud-secretmanager/`
- Handwritten: `packages/gcloud-mcp/`

### Design Questions

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
   # Generated librarys
   librarian add secretmanager google/cloud/secretmanager/v1

   # Handwritten librarys - how to specify filesystem path?
   librarian add gcloud-mcp --path packages/gcloud-mcp
   # OR
   librarian add gcloud-mcp packages/gcloud-mcp
   # OR
   librarian add gcloud-mcp  # infers path as <generate.output>/gcloud-mcp
   ```

4. **Example YAML structures:**

   **Option A:**
   ```yaml
   librarys:
     - name: secretmanager
       googleapis: google/cloud/secretmanager/v1
       # path inferred: <generate.output>/<name>

     - name: gcloud-mcp
       path: packages/gcloud-mcp
       # no googleapis = handwritten
   ```

   **Option B:**
   ```yaml
   librarys:
     - name: secretmanager
       source: google/cloud/secretmanager/v1  # googleapis
       location: generated/secretmanager       # optional, inferred if not set

     - name: gcloud-mcp
       location: packages/gcloud-mcp          # explicit for handwritten
   ```

### Open Questions

1. Is "library" about what to generate, what to release, or both?
2. Should filesystem paths always be computed/inferred, or explicitly stored?
3. How do we handle multi-package librarys (like gcloud-mcp spanning multiple dirs)?
4. What should the final field names be in the Library struct?

### Next Steps

- Decide on field naming convention
- Update Library struct with proper fields
- Implement path inference logic based on language and generate.output
- Update `librarian add` command syntax
- Add tests for generated vs handwritten librarys
