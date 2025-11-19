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
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed documentation_overrides.yaml
var documentationOverridesYAML []byte

// Config represents the complete librarian.yaml configuration file.
type Config struct {
	// Version is the version of librarian that created this config.
	Version string `yaml:"version"`

	// Language is the primary language for this repository (go, python, rust).
	Language string `yaml:"language"`

	// Repo is the repository name (e.g., "googleapis/google-cloud-python").
	Repo string `yaml:"repo,omitempty"`

	// Sources contains references to external source repositories.
	Sources *Sources `yaml:"sources,omitempty"`

	// Default contains default generation settings.
	Default *Default `yaml:"default"`

	// NameOverrides contains overrides for auto-derived library names.
	// Allows customizing library names for specific APIs when the auto-derived
	// name doesn't match existing package names or conventions.
	// Key is API path (e.g., "google/api/apikeys/v2"), value is library name.
	NameOverrides map[string]string `yaml:"name_overrides,omitempty"`

	// Libraries contains configuration overrides for libraries that need special handling.
	// Only include libraries that differ from defaults.
	// Versions are looked up from the Versions map below.
	Libraries []*Library `yaml:"libraries,omitempty"`

	// Versions contains version numbers for all libraries.
	// This is the source of truth for release versions.
	// Key is library name, value is version string.
	Versions map[string]string `yaml:"versions,omitempty"`
}

// Sources contains references to external source repositories.
// Each entry maps a source name to its configuration.
type Sources struct {
	// Googleapis is the googleapis repository configuration.
	Googleapis *Source `yaml:"googleapis,omitempty"`

	// Python contains Python-specific source repository configurations.
	Python *PythonSources `yaml:"python,omitempty"`
}

// Source represents a single source repository configuration.
type Source struct {
	// Commit is the git commit hash or tag to use.
	Commit string `yaml:"commit"`
}

// Default contains default generation settings.
type Default struct {
	// Output is the directory where generated code is written (relative to repository root).
	Output string `yaml:"output,omitempty"`

	// Generate contains default generation configuration.
	Generate *DefaultGenerate `yaml:"generate,omitempty"`

	// Release contains default release configuration.
	Release *DefaultRelease `yaml:"release,omitempty"`

	// Rust contains Rust-specific default configuration.
	Rust *RustDefault `yaml:"rust,omitempty"`
}

// DefaultGenerate contains default generation configuration.
type DefaultGenerate struct {
	// All generates all client libraries with default configurations
	// for the language, unless otherwise specified.
	All bool `yaml:"all,omitempty"`

	// OneLibraryPer specifies packaging strategy: "api" or "channel".
	// - "api": Bundle all versions of a service into one library (Python, Go default)
	// - "channel": Create separate library per version (Rust, Dart default)
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
}

// DefaultRelease contains release configuration.
type DefaultRelease struct {
	// TagFormat is the template for git tags (e.g., '{name}/v{version}').
	// Supported placeholders: {name}, {version}
	TagFormat string `yaml:"tag_format,omitempty"`

	// Remote is the git remote name (e.g., "upstream", "origin").
	Remote string `yaml:"remote,omitempty"`

	// Branch is the default branch for releases (e.g., "main", "master").
	Branch string `yaml:"branch,omitempty"`
}

// Library represents a single library configuration entry.
type Library struct {
	// Name is the library name (e.g., "secretmanager", "storage").
	Name string `yaml:"name,omitempty"`

	// API specifies which googleapis API to generate from (for generated libraries).
	// Can be a string (protobuf API path) or an APIObject (for discovery APIs).
	// If both API and APIs are empty, this is a handwritten library.
	API string `yaml:"api,omitempty"`

	// Version is the library version.
	Version string `yaml:"version,omitempty"`

	// APIs specifies multiple API versions to bundle into one library (for multi-version libraries).
	// Alternative to API field for libraries that bundle multiple versions.
	APIs []string `yaml:"apis,omitempty"`

	// Generate contains per-library generate configuration.
	Generate *LibraryGenerate `yaml:"generate,omitempty"`

	// Path specifies the filesystem location (overrides computed location from defaults.output).
	// For generated libraries: overrides where code is generated to.
	// For handwritten libraries: specifies the source directory.
	Path string `yaml:"path,omitempty"`

	// Keep lists files/directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`

	// Disabled marks this library as disabled.
	Disabled bool `yaml:"disabled,omitempty"`

	// Transport overrides the default transport.
	Transport string `yaml:"transport,omitempty"`

	// RestNumericEnums overrides the default rest_numeric_enums setting.
	RestNumericEnums *bool `yaml:"rest_numeric_enums,omitempty"`

	// ReleaseLevel overrides the default release level.
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// Release contains per-library release configuration.
	Release *LibraryRelease `yaml:"release,omitempty"`

	// Publish contains per-library publish configuration.
	Publish *LibraryPublish `yaml:"publish,omitempty"`

	// LaunchStage overrides the derived launch stage.
	LaunchStage string `yaml:"launch_stage,omitempty"`

	// Destinations overrides the derived destinations.
	Destinations []string `yaml:"destinations,omitempty"`

	// GRPCServiceConfig is the gRPC service config JSON file path.
	GRPCServiceConfig string `yaml:"grpc_service_config,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear string `yaml:"copyright_year,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`

	// Python contains Python-specific library configuration.
	Python *PythonPackage `yaml:"python,omitempty"`

	// Go contains Go-specific library configuration.
	Go *GoModule `yaml:"go,omitempty"`

	// Dart contains Dart-specific library configuration.
	Dart *DartPackage `yaml:"dart,omitempty"`

	// APIServiceConfigs maps API paths to their service config file paths (runtime only, not serialized).
	// For single-API libraries: map[API]serviceConfigPath
	// For multi-API libraries: map[APIs[0]]path1, map[APIs[1]]path2, etc.
	APIServiceConfigs map[string]string `yaml:"-"`
}

// LibraryGenerate contains per-library generate configuration.
type LibraryGenerate struct {
	// Disabled prevents library generation.
	Disabled bool `yaml:"disabled,omitempty"`
}

// LibraryRelease contains per-library release configuration.
type LibraryRelease struct {
	// Disabled prevents library release.
	Disabled bool `yaml:"disabled,omitempty"`
}

// LibraryPublish contains per-library publish configuration.
type LibraryPublish struct {
	// Disabled prevents library from being published to package registries.
	Disabled bool `yaml:"disabled,omitempty"`
}

// New returns a new Config with language-specific defaults.
func New(lang, version, googleapisCommit, discoveryCommit string) (*Config, error) {
	return &Config{
		Version:  version,
		Language: lang,
		Sources: &Sources{
			Googleapis: &Source{
				Commit: googleapisCommit,
			},
		},
	}, nil
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	defer enc.Close()

	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return nil
}

// ReadDocumentationOverrides reads the embedded documentation overrides.
func ReadDocumentationOverrides() ([]RustDocumentationOverride, error) {
	var overrides []RustDocumentationOverride
	if err := yaml.Unmarshal(documentationOverridesYAML, &overrides); err != nil {
		return nil, fmt.Errorf("failed to unmarshal documentation overrides: %w", err)
	}
	return overrides, nil
}
