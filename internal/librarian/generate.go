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
			return runGenerate(ctx, name, false)
		},
	}
}

func runGenerate(ctx context.Context, name string, newLibrary bool) error {
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

	// Find or create library by name
	library, err := config.FindLibraryByName(cfg, name, googleapisDir)
	if err != nil {
		return err
	}

	// Check one_library_per mode
	if cfg.Default == nil || cfg.Default.Generate == nil || cfg.Default.Generate.OneLibraryPer == "" {
		return fmt.Errorf("one_library_per must be set in librarian.yaml under default.generate.one_library_per")
	}

	return generateLibrary(ctx, cfg, googleapisDir, library, cfg.Default.Generate.OneLibraryPer, newLibrary)
}

// generateLibrary prepares and generates a library.
func generateLibrary(ctx context.Context, cfg *config.Config, googleapisDir string, library *config.Library, oneLibraryPer string, newLibrary bool) error {
	// Check if generation is disabled
	if library.Generate != nil && library.Generate.Disabled {
		fmt.Printf("  ⊘ %s (generation disabled)\n", library.Name)
		return nil
	}

	// Generate the library
	if newLibrary {
		if err := language.Create(ctx, cfg.Language, cfg.Repo, library, cfg.Default, googleapisDir, "", cfg.Default.Output); err != nil {
			return err
		}
	} else {
		if err := language.Generate(ctx, oneLibraryPer, cfg.Language, cfg.Repo, library, cfg.Default, googleapisDir); err != nil {
			return err
		}
	}

	// Print success for each API
	for apiPath := range library.APIServiceConfigs {
		fmt.Printf("  ✓ %s\n", apiPath)
	}
	return nil
}
