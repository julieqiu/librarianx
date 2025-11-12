Design: Python Code Generation
==============================

Objective
---------

Generate Python client libraries for Google Cloud APIs without using containers.

Background
----------

Librarian currently generates Go client libraries using the `internal/generate/golang` package. This package runs generation commands directly on the host without containers. The generation process parses BUILD.bazel files during `librarian add` and stores configuration in `librarian.yaml`. During `librarian generate`, the tool uses this saved configuration to construct protoc commands and generate code.

Python client libraries in google-cloud-python use a different system. The `.generator/cli.py` script runs inside a container and implements four commands: configure, generate, build, and release-init. This script parses BUILD.bazel using Python's starlark-pyo3 library, constructs protoc commands, runs synthtool for post-processing, and validates the generated code.

Users need Python generation to work the same way as Go generation - directly on the host without containers. This approach is faster, simpler to debug, and eliminates Docker as a dependency.

This document proposes a Go-based Python generator that follows the same patterns as the Go generator.

Overview
--------

The Python generator implements two commands in pure Go. The generate command combines configure, code generation, and validation into a single operation. Each command receives a configuration struct and performs its work by calling external tools like protoc, synthtool, and nox as subprocesses.

BUILD.bazel parsing moves to a shared `internal/bazel` package. This package parses BUILD.bazel files during `librarian add` and populates `config.API` structs. The configuration is saved to `librarian.yaml` and reused during generation without reading BUILD.bazel again.

The generator calls Python tools as subprocesses:

-	protoc with gapic-generator-python creates client code, setup.py, and basic README.rst
-	synthtool applies templates and enhances README.rst using .repo-metadata.json
-	nox runs tests

The design eliminates the mono-repo concept. Instead of checking if a repository is a mono-repo, the generator uses explicit paths from `librarian.yaml`.

Detailed Design
---------------

### Package Structure

The Python generator lives in `internal/generate/python`:

```
internal/
├── bazel/
│   └── python.go              # Parse BUILD.bazel → config.API
├── generate/
│   └── python/
│       ├── generate.go        # Main generate command
│       ├── generate_test.go
│       ├── protoc.go          # Construct protoc commands
│       ├── protoc_test.go
│       ├── postprocessor.go   # Run synthtool
│       └── postprocessor_test.go
└── release/
    └── python/
        ├── release.go         # Update versions, changelogs
        ├── release_test.go
        ├── changelog.go       # Changelog processing
        └── changelog_test.go
```

The generate package handles configuration, code generation, and validation in one command. The release package handles version updates and changelog management separately.

### BUILD.bazel Parsing

BUILD.bazel parsing moves to `internal/bazel`. This package provides language-specific parsers that return `config.API` structs directly.

The Python parser extracts py_gapic_library configuration:

```go
// internal/bazel/python.go
func ParsePythonGapicLibrary(buildPath, apiPath string) (*config.API, error)
```

This function reads BUILD.bazel, parses it with starlark-go, and returns a `config.API` with these fields populated:

-	Path (e.g., google/cloud/secretmanager/v1)
-	GrpcServiceConfig (e.g., secretmanager_grpc_service_config.json)
-	ServiceYAML (e.g., secretmanager_v1.yaml)
-	Transport (e.g., grpc+rest)
-	RestNumericEnums (true/false)
-	OptArgs (additional generator options)

If BUILD.bazel contains no py_gapic_library rule, the function returns nil. This indicates a proto-only library.

The `librarian add` command calls this parser and saves results to `librarian.yaml`. The `librarian generate` command reads the configuration from `librarian.yaml` without parsing BUILD.bazel again.

### Configuration Type

The `config.API` struct stores all information needed to generate code. This struct is shared across all languages:

```go
type API struct {
	Path string

	// Python fields
	GrpcServiceConfig string
	ServiceYAML       string
	Transport         string
	RestNumericEnums  bool
	OptArgs           []string

	// Metadata
	NamePretty           string
	ProductDocumentation string
	ReleaseLevel         string
	// ... other metadata fields
}
```

Language-specific fields coexist in the same struct. Python uses GrpcServiceConfig and Transport. Go uses different fields. This design keeps configuration simple and avoids nested structures.

### Protoc Command Construction

The `protoc` package constructs protoc commands from `config.API`:

```go
type ProtocCommand struct {
	Command string
	Args    []string
}

func BuildGapicCommand(api *config.API, version string) *ProtocCommand
func BuildProtoCommand(api *config.API) *ProtocCommand
```

BuildGapicCommand constructs commands for GAPIC generation:

```bash
protoc google/cloud/secretmanager/v1/*.proto \
  --proto_path=/source \
  --python_gapic_out=/output \
  --python_gapic_opt=metadata,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,transport=grpc+rest,rest-numeric-enums
```

