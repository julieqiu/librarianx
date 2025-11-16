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

	"gopkg.in/yaml.v3"
)

//go:embed service_config_overrides.yaml
var serviceConfigOverridesYAML []byte

// ServiceConfigOverrides contains overrides for googleapis API discovery.
type ServiceConfigOverrides struct {
	// ServiceConfigs maps API paths to their service YAML filenames.
	ServiceConfigs map[string]string `yaml:"service_configs"`

	// ExcludedAPIs contains language-specific API exclusions.
	ExcludedAPIs struct {
		// All lists APIs excluded from all languages.
		All []string `yaml:"all"`
		// Rust lists APIs excluded from Rust only.
		Rust []string `yaml:"rust"`
	} `yaml:"excluded_apis"`
}

// ReadServiceConfigOverrides reads the embedded service_config_overrides.yaml file.
func ReadServiceConfigOverrides() (*ServiceConfigOverrides, error) {
	var overrides ServiceConfigOverrides
	if err := yaml.Unmarshal(serviceConfigOverridesYAML, &overrides); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service config overrides: %w", err)
	}
	return &overrides, nil
}

// IsExcluded returns true if the given API path matches any exclusion pattern for the specified language.
// Checks both global "all" exclusions and language-specific exclusions.
func (o *ServiceConfigOverrides) IsExcluded(language, apiPath string) bool {
	// Check "all" exclusions first (applies to all languages)
	for _, pattern := range o.ExcludedAPIs.All {
		if matchGlobPattern(pattern, apiPath) {
			return true
		}
	}

	// Check language-specific exclusions
	var languageExclusions []string
	switch language {
	case "rust":
		languageExclusions = o.ExcludedAPIs.Rust
	// Add more languages here as needed
	// case "python":
	//     languageExclusions = o.ExcludedAPIs.Python
	}

	for _, pattern := range languageExclusions {
		if matchGlobPattern(pattern, apiPath) {
			return true
		}
	}

	return false
}

// GetServiceConfig returns the service config filename for the given API path.
// Returns empty string if no override is configured.
func (o *ServiceConfigOverrides) GetServiceConfig(apiPath string) string {
	return o.ServiceConfigs[apiPath]
}

// matchGlobPattern checks if a path matches a directory pattern.
// Treats all patterns as directory prefix matches.
// Pattern "google/ads" matches "google/ads" and all subdirectories like "google/ads/v1".
func matchGlobPattern(pattern, path string) bool {
	// Strip trailing / if present for backwards compatibility
	if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
		pattern = pattern[:len(pattern)-1]
	}

	// Exact match
	if pattern == path {
		return true
	}

	// Prefix match: path must start with pattern + "/"
	return len(path) > len(pattern) && path[:len(pattern)] == pattern && path[len(pattern)] == '/'
}
