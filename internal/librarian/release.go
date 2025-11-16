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
	"os/exec"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language"
	"github.com/urfave/cli/v3"
)

func releaseCommand() *cli.Command {
	return &cli.Command{
		Name:      "release",
		Usage:     "bump versions for release",
		UsageText: "librarian release [--execute]",
		Description: `Bump versions for all Cargo.toml files and update librarian.yaml.

By default, this is a dry run that only updates Cargo.toml and librarian.yaml files.
Use --execute to also create and push git tags.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "execute",
				Usage: "create and push git tags (default is dry run)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runRelease(ctx, cmd.Bool("execute"))
		},
	}
}

func runRelease(ctx context.Context, execute bool) error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	if cfg.Language != "rust" {
		return fmt.Errorf("release command only supports rust language, got: %s", cfg.Language)
	}

	// Always bump versions (updates Cargo.toml and librarian.yaml)
	fmt.Println("Bumping versions...")
	if err := language.Release(ctx, cfg, configPath); err != nil {
		return err
	}
	fmt.Println("✓ Updated Cargo.toml files and librarian.yaml")

	if !execute {
		fmt.Println("\nDry run complete. Run with --execute to create and push tags.")
		return nil
	}

	// Create and push tags
	fmt.Println("\nCreating and pushing tags...")
	if err := createAndPushTags(cfg); err != nil {
		return err
	}

	fmt.Println("✓ Release complete!")
	return nil
}

func createAndPushTags(cfg *config.Config) error {
	if cfg.Default == nil || cfg.Default.Release == nil {
		return fmt.Errorf("default.release configuration is required")
	}

	tagFormat := cfg.Default.Release.TagFormat
	if tagFormat == "" {
		tagFormat = "{name}/v{version}"
	}

	remote := cfg.Default.Release.Remote
	if remote == "" {
		remote = "origin"
	}

	for name, version := range cfg.Versions {
		tag := strings.ReplaceAll(tagFormat, "{name}", name)
		tag = strings.ReplaceAll(tag, "{version}", version)

		// Create tag
		fmt.Printf("  Creating tag %s...\n", tag)
		cmd := exec.Command("git", "tag", tag)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create tag %s: %w\n%s", tag, err, string(output))
		}

		// Push tag
		fmt.Printf("  Pushing tag %s...\n", tag)
		cmd = exec.Command("git", "push", remote, tag)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to push tag %s: %w\n%s", tag, err, string(output))
		}
	}

	return nil
}
