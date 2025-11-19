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
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverLibraries walks the googleapis directory tree and returns discovered libraries,
// grouped according to one_library_per mode.
func DiscoverLibraries(googleapisDir, lang, oneLibraryPer string) ([]*Library, error) {
	// Read service config overrides
	overrides, err := readServiceConfigOverrides()
	if err != nil {
		return nil, fmt.Errorf("failed to read service config overrides: %w", err)
	}

	// Discover all individual versioned APIs
	apis, err := discoverAPIs(googleapisDir, lang, overrides)
	if err != nil {
		return nil, err
	}

	// Group APIs by library name according to one_library_per mode
	libMap := make(map[string]*Library)

	for _, api := range apis {
		// Derive library name for this API
		libraryName, err := DeriveLibraryName(oneLibraryPer, api.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to derive library name for %s: %w", api.Path, err)
		}

		// Get or create library entry
		lib, exists := libMap[libraryName]
		if !exists {
			lib = &Library{
				Name:              libraryName,
				APIServiceConfigs: make(map[string]string),
			}
			libMap[libraryName] = lib
		}

		// Add API to library
		if oneLibraryPer == "channel" {
			// Version mode: one API per library
			lib.Channel = api.Path
		} else {
			// API mode: multiple APIs per library
			lib.Channels = append(lib.Channels, api.Path)
		}

		// Store service config for this API
		lib.APIServiceConfigs[api.Path] = api.ServiceConfigPath
	}

	// Convert map to sorted slice
	var libraries []*Library
	for _, lib := range libMap {
		// Sort APIs within each library for consistent output
		if len(lib.Channels) > 1 {
			sort.Strings(lib.Channels)
		}
		libraries = append(libraries, lib)
	}

	// Sort libraries by name
	sort.Slice(libraries, func(i, j int) bool {
		return libraries[i].Name < libraries[j].Name
	})

	return libraries, nil
}

// discoveredAPI represents a single versioned API discovered in googleapis.
type discoveredAPI struct {
	// Path is the relative path from googleapis root (e.g., "google/cloud/secretmanager/v1").
	Path string

	// ServiceConfigPath is the absolute path to the service YAML file.
	ServiceConfigPath string
}

// discoverAPIs walks the googleapis directory tree and returns all discovered versioned APIs.
func discoverAPIs(googleapisDir, lang string, overrides *ServiceConfigOverrides) ([]discoveredAPI, error) {
	var apis []discoveredAPI

	err := filepath.Walk(googleapisDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Get relative path from googleapis root
		relPath, err := filepath.Rel(googleapisDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		// Normalize path separators to forward slashes
		apiPath := filepath.ToSlash(relPath)

		// Check if this looks like a versioned API directory
		if !isVersionedAPI(apiPath) {
			return nil
		}

		// Check if excluded
		if overrides != nil && overrides.IsExcluded(lang, apiPath) {
			return nil
		}

		// Try to find service config
		serviceConfigPath, err := findServiceConfigForAPI(googleapisDir, apiPath, overrides)
		if err != nil {
			// Skip APIs without service config
			return nil
		}

		apis = append(apis, discoveredAPI{
			Path:              apiPath,
			ServiceConfigPath: serviceConfigPath,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk googleapis directory: %w", err)
	}

	return apis, nil
}

// isVersionedAPI checks if a path looks like a versioned API directory.
// Examples: google/cloud/secretmanager/v1, google/spanner/admin/database/v1.
func isVersionedAPI(apiPath string) bool {
	parts := strings.Split(apiPath, "/")
	if len(parts) < 2 {
		return false
	}

	lastPart := parts[len(parts)-1]

	// Check if last part looks like a version (v1, v2, v1alpha, v1beta, etc.)
	if len(lastPart) < 2 {
		return false
	}

	if lastPart[0] != 'v' {
		return false
	}

	// After 'v', should have at least one digit or word like "alpha", "beta"
	rest := lastPart[1:]
	if len(rest) == 0 {
		return false
	}

	// Simple check: starts with digit or contains "alpha" or "beta"
	firstChar := rest[0]
	isVersion := (firstChar >= '0' && firstChar <= '9') ||
		strings.Contains(rest, "alpha") ||
		strings.Contains(rest, "beta")

	return isVersion
}

// findServiceConfigForAPI finds the service config YAML file for an API.
func findServiceConfigForAPI(googleapisDir, apiPath string, overrides *ServiceConfigOverrides) (string, error) {
	// Check for override first
	if overrides != nil {
		if override := overrides.GetServiceConfig(apiPath); override != "" {
			// Override path is relative to the API directory
			parts := strings.Split(apiPath, "/")
			// Go up to the parent directory for the override
			parentDir := strings.Join(parts[:len(parts)-1], "/")
			configPath := filepath.Join(googleapisDir, parentDir, override)

			// Verify it exists
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
		}
	}

	// Auto-discovery: find service config using pattern
	parts := strings.Split(apiPath, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid API path: %q", apiPath)
	}

	version := parts[len(parts)-1]
	dir := filepath.Join(googleapisDir, apiPath)

	// Pattern: *_<version>.yaml (e.g., secretmanager_v1.yaml)
	pattern := filepath.Join(dir, "*_"+version+".yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	// Filter out _gapic.yaml files
	var configs []string
	for _, m := range matches {
		if !strings.HasSuffix(m, "_gapic.yaml") {
			configs = append(configs, m)
		}
	}

	if len(configs) == 0 {
		return "", fmt.Errorf("no service config found for %q", apiPath)
	}

	if len(configs) > 1 {
		return "", fmt.Errorf("multiple service configs found for %q: %v", apiPath, configs)
	}

	return configs[0], nil
}

// populateServiceConfigs populates the APIServiceConfigs field for a library
// by looking up service configs for all its APIs.
func populateServiceConfigs(lib *Library, googleapisDir string) error {
	// Read service config overrides
	overrides, err := readServiceConfigOverrides()
	if err != nil {
		return fmt.Errorf("failed to read service config overrides: %w", err)
	}

	if lib.APIServiceConfigs == nil {
		lib.APIServiceConfigs = make(map[string]string)
	}

	// Get all API paths for this library
	apiPaths := lib.Channels
	if len(apiPaths) == 0 && lib.Channel != "" {
		apiPaths = []string{lib.Channel}
	}

	// Look up service config for each API
	for _, apiPath := range apiPaths {
		serviceConfigPath, err := findServiceConfigForAPI(googleapisDir, apiPath, overrides)
		if err != nil {
			return fmt.Errorf("failed to find service config for %s: %w", apiPath, err)
		}
		lib.APIServiceConfigs[apiPath] = serviceConfigPath
	}

	return nil
}
