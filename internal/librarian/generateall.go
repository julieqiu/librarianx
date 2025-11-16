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

package librarian

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language"
)

// discoveredAPI represents a discovered API in the googleapis repository.
type discoveredAPI struct {
	// path is the relative path from googleapis root (e.g., "google/cloud/secretmanager/v1").
	path string

	// serviceConfigPath is the absolute path to the service YAML file.
	serviceConfigPath string
}

// runGenerateAll generates all APIs found in the googleapis repository.
func runGenerateAll(ctx context.Context) error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
		return fmt.Errorf("no googleapis source configured in %s", configPath)
	}

	commit := cfg.Sources.Googleapis.Commit
	if commit == "" {
		return fmt.Errorf("no commit specified for googleapis source in %s", configPath)
	}

	googleapisDir, err := googleapisDir(commit)
	if err != nil {
		return err
	}

	// Read service config overrides
	overrides, err := config.ReadServiceConfigOverrides()
	if err != nil {
		return fmt.Errorf("failed to read service config overrides: %w", err)
	}

	// Discover all APIs
	apis, err := discoverAPIs(googleapisDir, overrides)
	if err != nil {
		return fmt.Errorf("failed to discover APIs: %w", err)
	}

	fmt.Printf("Discovered %d APIs\n", len(apis))

	// Convert to language.APIToGenerate
	var apisToGenerate []language.APIToGenerate
	for _, api := range apis {
		apisToGenerate = append(apisToGenerate, language.APIToGenerate{
			Path:              api.path,
			ServiceConfigPath: api.serviceConfigPath,
		})
	}

	// Generate all APIs using language package
	defaultOutput := "src/generated"
	if cfg.Default.Output != "" {
		defaultOutput = cfg.Default.Output
	}

	// Generate with progress reporting
	for _, api := range apisToGenerate {
		library := &config.Library{
			API:  api.Path,
			Rust: &config.RustCrate{},
		}
		if err := language.Generate(ctx, library, googleapisDir, api.ServiceConfigPath, defaultOutput); err != nil {
			fmt.Printf("  ✗ %s: %v\n", api.Path, err)
			return err
		}
		fmt.Printf("  ✓ %s\n", api.Path)
	}
	return nil
}

// discoverAPIs walks the googleapis directory tree and returns all discovered APIs.
func discoverAPIs(googleapisDir string, overrides *config.ServiceConfigOverrides) ([]*discoveredAPI, error) {
	var apis []*discoveredAPI

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
		if overrides != nil && overrides.IsExcluded(apiPath) {
			return nil
		}

		// Try to find service config
		serviceConfigPath, err := findServiceConfigForAPI(googleapisDir, apiPath, overrides)
		if err != nil {
			// Skip APIs without service config
			return nil
		}

		apis = append(apis, &discoveredAPI{
			path:              apiPath,
			serviceConfigPath: serviceConfigPath,
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
func findServiceConfigForAPI(googleapisDir, apiPath string, overrides *config.ServiceConfigOverrides) (string, error) {
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
