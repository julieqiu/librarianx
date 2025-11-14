// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete librarian.yaml configuration file.
type Config struct {
	// Version is the version of librarian that created this config.
	Version string `yaml:"version"`

	// Language is the primary language for this repository (go, python, rust).
	Language string `yaml:"language,omitempty"`

	// Container contains the container image configuration.
	Container *Container `yaml:"container,omitempty"`

	// Sources contains references to external source repositories.
	Sources Sources `yaml:"sources,omitempty"`

	// Global contains global repository settings.
	Global *Global `yaml:"global,omitempty"`

	// Defaults contains default generation settings.
	Defaults *Defaults `yaml:"defaults,omitempty"`

	// Generate contains generation configuration.
	Generate *Generate `yaml:"generate,omitempty"`

	// Release contains release configuration.
	Release *Release `yaml:"release,omitempty"`

	// Libraries contains the list of library configurations.
	// Each entry can be either:
	// - A string API path (short syntax): "google/cloud/secretmanager/v1"
	// - A map with API path as key and overrides as value (extended syntax)
	Libraries []LibraryEntry `yaml:"libraries,omitempty"`
}

// Container contains the container image configuration.
type Container struct {
	// Image is the container registry path (without tag).
	Image string `yaml:"image"`

	// Tag is the container image tag (e.g., "latest", "v1.0.0").
	Tag string `yaml:"tag"`
}

// Sources contains references to external source repositories.
type Sources struct {
	// Googleapis is the googleapis source repository.
	Googleapis *Source `yaml:"googleapis,omitempty"`

	// Showcase is the gapic-showcase source repository (for testing).
	Showcase *Source `yaml:"showcase,omitempty"`

	// Discovery is the discovery-artifact-manager source repository.
	Discovery *Source `yaml:"discovery,omitempty"`

	// ProtobufSrc is the protobuf source repository.
	ProtobufSrc *Source `yaml:"protobuf_src,omitempty"`

	// Conformance is the conformance test source repository.
	Conformance *Source `yaml:"conformance,omitempty"`
}

// Source represents an external source repository.
type Source struct {
	// URL is the download URL for the source tarball.
	URL string `yaml:"url"`

	// SHA256 is the hash for integrity verification.
	SHA256 string `yaml:"sha256"`

	// ExtractedName is the directory name after extraction (if different from default).
	ExtractedName string `yaml:"extracted_name,omitempty"`

	// Subdir is a subdirectory within the extracted archive to use.
	Subdir string `yaml:"subdir,omitempty"`
}

// Global contains global repository settings.
type Global struct {
	// FilesAllowlist is the list of files that can be modified globally.
	FilesAllowlist []FileAllowlist `yaml:"files_allowlist,omitempty"`
}

// FileAllowlist represents a file that can be modified globally.
type FileAllowlist struct {
	// Path is the file path.
	Path string `yaml:"path"`

	// Permissions specifies the access level (read-only, read-write, write-only).
	Permissions string `yaml:"permissions"`
}

// Defaults contains default generation settings.
type Defaults struct {
	// Output is the directory where generated code is written (relative to repository root).
	Output string `yaml:"output,omitempty"`

	// OneLibraryPer specifies packaging strategy: "service" or "version".
	// - "service": Bundle all versions of a service into one library (Python, Go default)
	// - "version": Create separate library per version (Rust, Dart default)
	OneLibraryPer string `yaml:"one_library_per,omitempty"`

	// Transport is the default transport protocol (e.g., "grpc+rest", "grpc").
	Transport string `yaml:"transport,omitempty"`

	// RestNumericEnums indicates whether to use numeric enums in REST.
	RestNumericEnums bool `yaml:"rest_numeric_enums,omitempty"`

	// ReleaseLevel is the default release level ("stable" or "preview").
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// ExcludeAPIs is a list of API path patterns to exclude from wildcard discovery.
	// Patterns can use * as wildcard (e.g., "google/ads/*", "google/actions/*").
	ExcludeAPIs []string `yaml:"exclude_apis,omitempty"`

	// Rust contains default Rust-specific settings.
	Rust *RustDefaults `yaml:"rust,omitempty"`
}

// Generate contains generation configuration.
type Generate struct {
	// Output is the directory where generated code is written (relative to repository root).
	Output string `yaml:"output,omitempty"`
}

