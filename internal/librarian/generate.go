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
	"github.com/urfave/cli/v3"
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate [--all] [library-name]",
		Description: `Generate a client library from googleapis.

For Python (service-level bundling):
  librarian generate google-cloud-secretmanager

For Rust (version-level libraries):
  librarian generate google-cloud-secretmanager-v1

Generate all APIs:
  librarian generate --all`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "generate all discovered APIs",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Bool("all") {
				return runGenerateAll(ctx)
			}
			if cmd.NArg() < 1 {
				return fmt.Errorf("generate requires a library name argument or --all flag")
			}
			name := cmd.Args().Get(0)
			return runGenerate(ctx, name)
		},
	}
}

func runGenerate(ctx context.Context, name string) error {
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

	// First try to find the library by name in config
	var library *config.Library
	for _, lib := range cfg.Libraries {
		if lib.Name == name {
			library = lib
			break
		}
	}

	// If found, generate all its APIs
	if library != nil {
		apiPaths := library.APIs
		if len(apiPaths) == 0 && library.API != "" {
			apiPaths = []string{library.API}
		}

		if len(apiPaths) == 0 {
			return fmt.Errorf("library %q has no APIs configured", name)
		}

		// Read service config overrides
		overrides, err := config.ReadServiceConfigOverrides()
		if err != nil {
			return fmt.Errorf("failed to read service config overrides: %w", err)
		}

		// Generate each API
		for _, apiPath := range apiPaths {
			// Check if API is excluded
			if overrides.IsExcluded(cfg.Language, apiPath) {
				fmt.Printf("  ⊘ %s (excluded)\n", apiPath)
				continue
			}

			serviceConfigPath, err := findServiceConfigForAPI(googleapisDir, apiPath, overrides)
			if err != nil {
				return err
			}

			if err := generateLibraryForAPI(ctx, cfg, googleapisDir, apiPath, serviceConfigPath); err != nil {
				return err
			}
		}
		return nil
	}

	// Fallback: treat name as API path (for Rust version-level libraries)
	apiPath := strings.ReplaceAll(name, "-", "/")

	// Validate API directory exists
	apiDir := filepath.Join(googleapisDir, apiPath)
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return fmt.Errorf("library %q not found in config and API path %q not found in googleapis", name, apiPath)
	} else if err != nil {
		return err
	}

	// Read service config overrides
	overrides, err := config.ReadServiceConfigOverrides()
	if err != nil {
		return fmt.Errorf("failed to read service config overrides: %w", err)
	}

	// Check if API is excluded
	if overrides.IsExcluded(cfg.Language, apiPath) {
		return fmt.Errorf("API %q is excluded from generation", apiPath)
	}

	serviceConfigPath, err := findServiceConfigForAPI(googleapisDir, apiPath, overrides)
	if err != nil {
		return err
	}

	return generateLibraryForAPI(ctx, cfg, googleapisDir, apiPath, serviceConfigPath)
}

// generateLibraryForAPI generates a library for the given API path.
func generateLibraryForAPI(ctx context.Context, cfg *config.Config, googleapisDir, apiPath, serviceConfigPath string) error {
	// Check for name override first
	nameOverride := cfg.GetNameOverride(apiPath)

	// Derive the library name from API path (for matching libraries in generate.all mode)
	derivedName := deriveLibraryName(apiPath)

	var library *config.Library
	for _, lib := range cfg.Libraries {
		// Try matching in order of priority:
		// 1. If there's a name override, match by override name
		// 2. Match by explicit API field (for backward compatibility)
		// 3. Match in APIs array (for multi-version libraries)
		// 4. Match by derived name (for generate.all mode where api field is omitted)
		if nameOverride != "" && lib.Name == nameOverride {
			library = lib
			break
		} else if lib.API == apiPath {
			library = lib
			break
		} else if containsAPI(lib.APIs, apiPath) {
			library = lib
			break
		} else if lib.Name == derivedName {
			library = lib
			break
		}
	}

	if library == nil {
		if cfg.Default.Generate == nil || !cfg.Default.Generate.All {
			return fmt.Errorf("library %q not found in configuration and generate.all is false", apiPath)
		}

		library = &config.Library{
			API: apiPath,
		}
	}

	// Ensure API field is set (in case library was found by name override without explicit api field)
	if library.API == "" {
		library.API = apiPath
	}

	// Ensure name is set on the library
	if nameOverride != "" {
		library.Name = nameOverride
	} else if library.Name == "" {
		// Derive name from API path using config method
		library.Name = cfg.GetLibraryName(apiPath)
	}

	// Apply version from versions map if not already set in library
	if library.Version == "" && cfg.Versions != nil {
		libraryName := library.Name
		if libraryName == "" {
			libraryName = deriveLibraryName(library.API)
		}
		if version, ok := cfg.Versions[libraryName]; ok {
			library.Version = version
		}
	}

	applyDefaults(library, cfg.Default)

	// Check if generation is disabled for this library
	if library.Generate != nil && library.Generate.Disabled {
		fmt.Printf("  ⊘ %s (generation disabled)\n", apiPath)
		return nil
	}

	if err := language.Generate(ctx, cfg.Language, cfg.Repo, library, cfg.Default, googleapisDir, serviceConfigPath, cfg.Default.Output); err != nil {
		return err
	}

	fmt.Printf("  ✓ %s\n", apiPath)
	return nil
}

// deriveLibraryName derives a library name from an API path.
// For example: "google/api/cloudquotas/v1" -> "google-api-cloudquotas-v1".
func deriveLibraryName(apiPath string) string {
	return strings.ReplaceAll(apiPath, "/", "-")
}

// containsAPI checks if an API path is in the APIs slice.
func containsAPI(apis []string, apiPath string) bool {
	for _, api := range apis {
		if api == apiPath {
			return true
		}
	}
	return false
}

func applyDefaults(library *config.Library, defaults *config.Default) {
	if defaults.Generate != nil {
		if library.Transport == "" {
			library.Transport = defaults.Generate.Transport
		}
		if library.ReleaseLevel == "" {
			library.ReleaseLevel = defaults.Generate.ReleaseLevel
		}
		if library.RestNumericEnums == nil {
			b := defaults.Generate.RestNumericEnums
			library.RestNumericEnums = &b
		}
	}
}
