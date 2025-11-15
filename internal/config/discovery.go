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
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed service_config_overrides.yaml
var serviceConfigOverridesFS embed.FS

// ServiceConfigOverrides contains the overrides from service_config_overrides.yaml.
type ServiceConfigOverrides struct {
	ServiceConfigs map[string]string `yaml:"service_configs"`
	ExcludeAPIs    []string          `yaml:"exclude_apis"`
}

// defaultExclusions caches the loaded exclusions from service_config_overrides.yaml.
var defaultExclusions []string

// loadServiceConfigOverrides loads the service_config_overrides.yaml file.
func loadServiceConfigOverrides() (*ServiceConfigOverrides, error) {
	data, err := serviceConfigOverridesFS.ReadFile("service_config_overrides.yaml")
	if err != nil {
		return nil, err
	}

	var overrides ServiceConfigOverrides
	if err := yaml.Unmarshal(data, &overrides); err != nil {
		return nil, err
	}

	return &overrides, nil
}

// getDefaultExclusions returns the default API exclusions from service_config_overrides.yaml.
// Results are cached after first load.
func getDefaultExclusions() []string {
	if defaultExclusions != nil {
		return defaultExclusions
	}

	overrides, err := loadServiceConfigOverrides()
	if err != nil {
		// If we can't load the file, return empty list (fail gracefully)
		return []string{}
	}

	defaultExclusions = overrides.ExcludeAPIs
	return defaultExclusions
}

// DiscoveredAPI represents a discovered API from the googleapis filesystem.
type DiscoveredAPI struct {
	// Path is the API path relative to googleapis root (e.g., "google/cloud/secretmanager/v1")
	Path string

	// Service is the service name (e.g., "secretmanager")
	Service string

	// Namespace is the namespace (e.g., "cloud")
	Namespace string

	// Version is the version (e.g., "v1", "v1beta1")
	Version string

	// HasBuildFile indicates whether a BUILD.bazel file was found
	HasBuildFile bool
}

// DiscoverAPIs scans a googleapis directory and discovers all APIs.
// It looks for directories matching version patterns (v1, v1alpha, v1beta, etc.)
// and checks for BUILD.bazel files to confirm they are APIs.
func DiscoverAPIs(googleapisRoot string) ([]*DiscoveredAPI, error) {
	var discovered []*DiscoveredAPI
	versionPattern := regexp.MustCompile(`^v\d+(alpha\d*|beta\d*)?$`)

	// Walk the googleapis/google directory
	googleDir := filepath.Join(googleapisRoot, "google")

	err := filepath.WalkDir(googleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !d.IsDir() {
			return nil
		}

		// Check if this directory name matches a version pattern
		dirName := d.Name()
		if !versionPattern.MatchString(dirName) {
			return nil
		}

		// Get the API path relative to googleapis root
		relPath, err := filepath.Rel(googleapisRoot, path)
		if err != nil {
			return err
		}

		// Normalize path separators to forward slashes
		apiPath := filepath.ToSlash(relPath)

		// Check if this directory contains a BUILD.bazel file
		buildFile := filepath.Join(path, "BUILD.bazel")
		hasBuildFile := false
		if info, err := os.Stat(buildFile); err == nil && !info.IsDir() {
			hasBuildFile = true
		}

		// Parse the API path to extract components
		service, namespace, version := parseAPIPathForDiscovery(apiPath)

		discovered = append(discovered, &DiscoveredAPI{
			Path:         apiPath,
			Service:      service,
			Namespace:    namespace,
			Version:      version,
			HasBuildFile: hasBuildFile,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by API path for deterministic results
	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].Path < discovered[j].Path
	})

	return discovered, nil
}

