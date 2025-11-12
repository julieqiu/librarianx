# Alternatives Considered

This document describes alternative designs that were considered for the Librarian code generation system and explains why they were not chosen.

## Table of Contents

1. [Single Container Invocation with Configuration-Based Interface](#single-container-invocation-with-configuration-based-interface)
2. [Multiple Container Images per Language](#multiple-container-images-per-language)
3. [Request-Based Interface](#request-based-interface)
4. [Removing Version from librarian.yaml](#removing-version-from-librarianyaml)
5. [Renaming generate to infrastructure](#renaming-generate-to-infrastructure)
6. [Flat Release Commands (prepare/tag/publish)](#flat-release-commands-preparetagpublish)
7. [Three-Phase Release Process (release prepare/release tag/release publish)](#three-phase-release-process-release-preparerelease-tagrelease-publish)
8. [Multiple Configuration Files (Per-Edition Config Files)](#multiple-configuration-files-per-edition-config-files)
9. [Naming: Libraries vs Modules vs Packages vs Editions](#naming-libraries-vs-modules-vs-packages-vs-editions)

## Single Container Invocation with Configuration-Based Interface

We considered calling the container once per library generation with a configuration-based interface because of reduced Docker startup overhead and conceptual simplicity (one call instead of three).

**How it would work:**

Container receives `/config/generate.json` containing what to generate, executes the full pipeline (protoc → formatters → tests), and writes to `/output`.

**Example `generate.json`:**
```json
{
  "library": "google-cloud-secret-manager",
  "apis": [
    {
      "path": "google/cloud/secretmanager/v1",
      "service_config": "secretmanager_v1.yaml",
      "grpc_service_config": "secretmanager_grpc_service_config.json",
      "transport": "grpc+rest"
    }
  ],
  "metadata": {
    "name_pretty": "Secret Manager",
    "product_documentation": "https://cloud.google.com/secret-manager/docs"
  }
}
```

However, this approach had these costs:

1. **Language-specific logic in Go** - The librarian team would own Go code that needs to know how to construct protoc commands, which formatters to run, which tests to run, and the order of operations for each language
2. **Ownership confusion** - Container logic lives in librarian repo but requires Python/Go/Rust expertise to maintain
3. **Harder to debug** - Configuration goes in, code comes out - can't easily see what commands ran
4. **Less flexible** - Adding new generation steps requires changing Go code in the librarian repo

We ultimately went with a command-based interface where the container receives explicit commands to execute (`/commands/commands.json`) because of clearer ownership and simpler implementation. The command-based design keeps the container language-agnostic (~30 lines of Go), makes commands explicit and visible for debugging, and pushes language-specific knowledge to BUILD.bazel configuration (owned by language teams).

## Multiple Container Images per Language

We considered using separate container images for each phase (python-generator, python-formatter, python-tester) because of clearer separation of concerns and potentially smaller image sizes.

However, this approach had these costs:

1. **Multiple Dockerfiles** - Need to maintain 3 Dockerfiles per language (9 total for Python/Go/Rust)
2. **Multiple images to build and push** - More CI/CD complexity
3. **Version synchronization** - Need to keep all images in sync
4. **More orchestration complexity** - CLI needs to know which image to use for which phase

We ultimately went with a single container image per language because of simpler maintenance and deployment. A single image contains all dependencies for all phases, requires only one Dockerfile per language, and simplifies version management (one image version instead of three).

## Request-Based Interface

We considered using a request-based interface where the container receives `/request/generate-request.json` because of similarity to RPC patterns and potential for richer request metadata.

However, this approach had these costs:

1. **Same issues as configuration-based** - Container still needs to interpret the request and decide what commands to run
2. **Inconsistency in design docs** - Request-based mentioned in doc/newconfig.md while doc/generate.md used commands
3. **Less explicit** - Request describes what to generate, not how (commands to run)
4. **Harder to debug** - Can't easily see what commands ran

We ultimately went with a command-based interface (`/commands/commands.json`) because of explicitness and debuggability. Commands show exactly what will run, making it easy to inspect commands.json to see the exact commands that executed.

## Removing Version from librarian.yaml

We considered removing the `version` field from edition configurations in `librarian.yaml` and using language-specific version files (version.go, pyproject.toml, Cargo.toml) as the single source of truth because of eliminating duplication.

However, this approach had these costs:

1. **Language-specific parsing** - Librarian CLI needs to know how to parse version.go, pyproject.toml, Cargo.toml
2. **Slower reads** - Reading version requires parsing language-specific file formats
3. **Added complexity** - Different parsing logic for each language

We ultimately went with keeping `version` in edition configurations in `librarian.yaml` as a cache for fast access because of simplicity and performance. The librarian tool manages version consistency between `librarian.yaml` and language-specific files, providing fast YAML-based reads without language-specific knowledge.

## Renaming `generate` to `infrastructure`

We considered renaming the top-level `generate` section to `infrastructure` because of distinguishing between "how to generate" (infrastructure: container, googleapis) and "what to generate" (APIs, metadata in editions).

However, this approach had these costs:

1. **User expectation mismatch** - Users expect `generate` for generation-related configuration
2. **Inconsistency** - Different names at top level and edition level is confusing
3. **No real benefit** - The distinction is clear from context without renaming

User feedback: "I do not like the name infrastructure. The design is called generate."

We ultimately went with using `generate` at both top level and edition level because of consistency and user expectations. The distinction is clear from context: top level contains output directory and defaults (how), edition level contains APIs and metadata (what).

## Flat Release Commands (prepare/tag/publish)

We considered using flat command names without the `release` prefix (`librarian prepare`, `librarian tag`, `librarian publish`) because of shorter command names.

However, this approach had these costs:

1. **Namespace pollution** - Top-level commands should be high-level operations, not sub-phases
2. **Ambiguity** - `librarian tag` could mean many things (git tag? container tag?)
3. **Discoverability** - `librarian release --help` wouldn't show these commands
4. **Inconsistency** - Other multi-step operations use subcommands (e.g., `librarian config set`)

We ultimately went with subcommand structure (`librarian release <phase>`) because of clarity and discoverability. All release operations are grouped under the `release` namespace, `librarian release --help` shows all phases, and the pattern matches other multi-step commands.

## Two-Phase vs Three-Phase Release Process

We considered consolidating into two phases (combine `tag` and `publish` into one command) because of fewer commands to remember.

However, this approach had these costs:

1. **Less flexibility** - Can't tag without publishing (e.g., for internal releases)
2. **Couples git and registry operations** - Tagging is a git operation, publishing is a registry operation
3. **Less CI/CD friendly** - May want to run tag and publish in different jobs/environments
4. **Harder rollback** - Can't tag first, verify, then decide whether to publish

We ultimately went with three separate phases (`release prepare`, `release tag`, `release publish`) because of flexibility and clear separation of concerns. Each phase maps to a distinct operation (commit, git tag, registry push), users can prepare locally and review before tagging, and each phase can run in different CI/CD jobs for better control.

## Multiple Configuration Files (Per-Edition Config Files)

We considered using multiple configuration files where each edition has its own configuration file (e.g., `librarian.yaml` for repository settings and `<edition>/.librarian.yaml` for edition-specific settings) because of separation of concerns and reduced merge conflicts.

**How it would work:**

Repository-level config at `librarian.yaml`:
```yaml
version: v0.5.0
language: go

sources:
  googleapis:
    url: https://...
    sha256: ...

generate:
  container:
    image: ...
    tag: latest
  output_dir: ./
```

Edition-level config at `secretmanager/.librarian.yaml`:
```yaml
name: secretmanager
version: 0.1.0

generate:
  apis:
    - path: google/cloud/secretmanager/v1
      grpc_service_config: secretmanager_grpc_service_config.json
      service_yaml: secretmanager_v1.yaml
      transport: grpc+rest
```

However, this approach had these costs:

1. **Hard to discover information** - Need to search multiple files to understand repository configuration
2. **Scattered state** - Edition list is implicit (discovered by finding `.librarian.yaml` files)
3. **Harder to audit** - Can't see all editions and their versions in one place
4. **More file operations** - CLI needs to read N+1 files for N editions
5. **Git history fragmentation** - Changes to edition configs spread across many files

We ultimately went with a single `librarian.yaml` file containing all repository and edition configuration because of ease of discovery and auditing. All configuration is in one place, making it easy to understand the entire repository state at a glance. The single file serves as a litmus test for complexity: if `librarian.yaml` becomes very long (e.g., thousands of lines), this is a sign that the configuration language may be too verbose and needs to be redesigned with better defaults, conventions, or abstractions. A well-designed configuration language should support 50-100+ editions in a readable single file.

## Naming: Libraries vs Modules vs Packages vs Editions

We considered several names for the releasable units that librarian generates and manages.

**Libraries**: We considered using "libraries" because it's the common term for client libraries (e.g., "google-cloud-secretmanager library").

However, this approach had these costs:

1. **Not generic enough** - Librarian can release things beyond client libraries, such as the librarian CLI tool itself
2. **Language-specific connotation** - "Library" implies a language-specific artifact

**Modules**: We considered using "modules" because it's a common term in software development.

However, this approach had these costs:

1. **Overloaded in Go** - In Go, "module" has a specific meaning (go.mod defines a module)
2. **Inconsistent across languages** - Go uses "module" for what Python calls a "package"

**Packages**: We considered using "packages" because it's a common term in package managers.

However, this approach had these costs:

1. **Overloaded in Rust** - In Rust, "package" has a specific meaning (Cargo.toml defines a package)
2. **Inconsistent across languages** - Rust uses "package" for what Go calls a "module"
3. **Swapped terminology** - Go and Rust use "module" and "package" to mean opposite things

We ultimately went with "editions" because of neutrality and semantic accuracy. "Editions" is language-neutral and has no overloaded meaning in programming languages. The term comes from publishing, where an edition means "a specific version of a publication with changes to the content, format, or other features" - which accurately captures what librarian manages. Like a book publisher releases different editions of a book, librarian generates and releases different editions of client libraries, tools, and other software artifacts.