BuildProtoCommand constructs commands for proto-only generation:

```bash
protoc google/cloud/language/v1/*.proto \
  --proto_path=/source \
  --python_out=/output \
  --pyi_out=/output
```

Both functions use glob patterns `*.proto` to find proto files. This matches the current behavior and works because the API path already specifies the correct directory.

### Generation Flow

The generate command follows these steps:

1.	If this is a new library (not in librarian.yaml yet):
	-	Add configuration defaults (source_roots, preserve_regex, remove_regex, tag_format, version)
	-	Create CHANGELOG.md files
	-	Update global CHANGELOG.md
2.	For each API in the library:
	-	Construct protoc command from config.API
	-	Run protoc with --python_gapic_out to generate code
	-	Stage code in owl-bot-staging directory
3.	Generate .repo-metadata.json from service_yaml and write to output directory
4.	Run synthtool post-processor (reads .repo-metadata.json from step 3)
5.	Copy README.rst to docs/
6.	Validate generated code (namespace, distribution name, tests)
7.	Clean up temporary files
8.	Write generated code to output directory

The key insight is that .repo-metadata.json is generated during the generate command and immediately consumed by synthtool. It is not stored in librarian.yaml or .librarian/generator-input - it is created fresh each time from service_yaml.

The staging directory structure depends on the API:

For GAPIC libraries with version (e.g., v1):

```
owl-bot-staging/{library-id}/v1/
```

For proto-only libraries:

```
owl-bot-staging/{library-id}/thing-py/google/thing/
```

### File Generation Flow

Python libraries are fully generated. The generation pipeline creates all necessary files:

1.	**protoc with gapic-generator-python** creates:

	-	Client library code (google/cloud/servicename/)
	-	setup.py with package metadata
	-	Basic README.rst with placeholders
	-	noxfile.py for testing
	-	Version files (gapic_version.py)

2.	**Generate .repo-metadata.json** from service_yaml:

	-	Read service_yaml file from the API directory
	-	Extract metadata (title, documentation URLs, API ID, etc.)
	-	Write .repo-metadata.json to the library output directory
	-	This file is ephemeral - created during generation, consumed by synthtool, not version controlled

3.	**Synthtool post-processor** enhances the generated code:

	-	Reads .repo-metadata.json from step 2
	-	Applies templates to populate README.rst with correct values (replaces placeholders)
	-	Runs formatters (black, isort)
	-	Copies owl-bot-staging to final location

Unlike Go libraries where configure creates README.md and version files upfront, Python libraries generate everything during the generate step. The only files created during configuration are CHANGELOG.md files (to preserve release history).

The .repo-metadata.json file is not stored in librarian.yaml or committed to .librarian/generator-input. It is generated fresh from service_yaml on every generation run.

### Synthtool Integration

The postprocessor package calls synthtool as a subprocess:

```go
func RunSynthtool(ctx context.Context, outputDir, libraryPath string) error {
	cmd := exec.CommandContext(ctx, "python3", "-c",
		"from synthtool.languages import python_mono_repo; python_mono_repo.owlbot_main('packages/google-cloud-language')")
	cmd.Dir = outputDir
	return cmd.Run()
}
```

This requires Python and synthtool to be installed on the host. The command runs in the output directory and receives the relative path to the library.

For repositories with custom owlbot.py files, the generator runs that script instead:

```go
if fileExists(filepath.Join(outputDir, "owlbot.py")) {
	cmd := exec.CommandContext(ctx, "python3", filepath.Join(outputDir, "owlbot.py"))
}
```

For proto-only libraries without noxfile.py, the generator runs isort and black directly:

```go
exec.CommandContext(ctx, "isort", outputDir).Run()
exec.CommandContext(ctx, "black", outputDir).Run()
```

### Metadata Generation

The generate command creates .repo-metadata.json from service_yaml files. This file contains library metadata that synthtool uses to populate templates:

```go
type RepoMetadata struct {
	Name                 string `json:"name"`
	NamePretty           string `json:"name_pretty"`
	APIDescription       string `json:"api_description"`
	ProductDocumentation string `json:"product_documentation"`
	ClientDocumentation  string `json:"client_documentation"`
	IssueTracker         string `json:"issue_tracker"`
	ReleaseLevel         string `json:"release_level"`
	Language             string `json:"language"`
	LibraryType          string `json:"library_type"`
	Repo                 string `json:"repo"`
	DistributionName     string `json:"distribution_name"`
	APIID                string `json:"api_id"`
	DefaultVersion       string `json:"default_version"`
	APIShortname         string `json:"api_shortname"`
}

func generateRepoMetadata(serviceYAMLPath, libraryID string) (*RepoMetadata, error)
```