// Release contains release configuration.
type Release struct {
	// TagFormat is the template for git tags (e.g., '{name}/v{version}').
	// Supported placeholders: {name}, {version}
	TagFormat string `yaml:"tag_format,omitempty"`

	// Remote is the git remote name (e.g., "upstream", "origin").
	Remote string `yaml:"remote,omitempty"`

	// Branch is the default branch for releases (e.g., "main", "master").
	Branch string `yaml:"branch,omitempty"`

	// IgnoredChanges are files to ignore when detecting changes for release.
	IgnoredChanges []string `yaml:"ignored_changes,omitempty"`
}

// LibraryEntry represents a single library configuration entry.
// It can be either:
// - Short syntax: just a wildcard string ("*")
// - Extended syntax: a map with library name as key and LibraryConfig as value.
type LibraryEntry struct {
	// Name is the library name (package name), e.g., "google-cloud-secretmanager"
	// For wildcard entries, this is "*"
	Name string

	// Config contains optional configuration overrides.
	// If nil, all defaults are used.
	Config *LibraryConfig
}

// LibraryConfig contains configuration overrides for a library.
type LibraryConfig struct {
	// API specifies which googleapis API to generate from (for generated libraries).
	// Can be a string (protobuf API path) or an APIObject (for discovery APIs).
	// If both API and APIs are empty, this is a handwritten library.
	API interface{} `yaml:"api,omitempty"`

	// APIs specifies multiple API versions to bundle into one library (for multi-version libraries).
	// Alternative to API field for libraries that bundle multiple versions.
	APIs []string `yaml:"apis,omitempty"`

	// Path specifies the filesystem location (overrides computed location from defaults.output).
	// For generated libraries: overrides where code is generated to.
	// For handwritten libraries: specifies the source directory.
	Path string `yaml:"path,omitempty"`

	// Keep lists files/directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`

	// Disabled marks this library as disabled.
	Disabled bool `yaml:"disabled,omitempty"`

	// Reason explains why the library is disabled (required if disabled=true).
	Reason string `yaml:"reason,omitempty"`

	// Transport overrides the default transport.
	Transport string `yaml:"transport,omitempty"`

	// RestNumericEnums overrides the default rest_numeric_enums setting.
	RestNumericEnums *bool `yaml:"rest_numeric_enums,omitempty"`

	// ReleaseLevel overrides the default release level.
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// Release contains per-library release configuration.
	Release *LibraryRelease `yaml:"release,omitempty"`

	// Language-specific library configurations
	Rust   *RustLibrary   `yaml:"rust,omitempty"`
	Dart   *DartLibrary   `yaml:"dart,omitempty"`
	Python *PythonLibrary `yaml:"python,omitempty"`
	Go     *GoLibrary     `yaml:"go,omitempty"`
	Java   *JavaLibrary   `yaml:"java,omitempty"`
	Node   *NodeLibrary   `yaml:"node,omitempty"`
	Dotnet *DotnetLibrary `yaml:"dotnet,omitempty"`

	// Overrides for derived fields
	LaunchStage  string   `yaml:"launch_stage,omitempty"`
	Destinations []string `yaml:"destinations,omitempty"`

	// BazelMetadata contains metadata extracted from BUILD.bazel files.
	// This is populated automatically during conversion or when adding a library.
	BazelMetadata *BazelMetadata `yaml:"bazel_metadata,omitempty"`
}

