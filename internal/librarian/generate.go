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

Examples:
  librarian generate google-cloud-secretmanager-v1
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

	apiPath := strings.ReplaceAll(name, "-", "/")

	// Validate API directory exists
	apiDir := filepath.Join(googleapisDir, apiPath)
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return fmt.Errorf("API path %q not found in googleapis", apiPath)
	} else if err != nil {
		return err
	}

	// Read service config overrides
	overrides, err := config.ReadServiceConfigOverrides()
	if err != nil {
		return fmt.Errorf("failed to read service config overrides: %w", err)
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

	var library *config.Library
	for _, lib := range cfg.Libraries {
		// If there's a name override, look up by name
		// Otherwise, look up by API path (existing behavior)
		if nameOverride != "" && lib.Name == nameOverride {
			library = lib
			break
		} else if nameOverride == "" && lib.API == apiPath {
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

	// Ensure name override is set on the library
	if nameOverride != "" {
		library.Name = nameOverride
	}

	applyDefaults(library, cfg.Default)

	// Check if generation is disabled for this library
	if library.Generate != nil && library.Generate.Disabled {
		fmt.Printf("  ⊘ %s (generation disabled)\n", apiPath)
		return nil
	}

	return language.Generate(ctx, library, googleapisDir, serviceConfigPath, cfg.Default.Output)
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

	if defaults.Rust != nil {
		if library.Rust == nil {
			library.Rust = &config.RustCrate{}
		}
		if len(library.Rust.DisabledRustdocWarnings) == 0 {
			library.Rust.DisabledRustdocWarnings = defaults.Rust.DisabledRustdocWarnings
		}
		if len(library.Rust.PackageDependencies) == 0 {
			library.Rust.PackageDependencies = convertPackageDependencies(defaults.Rust.PackageDependencies)
		}
	}
}

func convertPackageDependencies(deps []*config.RustPackageDependency) []config.RustPackageDependency {
	result := make([]config.RustPackageDependency, len(deps))
	for i, dep := range deps {
		if dep != nil {
			result[i] = *dep
		}
	}
	return result
}