This function reads the service_yaml file and extracts:

-	title → name_pretty
-	documentation.summary → api_description
-	publishing.documentation_uri → product_documentation
-	publishing.new_issue_uri → issue_tracker
-	name → api_id and api_shortname

The function writes the result to `{library-path}/.repo-metadata.json`. Synthtool reads this file and uses the values to populate README.rst templates.

### Build Validation

The build command validates generated code:

1.	Verify library namespace is in the approved list
2.	Verify distribution name matches library ID
3.	Run nox test sessions

Namespace verification finds all gapic_version.py and .proto files in the library. For each file, it determines the namespace and checks if it matches approved namespaces like "google.cloud" or "google.ai".

Distribution name verification builds the package metadata and confirms the name field matches the library ID.

Nox runs the unit-3.14 session with protobuf_implementation='upb':

```go
cmd := exec.CommandContext(ctx, "nox", "-s", "unit-3.14(protobuf_implementation='upb')", "-f", noxfilePath)
```

### Release Preparation

The release command updates version files and changelogs:

1.	Read release-init-request.json
2.	For each library with release_triggered=true:
	-	Update version in gapic_version.py, version.py, pyproject.toml, setup.py
	-	Update snippet metadata JSON files
	-	Update CHANGELOG.md with new entries
3.	Update global CHANGELOG.md if it exists
4.	Write modified files to output directory

Version updates use regex replacement to find and replace version strings. The regex patterns depend on the file type:

For gapic_version.py and version.py:

```
__version__ = "old" → __version__ = "new"
```

For pyproject.toml and setup.py:

```
version = "old" → version = "new"
```

Changelog updates insert a new section at the top with grouped changes. Changes are grouped by type (feat, fix, docs) and formatted with commit links.

### No Mono-repo Concept

The design eliminates the mono-repo concept. The `.generator/cli.py` checks if packages/ exists and adjusts paths accordingly. This heuristic is unnecessary because librarian.yaml explicitly defines all paths.

Instead of:

```go
if isMonoRepo {
	path = fmt.Sprintf("packages/%s", libraryID)
} else {
	path = "."
}
```

Use the configured path:

```go
path = library.Path // Explicitly set in librarian.yaml
```

Global CHANGELOG.md handling checks if the file exists:

```go
globalChangelogPath := filepath.Join(cfg.RepoDir, "CHANGELOG.md")
if _, err := os.Stat(globalChangelogPath); err == nil {
	// Update global changelog
}
```

This approach is simpler and more explicit. Configuration controls behavior instead of filesystem heuristics.

### Testing Strategy

Each package includes tests that verify behavior without external dependencies. Tests use:

-	Temporary directories for filesystem operations
-	Mock exec functions for subprocess calls
-	Test fixtures for BUILD.bazel and service_yaml files

Example test structure:

```go
func TestBuildGapicCommand(t *testing.T) {
	api := &config.API{
		Path:              "google/cloud/language/v1",
		GrpcServiceConfig: "language_grpc_service_config.json",
		ServiceYAML:       "language_v1.yaml",
		Transport:         "grpc+rest",
		RestNumericEnums:  true,
		OptArgs:           []string{"python-gapic-namespace=google.cloud"},
	}

	cmd := BuildGapicCommand(api, "1.0.0")

	// Verify command and args
}
```

Alternatives Considered
-----------------------

### Rewrite Synthtool in Go

We considered rewriting synthtool logic in Go to eliminate the Python dependency. This approach would give more control and make the system fully self-contained.

We chose to call synthtool as a subprocess because:

-	The existing synthtool implementation works and is maintained by the Python team
-	Rewriting would require maintaining template compatibility
-	The subprocess approach is simpler and faster to implement
-	We can migrate to a Go implementation later if needed

### Parse BUILD.bazel for Exact Proto Files

We considered parsing BUILD.bazel to get the exact list of proto files instead of using glob patterns.

We chose glob patterns because:

-	The API path already specifies the correct directory
-	Protoc handles dependencies automatically
-	Parsing proto_library rules adds complexity with minimal benefit
-	The current system uses globs and works correctly

### Call Pytest Directly

We considered calling pytest directly instead of nox.

We chose nox because:

-	Nox manages virtual environments automatically
-	Nox provides standardized test sessions
-	Google-cloud-python libraries use nox and have noxfile.py
-	Nox configuration can specify which sessions to run

### Keep Mono-repo Detection

We considered keeping the mono-repo detection logic from `.generator/cli.py`.

We chose to eliminate it because:

-	Librarian.yaml explicitly defines all paths
-	Filesystem heuristics are fragile
-	Configuration makes behavior explicit and testable
-	The same code works for both mono-repos and split repos