// RustLibrary contains Rust-specific library configuration.
type RustLibrary struct {
	// PerServiceFeatures enables per-service feature flags.
	PerServiceFeatures bool `yaml:"per_service_features,omitempty"`

	// Additional fields can be added as needed
	ModulePath                string              `yaml:"module_path,omitempty"`
	TemplateOverride          string              `yaml:"template_override,omitempty"`
	TitleOverride             string              `yaml:"title_override,omitempty"`
	DescriptionOverride       string              `yaml:"description_override,omitempty"`
	PackageNameOverride       string              `yaml:"package_name_override,omitempty"`
	RootName                  string              `yaml:"root_name,omitempty"`
	Roots                     []string            `yaml:"roots,omitempty"`
	DefaultFeatures           []string            `yaml:"default_features,omitempty"`
	ExtraModules              []string            `yaml:"extra_modules,omitempty"`
	IncludeList               []string            `yaml:"include_list,omitempty"`
	IncludedIds               []string            `yaml:"included_ids,omitempty"`
	SkippedIds                []string            `yaml:"skipped_ids,omitempty"`
	NameOverrides             string              `yaml:"name_overrides,omitempty"`
	PackageDependencies       []PackageDependency `yaml:"package_dependencies,omitempty"`
	DisabledRustdocWarnings   []string            `yaml:"disabled_rustdoc_warnings,omitempty"`
	DisabledClippyWarnings    []string            `yaml:"disabled_clippy_warnings,omitempty"`
	HasVeneer                 bool                `yaml:"has_veneer,omitempty"`
	RoutingRequired           bool                `yaml:"routing_required,omitempty"`
	IncludeGrpcOnlyMethods    bool                `yaml:"include_grpc_only_methods,omitempty"`
	GenerateSetterSamples     bool                `yaml:"generate_setter_samples,omitempty"`
	PostProcessProtos         bool                `yaml:"post_process_protos,omitempty"`
	DetailedTracingAttributes bool                `yaml:"detailed_tracing_attributes,omitempty"`
	NotForPublication         bool                `yaml:"not_for_publication,omitempty"`
}

// DartLibrary contains Dart-specific library configuration.
type DartLibrary struct {
	// APIKeysEnvironmentVariables specifies environment variable for API keys.
	APIKeysEnvironmentVariables string `yaml:"api_keys_environment_variables,omitempty"`

	// DevDependencies lists additional dev dependencies.
	DevDependencies []string `yaml:"dev_dependencies,omitempty"`
}

// PythonLibrary contains Python-specific library configuration.
type PythonLibrary struct {
	// RestAsyncIOEnabled enables async I/O for REST transport.
	RestAsyncIOEnabled bool `yaml:"rest_async_io_enabled,omitempty"`

	// UnversionedPackageDisabled disables unversioned package generation.
	UnversionedPackageDisabled bool `yaml:"unversioned_package_disabled,omitempty"`
}

// GoLibrary contains Go-specific library configuration.
type GoLibrary struct {
	// RenamedServices maps original service names to renamed versions.
	// Example: {"Publisher": "TopicAdmin", "Subscriber": "SubscriptionAdmin"}
	RenamedServices map[string]string `yaml:"renamed_services,omitempty"`
}

// JavaLibrary contains Java-specific library configuration.
type JavaLibrary struct {
	// Package specifies the Java package name.
	// Example: "com.google.cloud.logging.v2"
	Package string `yaml:"package,omitempty"`

	// ServiceClassNames maps proto service names to generated class names.
	// Example: {"google.logging.v2.LoggingServiceV2": "Logging"}
	ServiceClassNames map[string]string `yaml:"service_class_names,omitempty"`
}

// NodeLibrary contains Node.js-specific library configuration.
type NodeLibrary struct {
	// SelectiveMethods lists method selectors for selective generation.
	// Example: ["google.storage.v2.Storage.GetBucket", "google.storage.v2.Storage.CreateBucket"]
	SelectiveMethods []string `yaml:"selective_methods,omitempty"`
}

// DotnetLibrary contains .NET-specific library configuration.
type DotnetLibrary struct {
	// RenamedServices maps original service names to renamed versions.
	// Example: {"Subscriber": "SubscriberServiceApi", "Publisher": "PublisherServiceApi"}
	RenamedServices map[string]string `yaml:"renamed_services,omitempty"`

	// RenamedResources maps resource names to renamed versions for disambiguation.
	// Example: {"datalabeling.googleapis.com/Dataset": "DataLabelingDataset"}
	RenamedResources map[string]string `yaml:"renamed_resources,omitempty"`
}

// BazelMetadata contains metadata extracted from googleapis BUILD.bazel files.
// This metadata is populated automatically during conversion or when adding a library,
// and is cached here so that BUILD.bazel files don't need to be parsed repeatedly.
type BazelMetadata struct {
	// Transport is the transport protocol (e.g., "grpc", "rest", "grpc+rest").
	Transport string `yaml:"transport,omitempty"`

	// RestNumericEnums indicates whether to use numeric enums in REST.
	RestNumericEnums bool `yaml:"rest_numeric_enums,omitempty"`

	// ReleaseLevel is the release level (e.g., "ga", "beta", "alpha").
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// GRPCServiceConfig is the gRPC service config JSON file path.
	GRPCServiceConfig string `yaml:"grpc_service_config,omitempty"`

	// Go contains Go-specific metadata from BUILD.bazel.
	Go *BazelGoMetadata `yaml:"go,omitempty"`

	// Python contains Python-specific metadata from BUILD.bazel.
	Python *BazelPythonMetadata `yaml:"python,omitempty"`
}

