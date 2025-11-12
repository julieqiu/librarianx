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

	// Defaults contains default generation settings.
	Defaults *Defaults `yaml:"defaults,omitempty"`

	// Generate contains generation configuration.
	Generate *Generate `yaml:"generate,omitempty"`

	// Release contains release configuration.
	Release *Release `yaml:"release,omitempty"`

	// Libraries contains the list of library libraries.
	Libraries []Library `yaml:"libraries,omitempty"`
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
}

// Source represents an external source repository.
type Source struct {
	// URL is the download URL for the source tarball.
	URL string `yaml:"url"`

	// SHA256 is the hash for integrity verification.
	SHA256 string `yaml:"sha256"`
}

// Defaults contains default generation settings.
type Defaults struct {
	// GeneratedDir is the directory where generated code is written (relative to repository root).
	GeneratedDir string `yaml:"generated_dir,omitempty"`

	// Transport is the default transport protocol (e.g., "grpc+rest", "grpc").
	Transport string `yaml:"transport,omitempty"`

	// RestNumericEnums indicates whether to use numeric enums in REST.
	RestNumericEnums bool `yaml:"rest_numeric_enums,omitempty"`

	// ReleaseLevel is the default release level ("stable" or "preview").
	ReleaseLevel string `yaml:"release_level,omitempty"`
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

	// Version is the current released version.
	Version string `yaml:"version,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear int `yaml:"copyright_year,omitempty"`

	// Generate contains generation configuration for this library.
	Generate *LibraryGenerate `yaml:"generate,omitempty"`

	// Apis is the list of googleapis paths for generated librarys.
	Apis []string `yaml:"apis,omitempty"`

	// Location is the explicit filesystem path (optional).
	// If not set and apis is present, computed from generate.output template.
	Location string `yaml:"location,omitempty"`
}

// LibraryGenerate contains generation configuration for a library.
type LibraryGenerate struct {
	// APIs is the list of API configurations.
	APIs []API `yaml:"apis,omitempty"`

	// Keep is the list of files/directories not overwritten during generation.
	Keep []string `yaml:"keep,omitempty"`

	// Remove is the list of files/directories deleted after generation.
	Remove []string `yaml:"remove,omitempty"`
}

// API represents an API configuration.
type API struct {
	// Path is the API path relative to googleapis root (e.g., "google/cloud/secretmanager/v1").
	Path string `yaml:"path"`
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

	// Check if library with same name and apis already exists
	for _, ed := range c.Libraries {
		if ed.Name == name && stringSliceEqual(ed.Apis, apis) {
			return fmt.Errorf("library %q with apis %v already exists", name, apis)
		}
	}

	c.Libraries = append(c.Libraries, Library{
		Name: name,
		Apis: apis,
	})

	return nil
}

// stringSliceEqual checks if two string slices are equal.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
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
