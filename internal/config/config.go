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
	Language string `yaml:"language,omitempty"`

	// Sources contains references to external source repositories.
	Sources Sources `yaml:"sources,omitempty"`

	// Generate contains generation configuration.
	Generate *Generate `yaml:"generate,omitempty"`

	// Release contains release configuration.
	Release *Release `yaml:"release,omitempty"`
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