// BazelGoMetadata contains Go-specific metadata from BUILD.bazel.
type BazelGoMetadata struct {
	// ImportPath is the Go package import path.
	// Example: "cloud.google.com/go/batch/apiv1;batchpb"
	ImportPath string `yaml:"import_path,omitempty"`

	// Metadata indicates whether to generate gapic_metadata.json.
	Metadata bool `yaml:"metadata,omitempty"`

	// Diregapic indicates whether this is a DIREGAPIC (Discovery REST GAPIC).
	Diregapic bool `yaml:"diregapic,omitempty"`
}

// BazelPythonMetadata contains Python-specific metadata from BUILD.bazel.
type BazelPythonMetadata struct {
	// OptArgs contains additional options passed to the generator.
	// Example: ["warehouse-package-name=google-cloud-batch"]
	OptArgs []string `yaml:"opt_args,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for LibraryEntry.
// It supports both:
// - String syntax: "*" (wildcard for auto-discovery)
// - Map syntax: {"google-cloud-secretmanager": {api: "google/cloud/secretmanager/v1", ...}}.
func (e *LibraryEntry) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as a string first (for wildcard)
	var name string
	if err := unmarshal(&name); err == nil {
		e.Name = name
		e.Config = nil
		return nil
	}

	// Try to unmarshal as a map with single key
	var m map[string]LibraryConfig
	if err := unmarshal(&m); err != nil {
		return fmt.Errorf("library entry must be either a string (wildcard) or a map with library name as key")
	}

	if len(m) != 1 {
		return fmt.Errorf("library entry map must have exactly one key (library name), got %d keys", len(m))
	}

	// Extract the single key-value pair
	for name, config := range m {
		e.Name = name
		e.Config = &config
		return nil
	}

	return fmt.Errorf("library entry map is empty")
}

// MarshalYAML implements custom marshaling for LibraryEntry.
func (e LibraryEntry) MarshalYAML() (interface{}, error) {
	// If no config overrides (e.g., wildcard), use string syntax
	if e.Config == nil {
		return e.Name, nil
	}

	// Use map syntax with library name as key
	return map[string]LibraryConfig{
		e.Name: *e.Config,
	}, nil
}

// Library represents an library.
type Library struct {
	// Name is the library name (e.g., "secretmanager").
	Name string `yaml:"name"`

	// Version is the current released version.
	Version string `yaml:"version,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear int `yaml:"copyright_year,omitempty"`

	// ModulePathVersion is the module version suffix (e.g., "v2").
	ModulePathVersion string `yaml:"module_path_version,omitempty"`

	// SourceRoots are the source directories for this library.
	SourceRoots []string `yaml:"source_roots,omitempty"`

	// Release contains per-library release configuration.
	Release *LibraryRelease `yaml:"release,omitempty"`

	// Generate contains generation configuration for this library.
	Generate *LibraryGenerate `yaml:"generate,omitempty"`

	// Apis is the list of googleapis paths for generated librarys.
	Apis []string `yaml:"apis,omitempty"`

	// Location is the explicit filesystem path (optional).
	// If not set and apis is present, computed from generate.output template.
	Location string `yaml:"location,omitempty"`
}

// LibraryRelease contains per-library release configuration.
type LibraryRelease struct {
	// Disabled prevents automatic releases.
	Disabled bool `yaml:"disabled,omitempty"`
}

// LibraryGenerate contains generation configuration for a library.
type LibraryGenerate struct {
	// APIs is the list of API configurations.
	APIs []API `yaml:"apis,omitempty"`

	// Keep is the list of files/directories not overwritten during generation.
	Keep []string `yaml:"keep,omitempty"`

	// DeleteOutputPaths is the list of paths to delete from output.
	DeleteOutputPaths []string `yaml:"delete_output_paths,omitempty"`
}

