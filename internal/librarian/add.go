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
	"sort"

	"github.com/googleapis/librarian/internal/config"
)

// Add adds or updates library configuration in librarian.yaml.
// Returns an error if the library already exists in any section.
// The library parameter must have at minimum the Name field set.
// If API field is not set, it will be derived from Name.
// Modifies cfg in place.
func Add(ctx context.Context, cfg *config.Config, googleapisDir string, library *config.Library) error {
	if library.Name == "" {
		return fmt.Errorf("library name is required")
	}

	// Get one_library_per mode from config
	if cfg.Default == nil || cfg.Default.Generate == nil || cfg.Default.Generate.OneLibraryPer == "" {
		return fmt.Errorf("one_library_per must be set in librarian.yaml under default.generate.one_library_per")
	}
	oneLibraryPer := cfg.Default.Generate.OneLibraryPer

	// Determine API path
	apiPath := library.API
	if apiPath == "" {
		// Derive from name
		var err error
		apiPath, err = config.DeriveAPIPath(oneLibraryPer, library.Name)
		if err != nil {
			return fmt.Errorf("failed to derive API path from name %q: %w", library.Name, err)
		}
		library.API = apiPath
	}

	// Verify API path exists in googleapis
	apiDir := filepath.Join(googleapisDir, apiPath)
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return fmt.Errorf("API path %q not found in googleapis", apiPath)
	} else if err != nil {
		return fmt.Errorf("failed to check API path: %w", err)
	}

	// Check if library already exists
	if err := checkLibraryNotExists(cfg, library.Name, apiPath); err != nil {
		return err
	}

	// Derive standard name from API path
	derivedName, err := config.DeriveLibraryName(oneLibraryPer, apiPath)
	if err != nil {
		return fmt.Errorf("failed to derive library name: %w", err)
	}

	// Add to name_overrides if name differs from derived name
	if library.Name != derivedName {
		if cfg.NameOverrides == nil {
			cfg.NameOverrides = make(map[string]string)
		}
		cfg.NameOverrides[apiPath] = library.Name
	}

	// Add to versions if version provided
	if library.Version != "" {
		if cfg.Versions == nil {
			cfg.Versions = make(map[string]string)
		}
		cfg.Versions[library.Name] = library.Version
	}

	// Add to libraries if any config options provided
	needsLibraryEntry := library.CopyrightYear != "" ||
		(library.Generate != nil && library.Generate.Disabled) ||
		(library.Publish != nil && library.Publish.Disabled) ||
		(library.Rust != nil && (library.Rust.PerServiceFeatures || library.Rust.GenerateSetterSamples))

	if needsLibraryEntry {
		// Insert in alphabetical order
		cfg.Libraries = insertLibraryAlphabetically(cfg.Libraries, library)
	}

	return nil
}

// checkLibraryNotExists returns an error if the library already exists in any section.
func checkLibraryNotExists(cfg *config.Config, name, apiPath string) error {
	// Check versions
	if cfg.Versions != nil {
		if _, exists := cfg.Versions[name]; exists {
			return fmt.Errorf("library %q already exists in versions (use 'librarian remove' to remove it first)", name)
		}
	}

	// Check name_overrides
	if cfg.NameOverrides != nil {
		if existingName, exists := cfg.NameOverrides[apiPath]; exists {
			return fmt.Errorf("API path %q already has a name override to %q (use 'librarian remove' to remove it first)", apiPath, existingName)
		}
	}

	// Check libraries
	for _, lib := range cfg.Libraries {
		if lib.Name == name {
			return fmt.Errorf("library %q already exists in libraries (use 'librarian remove' to remove it first)", name)
		}
	}

	return nil
}

// insertLibraryAlphabetically inserts a library into the list in alphabetical order by name.
func insertLibraryAlphabetically(libraries []*config.Library, newLib *config.Library) []*config.Library {
	// Find insertion point
	idx := sort.Search(len(libraries), func(i int) bool {
		return libraries[i].Name >= newLib.Name
	})

	// Insert at idx
	libraries = append(libraries, nil)
	copy(libraries[idx+1:], libraries[idx:])
	libraries[idx] = newLib

	return libraries
}
