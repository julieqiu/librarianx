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

	// Sources contains references to external source repositories.
	Sources Sources `yaml:"sources,omitempty"`

	// Generate contains generation configuration.
	Generate *Generate `yaml:"generate,omitempty"`

	// Release contains release configuration.
	Release *Release `yaml:"release,omitempty"`

	// Libraries contains the list of library libraries.
	Libraries []Library `yaml:"libraries,omitempty"`
}

// Sources contains references to external source repositories.
type Sources struct {
	// Googleapis is the googleapis source repository.
	Googleapis *Source `yaml:"googleapis,omitempty"`
}

// Source represents an external source repository.
type Source struct {
	// URL is the download URL for the source tarball.
	URL string `yaml:"url"`

	// SHA256 is the hash for integrity verification.
	SHA256 string `yaml:"sha256"`
}

// Generate contains generation configuration.
type Generate struct {
	// Output is the directory where generated code is written (relative to repository root).
	Output string `yaml:"output,omitempty"`
}

// Release contains release configuration.
type Release struct {
	// TagFormat is the template for git tags (e.g., '{id}/v{version}').
	// Supported placeholders: {id}, {name}, {version}
	TagFormat string `yaml:"tag_format,omitempty"`
}

// Library represents an library.
type Library struct {
	// Name is the library name (e.g., "secretmanager").
	Name string `yaml:"name"`

	// Version is the version of the library.
	Version string `yaml:"version,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear int `yaml:"copyright_year,omitempty"`

	// APIs is the list of googleapis paths for generated librarys.
	APIs []API `yaml:"apis,omitempty"`

	// Location is the explicit filesystem path (optional).
	// If not set and apis is present, computed from generate.output template.
	Location string `yaml:"location,omitempty"`

	// ModulePathVersion is the major version for the overall module, e.g. "v2"
	// to create a module path of cloud.google.com/go/{Name}/v2
	//
	// The data for this field comes from .librarian/generator-input/repo-config.yaml.
	ModulePathVersion string `yaml:"module_path_version,omitempty"`

	// DeleteGenerationOutputPaths specifies paths (files or directories) to
	// be deleted from the output directory at the end of generation. This is for files
	// which it is difficult to prevent from being generated, but which shouldn't appear
	// in the repo.
	//
	// The data for this field comes from .librarian/generator-input/repo-config.yaml.
	DeleteGenerationOutputPaths []string `yaml:"delete_generation_output_paths,omitempty"`
}

// API corresponds to a single API definition within a librarian request/response.
type API struct {
	// Path is the directory to the API definition in protos, within googleapis (e.g. google/cloud/functions/v2)
	Path string `yaml:"path,omitempty"`

	// Status indicates whether this API is new or existing (used during configure).
	Status string `yaml:"status,omitempty"`

	// ServiceConfig is the name of the service config file, relative to Path.
	ServiceConfig string `yaml:"service_config,omitempty"`

	// ProtoPackage is the protobuf package, when it doesn't match the Path
	// (after replacing slash with period).
	//
	// The data for this field comes from .librarian/generator-input/repo-config.yaml.
	ProtoPackage string `yaml:"proto_package,omitempty"`

	// ClientDirectory is the directory containing the client code, relative to the module root.
	// (This is currently only used to find snippet metadata files.)
	//
	// The data for this field comes from .librarian/generator-input/repo-config.yaml.
	ClientDirectory string `yaml:"client_directory,omitempty"`

	// DisableGAPIC is a flag to disable GAPIC generation for an API, overriding
	// settings from the BUILD.bazel file.
	//
	// The data for this field comes from .librarian/generator-input/repo-config.yaml.
	DisableGAPIC bool `yaml:"disable_gapic,omitempty"`

	// NestedProtos lists any nested proto files (under Path) that should be included
	// in generation. Currently, only proto files *directly* under Path (as opposed to
	// in subdirectories) are passed to protoc; this setting allows selected nested
	// protos to be included as well.
	//
	// The data for this field comes from .librarian/generator-input/repo-config.yaml.
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

// Add adds an library to the config.
// If location is provided, creates a handwritten library with explicit location.
// Otherwise, creates a generated library with the given APIs.
func (c *Config) Add(name string, apis []string, location string) error {
	if name == "" {
		return fmt.Errorf("library name cannot be empty")
	}

	// Handwritten library with explicit location
	if location != "" {
		// Check if library with same name already exists
		for _, ed := range c.Libraries {
			if ed.Name == name {
				return fmt.Errorf("library %q already exists", name)
			}
		}

		c.Libraries = append(c.Libraries, Library{
			Name:     name,
			Location: location,
		})
		return nil
	}

	// Generated library with APIs
	if len(apis) == 0 {
		return fmt.Errorf("library must have at least one API or a location")
	}

	var apiStructs []API
	for _, api := range apis {
		apiStructs = append(apiStructs, API{Path: api})
	}

	// Check if library with same name and apis already exists
	for _, ed := range c.Libraries {
		if ed.Name == name && apiPathsEqual(ed.APIs, apis) {
			return fmt.Errorf("library %q with apis %v already exists", name, apis)
		}
	}

	c.Libraries = append(c.Libraries, Library{
		Name: name,
		APIs: apiStructs,
	})

	return nil
}

// apiSliceEqual checks if two API slices are equal.
func apiSliceEqual(a, b []API) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Path != b[i].Path {
			return false
		}
	}
	return true
}

// apiPathsEqual checks if an API slice matches a string slice of paths.
func apiPathsEqual(apis []API, paths []string) bool {
	if len(apis) != len(paths) {
		return false
	}
	for i := range apis {
		if apis[i].Path != paths[i] {
			return false
		}
	}
	return true
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
		if len(e.APIs) != 1 {
			return "", fmt.Errorf("template uses {api.path} but library %q has %d APIs (expected exactly 1)", e.Name, len(e.APIs))
		}
		result = strings.ReplaceAll(result, "{api.path}", e.APIs[0].Path)
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

// GetModulePath returns the module path for the library, applying
// any configured version.
func (l *Library) GetModulePath() string {
	prefix := "cloud.google.com/go/" + l.Name
	if l.ModulePathVersion != "" {
		return prefix + "/" + l.ModulePathVersion
	}

	// No override: assume implicit v1.
	return prefix
}

// GetProtoPackage returns the protobuf package for the API config,
// which is derived from the path unless overridden.
func (a *API) GetProtoPackage() string {
	if a.ProtoPackage != "" {
		return a.ProtoPackage
	}

	// No override: derive the value.
	return strings.ReplaceAll(a.Path, "/", ".")
}

// GetClientDirectory returns the directory for the clients of this
// API, relative to the module root.
func (a *API) GetClientDirectory(libraryName string) (string, error) {
	if a.ClientDirectory != "" {
		return a.ClientDirectory, nil
	}

	// No override: derive the value.
	startOfModuleName := strings.Index(a.Path, libraryName+"/")
	if startOfModuleName == -1 {
		return "", fmt.Errorf("librariangen: unexpected API path format: %s", a.Path)
	}

	// google/spanner/v1 => ["google", "spanner", "v1"]
	// google/spanner/admin/instance/v1 => ["google", "spanner", "admin", "instance", "v1"]
	parts := strings.Split(a.Path, "/")
	moduleIndex := -1
	for i, p := range parts {
		if p == libraryName {
			moduleIndex = i
			break
		}
	}
	if moduleIndex == -1 {
		return "", fmt.Errorf("librariangen: module name '%s' not found in API path '%s'", libraryName, a.Path)
	}

	// Remove everything up to and include the module name.
	// google/spanner/v1 => ["v1"]
	// google/spanner/admin/instance/v1 => ["admin", "instance", "v1"]
	parts = parts[moduleIndex+1:]
	parts[len(parts)-1] = "api" + parts[len(parts)-1]
	return strings.Join(parts, "/"), nil
}

// HasDisableGAPIC returns a value saying whether GAPIC generation is explicitly
// disabled for this module.
func (a *API) HasDisableGAPIC() bool {
	return a.DisableGAPIC
}