// API represents an API configuration.
type API struct {
	// Path is the API path relative to googleapis root (e.g., "google/cloud/secretmanager/v1").
	Path string `yaml:"path"`

	// HasGAPIC indicates whether a GAPIC library rule was found.
	// If false, this is a proto-only library.
	HasGAPIC bool `yaml:"has_gapic,omitempty"`

	// GRPCServiceConfig is the name of the gRPC service config JSON file.
	GRPCServiceConfig string `yaml:"grpc_service_config,omitempty"`

	// ServiceYAML is the client config file in the API version directory.
	ServiceYAML string `yaml:"service_yaml,omitempty"`

	// Transport is typically one of "grpc", "rest" or "grpc+rest".
	Transport string `yaml:"transport,omitempty"`

	// RestNumericEnums indicates whether to use numeric enums in REST.
	RestNumericEnums bool `yaml:"rest_numeric_enums,omitempty"`

	// ReleaseLevel is typically one of "beta", "" (same as beta) or "ga".
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// Python contains Python-specific configuration.
	Python *PythonAPI `yaml:"python,omitempty"`

	// Go contains Go-specific configuration.
	Go *GoAPI `yaml:"go,omitempty"`
}

// PythonAPI contains Python-specific API configuration.
type PythonAPI struct {
	// OptArgs contains additional options passed to the generator.
	// E.g., ["warehouse-package-name=google-cloud-secret-manager"]
	OptArgs []string `yaml:"opt_args,omitempty"`
}

// GoAPI contains Go-specific API configuration.
type GoAPI struct {
	// ClientDirectory is the custom client directory (optional).
	ClientDirectory string `yaml:"client_directory,omitempty"`

	// DisableGapic disables GAPIC generation for this API.
	DisableGapic bool `yaml:"disable_gapic,omitempty"`

	// ProtoPackage is the custom protobuf package name.
	ProtoPackage string `yaml:"proto_package,omitempty"`

	// NestedProtos are additional nested proto files to include.
	NestedProtos []string `yaml:"nested_protos,omitempty"`
}

// Read reads the configuration from a file.
func Read(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &c, nil
}

