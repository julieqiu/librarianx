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

// Package librarian provides functionality for managing Google Cloud client library configurations.
package librarian

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/urfave/cli/v3"
)

// Sentinel errors for validation.
var (
	errConfigAlreadyExists = errors.New("librarian.yaml already exists in current directory")
)

// Run executes the librarian command with the given arguments.
func Run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Name:      "librarian",
		Usage:     "manage Google Cloud client libraries",
		UsageText: "librarian [command]",
		Version:   Version(),
		Commands: []*cli.Command{
			initCommand(),
			versionCommand(),
		},
	}

	return cmd.Run(ctx, args)
}

// versionCommand prints the version information.
func versionCommand() *cli.Command {
	return &cli.Command{
		Name:      "version",
		Usage:     "print the version",
		UsageText: "librarian version",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Printf("librarian version %s\n", Version())
			return nil
		},
	}
}

// initCommand creates a new repository configuration.
func initCommand() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Usage:     "initialize librarian in current directory",
		UsageText: "librarian init [language]",
		Description: `Initialize librarian in current directory.
Creates librarian.yaml with default settings.

If no language is specified, the directory will be setup for release only.
If language is specified, creates configuration for that language.
Supported languages: go, python, rust

Example:
  librarian init
  librarian init go
  librarian init python`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			language := ""
			if cmd.NArg() > 0 {
				language = cmd.Args().Get(0)
			}

			// Fetch latest googleapis commit and SHA256 if language is specified
			var source *config.Source
			if language != "" {
				var err error
				source, err = fetch.LatestGoogleapis()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to fetch latest googleapis commit: %v\n", err)
					fmt.Fprintf(os.Stderr, "Using empty source configuration. You can update it later with 'librarian update --googleapis'\n")
				}
			}

			return runInit(language, source)
		},
	}
}

func runInit(language string, source *config.Source) error {
	// Check if librarian.yaml already exists
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err == nil {
		return errConfigAlreadyExists
	}

	// Create default config based on language
	cfg := createDefaultConfig(language, source)

	// Write config to librarian.yaml
	if err := cfg.Write(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Created librarian.yaml\n")
	return nil
}

func createDefaultConfig(language string, source *config.Source) *config.Config {
	cfg := &config.Config{
		Version: Version(),
		Release: &config.Release{
			TagFormat: "{name}/v{version}",
		},
	}

	if language == "" {
		// No language specified - minimal config with release defaults
		return cfg
	}

	// Language-specific configuration
	cfg.Language = language

	if source != nil {
		cfg.Sources = config.Sources{
			Googleapis: source,
		}
	}

	return cfg
}
