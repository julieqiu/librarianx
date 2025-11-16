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
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed service_config_overrides.yaml
var serviceConfigOverridesYAML []byte

// ServiceConfigOverrides contains overrides for googleapis API discovery.
type ServiceConfigOverrides struct {
	// ServiceConfigs maps API paths to their service YAML filenames.
	ServiceConfigs map[string]string `yaml:"service_configs"`

	// ExcludeAPIs lists API path patterns to exclude from wildcard discovery.
	ExcludeAPIs []string `yaml:"exclude_apis"`
}

// ReadServiceConfigOverrides reads the embedded service_config_overrides.yaml file.
func ReadServiceConfigOverrides() (*ServiceConfigOverrides, error) {
	var overrides ServiceConfigOverrides
	if err := yaml.Unmarshal(serviceConfigOverridesYAML, &overrides); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service config overrides: %w", err)
	}
	return &overrides, nil
}

// IsExcluded returns true if the given API path matches any exclusion pattern.
func (o *ServiceConfigOverrides) IsExcluded(apiPath string) bool {
	for _, pattern := range o.ExcludeAPIs {
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

// matchGlobPattern checks if a path matches a glob pattern.
// Supports * as wildcard for matching path segments.
func matchGlobPattern(pattern, path string) bool {
	// Use filepath.Match for simple glob matching
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}