// parseAPIPathForDiscovery is similar to naming.ParseAPIPath but returns values
// suitable for discovery (keeps the path structure).
func parseAPIPathForDiscovery(apiPath string) (service, namespace, version string) {
	// Remove leading/trailing slashes
	apiPath = strings.Trim(apiPath, "/")

	// Split into parts
	parts := strings.Split(apiPath, "/")

	// Remove "google" prefix if present
	if len(parts) > 0 && parts[0] == "google" {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		return "", "", ""
	}

	// Extract version (last part)
	versionRegex := regexp.MustCompile(`^v\d+(alpha\d*|beta\d*)?$`)
	if len(parts) > 0 && versionRegex.MatchString(parts[len(parts)-1]) {
		version = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}

	// Extract namespace and service
	switch len(parts) {
	case 0:
		return "", "", version
	case 1:
		service = parts[0]
		return service, "", version
	default:
		namespace = parts[0]
		service = parts[len(parts)-1]
		return service, namespace, version
	}
}

// GroupByService groups discovered APIs by service for service-level packaging.
// Returns a map of service name to list of API paths.
func GroupByService(apis []*DiscoveredAPI) map[string][]string {
	groups := make(map[string][]string)

	for _, api := range apis {
		// Create a service key based on namespace and service
		var serviceKey string
		if api.Namespace != "" {
			serviceKey = api.Namespace + "/" + api.Service
		} else {
			serviceKey = api.Service
		}

		groups[serviceKey] = append(groups[serviceKey], api.Path)
	}

	return groups
}

// FilterDiscoveredAPIs filters discovered APIs based on the libraries configuration.
// It returns only APIs that are not explicitly configured in the libraries list
// and do not match any exclude patterns.
func (c *Config) FilterDiscoveredAPIs(discovered []*DiscoveredAPI) []*DiscoveredAPI {
	// Check if wildcard is enabled
	hasWildcard := false
	for _, entry := range c.Libraries {
		if entry.Name == "*" {
			hasWildcard = true
			break
		}
	}

	if !hasWildcard {
		return nil
	}

	// Build a set of explicitly configured API paths
	// We need to extract API paths from library configs
	configured := make(map[string]bool)
	for _, entry := range c.Libraries {
		if entry.Name != "*" && entry.Config != nil {
			// Extract API path from config
			if entry.Config.API != nil {
				if apiStr, ok := entry.Config.API.(string); ok {
					configured[apiStr] = true
				}
			}
			if len(entry.Config.APIs) > 0 {
				for _, api := range entry.Config.APIs {
					configured[api] = true
				}
			}
		}
	}

	// Get exclude patterns - start with defaults from service_config_overrides.yaml
	excludePatterns := getDefaultExclusions()

	// Add user-provided exclusions from librarian.yaml
	if c.Defaults != nil {
		excludePatterns = append(excludePatterns, c.Defaults.ExcludeAPIs...)
	}

	// Filter out configured APIs and excluded patterns
	var filtered []*DiscoveredAPI
	for _, api := range discovered {
		// Skip if explicitly configured
		if configured[api.Path] {
			continue
		}

		// Skip if matches any exclude pattern
		if matchesAnyPattern(api.Path, excludePatterns) {
			continue
		}

		filtered = append(filtered, api)
	}

	return filtered
}

// matchesAnyPattern checks if a path matches any of the given patterns.
// Patterns can use * as a wildcard (e.g., "google/ads/*").
func matchesAnyPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesPattern(path, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a path matches a pattern with * wildcards.
func matchesPattern(path, pattern string) bool {
	// Simple wildcard matching
	// Convert pattern to a simple check
	if !strings.Contains(pattern, "*") {
		// Exact match
		return path == pattern
	}

	// Handle suffix wildcard: "google/ads/*"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/")
	}

	// Handle prefix wildcard: "*/v1"
	if strings.HasPrefix(pattern, "*/") {
		suffix := strings.TrimPrefix(pattern, "*/")
		return strings.HasSuffix(path, "/"+suffix)
	}

	// Handle middle wildcard: "google/*/v1"
	// For simplicity, split on * and check parts
	parts := strings.Split(pattern, "*")
	if len(parts) != 2 {
		// Only support single * for now
		return false
	}

	return strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1])
}

