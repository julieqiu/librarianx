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
)

// FindLibraryByName finds a library by name in the config and prepares it for generation.
// Returns the library with service configs populated, ready for generation.
// If not found in config, tries to create a library from the API path (fallback for version-mode languages).
func FindLibraryByName(cfg *Config, name, googleapisDir string) (*Library, error) {
	// Find library by name in config
	library := findLibraryInConfig(cfg, name)
	if library == nil {
		var err error
		// Create library from name
		library, err = createLibraryFromName(cfg, name, googleapisDir)
		if err != nil {
			return nil, err
		}
	}

	// Ensure library has APIs configured (may derive from name for version-mode)
	if err := ensureLibraryHasAPIs(cfg, library, name); err != nil {
		return nil, err
	}

	// Populate service configs and apply all configuration
	if err := prepareLibrary(cfg, library, googleapisDir); err != nil {
		return nil, err
	}

	return library, nil
}

// findLibraryInConfig searches for a library by name in the config.
func findLibraryInConfig(cfg *Config, name string) *Library {
	for _, lib := range cfg.Libraries {
		if lib.Name == name {
			return lib
		}
	}
	return nil
}

// createLibraryFromName creates a new library from a library name (fallback path).
func createLibraryFromName(cfg *Config, name, googleapisDir string) (*Library, error) {
	// Get one_library_per mode to derive API path
	if cfg.Default == nil || cfg.Default.Generate == nil || cfg.Default.Generate.OneLibraryPer == "" {
		return nil, fmt.Errorf("library %q not found in config and one_library_per not set", name)
	}

	// Derive API path from library name
	apiPath, err := DeriveAPIPath(cfg.Default.Generate.OneLibraryPer, name)
	if err != nil {
		return nil, fmt.Errorf("failed to derive API path from name %q: %w", name, err)
	}

	// Create library from API path
	return createLibraryFromAPIPath(name, apiPath, googleapisDir)
}

// ensureLibraryHasAPIs ensures the library has APIs configured, deriving them if needed.
func ensureLibraryHasAPIs(cfg *Config, library *Library, name string) error {
	apiPaths := GetLibraryAPIs(library)
	if len(apiPaths) > 0 {
		return nil
	}

	// For version-mode languages (like Rust), derive API from library name
	if cfg.Default == nil || cfg.Default.Generate == nil || cfg.Default.Generate.OneLibraryPer != "channel" {
		return fmt.Errorf("library %q has no APIs configured", name)
	}

	apiPath, err := DeriveAPIPath("version", name)
	if err != nil {
		return fmt.Errorf("library %q has no APIs configured and failed to derive API path: %w", name, err)
	}
	library.Channel = apiPath
	return nil
}

// prepareLibrary populates service configs, filters APIs, and applies defaults.
func prepareLibrary(cfg *Config, library *Library, googleapisDir string) error {
	// Populate service configs for the library
	if err := populateServiceConfigs(library, googleapisDir); err != nil {
		return err
	}

	// Filter out excluded APIs
	filteredServiceConfigs := make(map[string]string)
	for apiPath, serviceConfigPath := range library.APIServiceConfigs {
		excluded, err := IsAPIExcluded(cfg.Language, apiPath)
		if err != nil {
			return fmt.Errorf("failed to check if API is excluded: %w", err)
		}
		if excluded {
			continue
		}
		filteredServiceConfigs[apiPath] = serviceConfigPath
	}

	if len(filteredServiceConfigs) == 0 {
		return fmt.Errorf("library %q has no APIs after filtering excluded APIs", library.Name)
	}

	// Update library with filtered service configs
	library.APIServiceConfigs = filteredServiceConfigs

	// Apply version from versions map if not already set
	if library.Version == "" && cfg.Versions != nil {
		if version, ok := cfg.Versions[library.Name]; ok {
			library.Version = version
		}
	}

	if cfg.Default != nil && cfg.Default.Generate != nil {
		if library.Transport == "" {
			library.Transport = cfg.Default.Generate.Transport
		}
		if library.ReleaseLevel == "" {
			library.ReleaseLevel = cfg.Default.Generate.ReleaseLevel
		}
		if library.RestNumericEnums == nil {
			b := cfg.Default.Generate.RestNumericEnums
			library.RestNumericEnums = &b
		}
	}

	switch cfg.Language {
	case "rust":
		if cfg.Default != nil && cfg.Default.Rust != nil {
			if library.Rust == nil {
				library.Rust = &RustCrate{}
			}
			if len(library.Rust.DisabledRustdocWarnings) == 0 {
				library.Rust.DisabledRustdocWarnings = cfg.Default.Rust.DisabledRustdocWarnings
			}
			// Merge default package dependencies with library-specific ones
			library.Rust.PackageDependencies = mergePackageDependencies(cfg.Default.Rust.PackageDependencies, library.Rust.PackageDependencies)
		}
	}
	return nil
}

// createLibraryFromAPIPath creates a library from an API path (fallback when library not in config).
// This is used for version-mode languages where libraries are named after their API paths.
func createLibraryFromAPIPath(name, apiPath, googleapisDir string) (*Library, error) {
	// Validate API directory exists in googleapis
	apiDir := filepath.Join(googleapisDir, apiPath)
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("library %q not found in config and API path %q not found in googleapis", name, apiPath)
	} else if err != nil {
		return nil, fmt.Errorf("failed to check API directory: %w", err)
	}

	// Create minimal library config
	library := &Library{
		Name:    name,
		Channel: apiPath,
	}

	// Populate service configs
	if err := populateServiceConfigs(library, googleapisDir); err != nil {
		return nil, err
	}

	return library, nil
}

// GetLibraryAPIs returns all API paths for a library.
// Handles both single-API (library.API) and multi-API (library.APIs) libraries.
func GetLibraryAPIs(lib *Library) []string {
	if len(lib.Channels) > 0 {
		return lib.Channels
	}
	if lib.Channel != "" {
		return []string{lib.Channel}
	}
	return nil
}

// mergePackageDependencies merges default dependencies with library-specific ones.
// Library-specific dependencies override defaults.
func mergePackageDependencies(defaults []*RustPackageDependency, librarySpecific []RustPackageDependency) []RustPackageDependency {
	// Create a map of library-specific dependencies by name for quick lookup
	libMap := make(map[string]RustPackageDependency)
	for _, dep := range librarySpecific {
		libMap[dep.Name] = dep
	}

	// Start with all default dependencies (convert from pointers)
	result := make([]RustPackageDependency, 0, len(defaults)+len(librarySpecific))
	for _, dep := range defaults {
		if dep != nil {
			result = append(result, *dep)
		}
	}

	// Override with library-specific dependencies and track which names we've seen
	seenNames := make(map[string]bool)
	for i, dep := range result {
		if override, ok := libMap[dep.Name]; ok {
			result[i] = override
			seenNames[override.Name] = true
		} else {
			seenNames[dep.Name] = true
		}
	}

	// Add any library-specific dependencies that weren't in defaults
	for _, dep := range librarySpecific {
		if !seenNames[dep.Name] {
			result = append(result, dep)
		}
	}

	return result
}
