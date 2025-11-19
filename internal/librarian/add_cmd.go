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

	"github.com/julieqiu/librarianx/internal/config"
	"github.com/urfave/cli/v3"
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add a library to librarian.yaml",
		UsageText: "librarian add <library-name> [api-path] [flags]",
		Description: `Add a library to librarian.yaml configuration.

Examples:
  # Add library with auto-detected API path
  librarian add google-cloud-secretmanager-v1 --version=1.0.0

  # Add library with explicit API path (for name overrides)
  librarian add google-cloud-translation-v3 google/cloud/translate/v3 --version=1.0.0

  # Add library with custom configuration
  librarian add google-cloud-compute-v1 --version=1.0.0 --generate-disabled`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "version",
				Usage: "library version to add to versions:",
			},
			&cli.StringFlag{
				Name:  "copyright-year",
				Usage: "copyright year for the library",
			},
			&cli.BoolFlag{
				Name:  "generate-disabled",
				Usage: "mark generation as disabled",
			},
			&cli.BoolFlag{
				Name:  "publish-disabled",
				Usage: "mark publishing as disabled",
			},
			&cli.BoolFlag{
				Name:  "per-service-features",
				Usage: "enable per-service features (Rust only)",
			},
			&cli.BoolFlag{
				Name:  "generate-setter-samples",
				Usage: "enable setter sample generation (Rust only)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return fmt.Errorf("add requires a library name argument")
			}

			name := cmd.Args().Get(0)
			apiPath := ""
			if cmd.NArg() >= 2 {
				apiPath = cmd.Args().Get(1)
			}

			return runAdd(ctx, name, apiPath, cmd)
		},
	}
}

func runAdd(ctx context.Context, name, apiPath string, cmd *cli.Command) error {
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

	library := &config.Library{
		Name: name,
	}

	if apiPath != "" {
		library.Channel = apiPath
	}

	if version := cmd.String("version"); version != "" {
		library.Version = version
	}

	if copyrightYear := cmd.String("copyright-year"); copyrightYear != "" {
		library.CopyrightYear = copyrightYear
	}

	if cmd.Bool("generate-disabled") {
		library.Generate = &config.LibraryGenerate{
			Disabled: true,
		}
	}

	if cmd.Bool("publish-disabled") {
		library.Publish = &config.LibraryPublish{
			Disabled: true,
		}
	}

	if cmd.Bool("per-service-features") || cmd.Bool("generate-setter-samples") {
		library.Rust = &config.RustCrate{}
		if cmd.Bool("per-service-features") {
			library.Rust.PerServiceFeatures = true
		}
		if cmd.Bool("generate-setter-samples") {
			library.Rust.GenerateSetterSamples = true
		}
	}

	if err := Add(ctx, cfg, googleapisDir, library); err != nil {
		return err
	}

	if err := cfg.Write(configPath); err != nil {
		return err
	}

	fmt.Printf("✓ Added %s to librarian.yaml\n", name)
	if library.Channel != "" {
		fmt.Printf("  API path: %s\n", library.Channel)
	}

	return nil
}
