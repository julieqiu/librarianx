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

	"gopkg.in/yaml.v3"
)

// Config represents the complete librarian.yaml configuration file.
type Config struct {
	// Version is the version of librarian that created this config.
	Version string `yaml:"version"`

	// Language is the primary language for this repository (go, python, rust).
	Language string `yaml:"language"`

	// Sources contains references to external source repositories.
	Sources Sources `yaml:"sources,omitempty"`

	// Defaults contains default generation settings.
	Defaults ConfigDefault `yaml:"default"`

	// Release contains release configuration.
	Release *ConfigRelease `yaml:"release,omitempty"`

	// Libraries contains the list of library configurations.
	// Each entry can be either:
	// - A string API path (short syntax): "google/cloud/secretmanager/v1"
	// - A map with API path as key and overrides as value (extended syntax)
	Libraries []LibraryEntry `yaml:"libraries,omitempty"`
}

// LibraryEntry represents a single library entry in the configuration.
// It can be either a simple string (like "*") or a map with library name and config.
type LibraryEntry struct {
	// Simple contains the library name when using string syntax (e.g., "*")
	Simple string

	// Map contains the library name and configuration when using map syntax
	Map map[string]Library
}

// UnmarshalYAML implements custom unmarshaling for LibraryEntry.
func (e *LibraryEntry) UnmarshalYAML(node *yaml.Node) error {
	// Try to unmarshal as a string first
	var s string
	if err := node.Decode(&s); err == nil {
		e.Simple = s
		return nil
	}

	// If that fails, try to unmarshal as a map
	var m map[string]Library
	if err := node.Decode(&m); err != nil {
		return err
	}
	e.Map = m
	return nil
}

// MarshalYAML implements custom marshaling for LibraryEntry.
func (e LibraryEntry) MarshalYAML() (interface{}, error) {
	if e.Simple != "" {
		return e.Simple, nil
	}
	return e.Map, nil
}

// Sources contains references to external source repositories.
type Sources struct {
	// Googleapis is the googleapis source repository.
	Googleapis *Source `yaml:"googleapis,omitempty"`

	// Discovery is the discovery-artifact-manager source repository.
	Discovery *Source `yaml:"discovery,omitempty"`

	// Showcase is the gapic-showcase source repository.
	Showcase *Source `yaml:"showcase,omitempty"`

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

	// ExtractedName is the name of the extracted directory (if different from default).
	ExtractedName string `yaml:"extracted_name,omitempty"`

	// Subdir is the subdirectory within the extracted archive to use.
	Subdir string `yaml:"subdir,omitempty"`
}

// ConfigDefault contains default generation settings.
type ConfigDefault struct {
	// Output is the directory where generated code is written (relative to repository root).
	Output string `yaml:"output,omitempty"`

	Generate *ConfigGenerate `yaml:"generate,omitempty"`

	Release *ConfigRelease `yaml:"release,omitempty"`

	Rust *RustDefault `yaml:"rust,omitempty"`
}

type ConfigGenerate struct {
	// Generated all generates all client libraries with default configurations
	// for the language, unless otherwise specified.
	All bool `yaml:"all,omitempty"`

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
}

// ConfigRelease contains release configuration.
type ConfigRelease struct {
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
	// API specifies which googleapis API to generate from (for generated libraries).
	// Can be a string (protobuf API path) or an APIObject (for discovery APIs).
	// If both API and APIs are empty, this is a handwritten library.
	API string `yaml:"api,omitempty"`

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

	// Overrides for derived fields
	LaunchStage  string   `yaml:"launch_stage,omitempty"`
	Destinations []string `yaml:"destinations,omitempty"`

	// GRPCServiceConfig is the gRPC service config JSON file path.
	GRPCServiceConfig string `yaml:"grpc_service_config,omitempty"`

	// Language-specific library configurations
	Rust   *RustCrate     `yaml:"rust,omitempty"`
	Python *PythonPackage `yaml:"python,omitempty"`
	Go     *GoModule      `yaml:"go,omitempty"`
	Dart   *DartPackage   `yaml:"dart,omitempty"`
}

// LibraryGenerate contains per-library generate configuration.
type LibraryGenerate struct {
	// Disabled prevents library generation.
	Disabled bool `yaml:"disabled,omitempty"`
}

// LibraryRelease contains per-library release configuration.
type LibraryRelease struct {
	// Disabled prevents library release and publish.
	Disabled bool `yaml:"disabled,omitempty"`
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

// Set sets a configuration key to the specified value.
func (c *Config) Set(key, value string) error {
	switch key {
	case "release.tag_format":
		if c.Release == nil {
			c.Release = &ConfigRelease{}
		}
		c.Release.TagFormat = value
	case "generate.output":
		c.Defaults.Output = value
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}

// Unset removes a configuration key.
func (c *Config) Unset(key string) error {
	switch key {
	case "release.tag_format":
		if c.Release != nil {
			c.Release.TagFormat = ""
		}
	case "generate.output":
		c.Defaults.Output = ""
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}
