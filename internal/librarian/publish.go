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

func publishCommand() *cli.Command {
	return &cli.Command{
		Name:      "publish",
		Usage:     "publish libraries to package registry",
		UsageText: "librarian publish [--dry-run] [--skip-semver-checks]",
		Description: `Publish all libraries that have changed since the last release.

By default, this runs cargo semver-checks and publishes to crates.io.
Use --dry-run to test without actually publishing.
Use --skip-semver-checks to skip the semver validation step.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "run without actually publishing",
			},
			&cli.BoolFlag{
				Name:  "skip-semver-checks",
				Usage: "skip cargo semver-checks validation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runPublish(ctx, cmd.Bool("dry-run"), cmd.Bool("skip-semver-checks"))
		},
	}
}

func runPublish(ctx context.Context, dryRun bool, skipSemverChecks bool) error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	if cfg.Language != "rust" {
		return fmt.Errorf("publish command only supports rust language, got: %s", cfg.Language)
	}

	fmt.Println("Publishing libraries...")
	if err := language.Publish(ctx, cfg, dryRun, skipSemverChecks); err != nil {
		return err
	}

	if dryRun {
		fmt.Println("✓ Dry run complete!")
	} else {
		fmt.Println("✓ Publish complete!")
	}
	return nil
}