// GetAllLibraries returns all libraries, combining explicitly configured ones
// with auto-discovered ones (if wildcard '*' is present).
func (c *Config) GetAllLibraries(googleapisRoot string) ([]LibraryEntry, error) {
	// Check if wildcard is present
	hasWildcard := false
	for _, entry := range c.Libraries {
		if entry.Name == "*" {
			hasWildcard = true
			break
		}
	}

	// If wildcard is not present, return only explicit libraries
	if !hasWildcard {
		libraries := make([]LibraryEntry, 0, len(c.Libraries))
		libraries = append(libraries, c.Libraries...)
		return libraries, nil
	}

	// Wildcard mode: start with non-wildcard entries
	libraries := make([]LibraryEntry, 0, len(c.Libraries))
	for _, entry := range c.Libraries {
		if entry.Name != "*" {
			libraries = append(libraries, entry)
		}
	}

	// Discover APIs
	discovered, err := DiscoverAPIs(googleapisRoot)
	if err != nil {
		return nil, err
	}

	// Filter out already configured APIs
	filtered := c.FilterDiscoveredAPIs(discovered)

	// Add discovered APIs as library entries
	// Derive library name and path from API path using language conventions
	packaging := c.GetOneLibraryPer()
	output := ""
	if c.Defaults != nil {
		output = c.Defaults.Output
	}
	for _, api := range filtered {
		name := DeriveLibraryName(api.Path, c.Language, packaging)
		path := deriveLibraryPath(api.Path, c.Language, output)

		libraries = append(libraries, LibraryEntry{
			Name: name,
			Config: &LibraryConfig{
				API:  api.Path,
				Path: path,
			},
		})
	}

	return libraries, nil
}

// deriveLibraryPath derives the output path for an auto-discovered library.
// For Rust: google/cloud/bigquery/v2 → src/generated/cloud/bigquery/v2/
// For other languages, the path is typically not set (uses default).
func deriveLibraryPath(apiPath, language, output string) string {
	// Only Rust needs custom path derivation for auto-discovered libraries
	if language != "rust" {
		return ""
	}

	// Remove "google/" prefix if present
	path := apiPath
	if strings.HasPrefix(path, "google/") {
		path = strings.TrimPrefix(path, "google/")
	}

	// Append to output directory
	if output != "" && !strings.HasSuffix(output, "/") {
		output = output + "/"
	}

	return output + path + "/"
}

// GetLibrariesForGeneration returns libraries grouped appropriately based on packaging strategy.
// For service-level packaging (Python/Go), it groups multiple versions of the same service.
// For version-level packaging (Rust/Dart), each version is a separate library.
func (c *Config) GetLibrariesForGeneration(googleapisRoot string) ([]*Library, error) {
	entries, err := c.GetAllLibraries(googleapisRoot)
	if err != nil {
		return nil, err
	}

	packaging := c.GetOneLibraryPer()

	// For version-level packaging, each entry becomes a separate library
	if packaging == "version" {
		var libraries []*Library
		for i := range entries {
			entry := &entries[i]
			libraries = append(libraries, entry.ToLibrary())
		}
		return libraries, nil
	}

	// For service-level packaging, group entries by service name
	serviceGroups := make(map[string]*Library)

	for i := range entries {
		entry := &entries[i]
		name := entry.Name

		if lib, exists := serviceGroups[name]; exists {
			// Add this API to existing service group
			if entry.Config != nil && entry.Config.API != nil {
				if apiStr, ok := entry.Config.API.(string); ok {
					lib.Apis = append(lib.Apis, apiStr)
				}
			}
			if entry.Config != nil && len(entry.Config.APIs) > 0 {
				lib.Apis = append(lib.Apis, entry.Config.APIs...)
			}
		} else {
			// Create new service group
			lib := entry.ToLibrary()
			serviceGroups[name] = lib
		}
	}

	// Convert map to slice
	var libraries []*Library
	for _, lib := range serviceGroups {
		libraries = append(libraries, lib)
	}

	return libraries, nil
}