// Write writes the configuration to a file.
func (c *Config) Write(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Set sets a key in the config based on the key path.
func (c *Config) Set(key, value string) error {
	switch key {
	case "release.tag_format":
		if c.Release == nil {
			c.Release = &Release{}
		}
		c.Release.TagFormat = value
	case "generate.output":
		if c.Generate == nil {
			c.Generate = &Generate{}
		}
		c.Generate.Output = value
	default:
		return fmt.Errorf("invalid key: %s", key)
	}
	return nil
}

// Unset removes a key value from the config based on the key path.
func (c *Config) Unset(key string) error {
	switch key {
	case "release.tag_format":
		if c.Release != nil {
			c.Release.TagFormat = ""
		}
	case "generate.output":
		if c.Generate != nil {
			c.Generate.Output = ""
		}
	default:
		return fmt.Errorf("invalid key: %s", key)
	}
	return nil
}

// New creates a new Config with default settings.
// If language is specified, it includes language-specific configuration.
// If source is provided, it is added to the Sources.
func New(version, language string, source *Source) *Config {
	cfg := &Config{
		Version: version,
		Release: &Release{
			TagFormat: "{name}/v{version}",
		},
	}

	if language == "" {
		return cfg
	}

	cfg.Language = language

	if source != nil {
		cfg.Sources = Sources{
			Googleapis: source,
		}
	}

	return cfg
}

// GetOneLibraryPer returns the packaging strategy from defaults.
// Returns "service" for Python/Go, "version" for Rust/Dart by default.
func (c *Config) GetOneLibraryPer() string {
	// If explicitly set in defaults, use that
	if c.Defaults != nil && c.Defaults.OneLibraryPer != "" {
		return c.Defaults.OneLibraryPer
	}

	// Otherwise, use language-specific defaults
	switch c.Language {
	case "python", "go":
		return "service"
	case "rust", "dart":
		return "version"
	default:
		return "service"
	}
}

// Add adds a library to the config using the new name-based format.
// name is the library name (package name), e.g., "google-cloud-secretmanager".
// config contains optional overrides (can be nil for default configuration).
func (c *Config) Add(name string, config *LibraryConfig) error {
	if name == "" {
		return fmt.Errorf("library name cannot be empty")
	}

	// Check if library with same name already exists
	for _, entry := range c.Libraries {
		if entry.Name == name {
			return fmt.Errorf("library with name %q already exists", name)
		}
	}

	c.Libraries = append(c.Libraries, LibraryEntry{
		Name:   name,
		Config: config,
	})

	return nil
}

// AddLegacy adds a library using the old format (for backward compatibility).
// Deprecated: Use Add with proper API configuration instead.
func (c *Config) AddLegacy(name string, apis []string, location string, googleapisRoot string) error {
	if name == "" {
		return fmt.Errorf("library name cannot be empty")
	}

	if len(apis) == 0 && location == "" {
		return fmt.Errorf("library must have at least one API or a location")
	}

	config := &LibraryConfig{}

	if len(apis) == 1 {
		// Single API - use `api` field
		config.API = apis[0]
	} else if len(apis) > 1 {
		// Multiple APIs - use `apis` field
		config.APIs = apis
	}
	// If no APIs, this is a handwritten library

	if location != "" {
		config.Path = location
	}

	// Extract language-specific settings from service config if available
	if len(apis) > 0 && googleapisRoot != "" && c.Language != "" {
		// Use the first API to find service config
		apiPath := apis[0]
		if extracted, err := ExtractServiceConfigSettings(googleapisRoot, apiPath, c.Language); err != nil {
			// Don't fail if extraction fails - just log and continue
			fmt.Fprintf(os.Stderr, "Warning: failed to extract service config settings: %v\n", err)
		} else if extracted != nil {
			// Merge extracted settings into config
			if extracted.Java != nil {
				config.Java = extracted.Java
			}
			if extracted.Python != nil {
				config.Python = extracted.Python
			}
			if extracted.Go != nil {
				config.Go = extracted.Go
			}
			if extracted.Node != nil {
				config.Node = extracted.Node
			}
			if extracted.Dotnet != nil {
				config.Dotnet = extracted.Dotnet
			}
		}
	}

	return c.Add(name, config)
}

// ExpandTemplate expands template keywords in a string.
// Supported keywords:
//   - {name} - The library name
//   - {api.path} - The API path (requires exactly one API in the library)
//
// Returns the expanded template string and an error if validation fails.
func (e *Library) ExpandTemplate(template string) (string, error) {
	result := template

	// Replace {name} with library name
	result = strings.ReplaceAll(result, "{name}", e.Name)

	// Replace {api.path} with API path (requires exactly one API)
	if strings.Contains(result, "{api.path}") {
		if len(e.Apis) != 1 {
			return "", fmt.Errorf("template uses {api.path} but library %q has %d APIs (expected exactly 1)", e.Name, len(e.Apis))
		}
		result = strings.ReplaceAll(result, "{api.path}", e.Apis[0])
	}

	return result, nil
}

// GeneratedLocation returns the filesystem location where generated code should be written.
// If Location is explicitly set, returns that.
// Otherwise, expands the generate.output template with library data.
// Returns an error if template expansion fails validation.
func (e *Library) GeneratedLocation(generateOutput string) (string, error) {
	if e.Location != "" {
		return e.Location, nil
	}
	return e.ExpandTemplate(generateOutput)
}

// ToLibrary converts a LibraryEntry to a Library for backward compatibility.
// This is a temporary method to help with migration to the new config format.
func (e *LibraryEntry) ToLibrary() *Library {
	lib := &Library{
		Name: e.Name,
	}

	if e.Config != nil {
		// Extract API paths
		if e.Config.API != nil {
			// Single API (string or object)
			if apiStr, ok := e.Config.API.(string); ok {
				lib.Apis = []string{apiStr}
			}
		} else if len(e.Config.APIs) > 0 {
			// Multiple APIs
			lib.Apis = e.Config.APIs
		}

		if e.Config.Path != "" {
			lib.Location = e.Config.Path
		}
		if e.Config.Release != nil {
			lib.Release = e.Config.Release
		}
		if len(e.Config.Keep) > 0 {
			if lib.Generate == nil {
				lib.Generate = &LibraryGenerate{}
			}
			lib.Generate.Keep = e.Config.Keep
		}
	}

	return lib
}

// FindLibraryByName finds a library entry by name.
// Returns the library entry and its index, or nil and -1 if not found.
func (c *Config) FindLibraryByName(name string) (*LibraryEntry, int) {
	for i := range c.Libraries {
		entry := &c.Libraries[i]
		if entry.Name == name {
			return entry, i
		}
	}
	return nil, -1
}
