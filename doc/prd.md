# Objective

Librarian is a tool for managing Google Cloud client libraries across multiple languages.

## Background

Google Cloud client libraries are generated from API definitions in the
googleapis repository.
Managing these libraries involves:

- Generating client code from Protocol Buffer definitions
- Managing versions across hundreds of libraries
- Publishing to language-specific package registries (PyPI, crates.io, etc.)
- Supporting three types of libraries:
  1. **Fully generated** - Code is completely generated from API definitions
  2. **Fully handwritten** - Code is manually written and maintained
  3. **Hybrid** - Mix of generated and handwritten code

Different languages have different conventions:

- **Go** uses a monorepo structure with libraries at the repository root
- **Python** uses a packages/ directory with each library in its own subdirectory
- **Rust** uses a src/generated/ directory with API-versioned paths

These workflows share common patterns (generation,
versioning, releases) but differ in implementation details (build tools,
package formats, directory structures).

## The Problem

Managing client libraries requires:

1. **Configuration complexity** - Each library needs to know which APIs to generate,
which files to preserve, how to format code,
and where to publish
2. **Language-specific tooling** - Different generators (gapic-generator-python,
gapic-generator-go, Sidekick), build tools (pip,
cargo, go), and conventions
3. **Consistency across languages** - Same workflow should work for all languages with minimal changes
4. **Reproducible builds** - Generation must be deterministic and hermetic
5. **Version management** - Track versions across hundreds of libraries
6. **Release automation** - Publish to package registries with proper versioning and tagging

## The Solution

Librarian provides:

1. **Single configuration file** (`librarian.yaml`) - All settings in one place, version controlled
2. **Command-based container architecture** - Language-agnostic containers that execute explicit commands
3. **Unified CLI** - Same commands work across all languages (`librarian create`,
`librarian generate`, `librarian release`)
4. **Clear library types** - Configuration explicitly shows whether a library is generated, handwritten, or hybrid
5. **Hermetic builds** - Container images pin all dependencies; googleapis sources are immutable and SHA256-verified
6. **Language-specific customization** - Each language can have its own
conventions while sharing the same core workflow

## Key Design Principles

1. **Single source of truth** - `librarian.yaml` contains all configuration
2. **Explicit over implicit** - Commands show exactly what will run (visible in commands.json)
3. **Language-agnostic core** - Containers and CLI don't require language expertise
4. **Clear ownership** - Librarian team owns infrastructure (CLI,
containers), language teams own generators and conventions
5. **Reproducible** - Immutable source references, pinned dependencies, hermetic containers
6. **Discoverable** - Clear errors, helpful messages, `--help` flags
