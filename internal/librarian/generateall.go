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

	"github.com/googleapis/librarian/internal/config"
)

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

	// Get one_library_per mode
	if cfg.Default == nil || cfg.Default.Generate == nil || cfg.Default.Generate.OneLibraryPer == "" {
		return fmt.Errorf("one_library_per must be set in librarian.yaml under default.generate.one_library_per")
	}
	oneLibraryPer := cfg.Default.Generate.OneLibraryPer

	// Discover all libraries (grouped by one_library_per mode)
	libraries, err := config.DiscoverLibraries(googleapisDir, cfg.Language, oneLibraryPer)
	if err != nil {
		return fmt.Errorf("failed to discover libraries: %w", err)
	}

	fmt.Printf("Discovered %d libraries\n", len(libraries))

	// Generate each library
	for _, lib := range libraries {
		// Generate each API in the library
		for apiPath, serviceConfigPath := range lib.APIServiceConfigs {
			if err := generateLibraryForAPI(ctx, cfg, googleapisDir, apiPath, serviceConfigPath, false); err != nil {
				fmt.Printf("  ✗ %s: %v\n", apiPath, err)
				return err
			}
		}
	}
	return nil
}
