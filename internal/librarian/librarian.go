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
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/golang"
	"github.com/googleapis/librarian/internal/python"
	"github.com/googleapis/librarian/internal/rust"
	"github.com/urfave/cli/v3"
)

// Sentinel errors for validation.
var (
	errConfigAlreadyExists = errors.New("librarian.yaml already exists in current directory")
	errConfigNotFound      = errors.New("librarian.yaml not found in current directory")
	errInvalidKey          = errors.New("invalid key name")
)

// Run executes the librarian command with the given arguments.
func Run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Name:      "librarian",
		Usage:     "manage Google Cloud client libraries",
		UsageText: "librarian [command]",
		Version:   Version(),
		Commands: []*cli.Command{
			addCommand(),
			configCommand(),
			createCommand(),
			generateCommand(),
			initCommand(),
			// TODO(https://github.com/julieqiu/librarianx/issues/XXX): Implement publishCommand and releaseCommand
			// publishCommand(),
			// releaseCommand(),
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
	cfg := config.New(Version(), language, source)

	// Write config to librarian.yaml
	if err := cfg.Write(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Created librarian.yaml\n")
	return nil
}

// addCommand adds an library to the configuration.
func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add an library to the configuration",
		UsageText: "librarian add <name> [<api>...] [--location <path>]",
		Description: `Add an library to librarian.yaml.

For generated librarys, provide API paths:
  librarian add secretmanager google/cloud/secretmanager/v1 google/cloud/secretmanager/v1beta2

For handwritten librarys, use --location:
  librarian add gcloud-mcp --location packages/gcloud-mcp/`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "location",
				Usage: "explicit filesystem path for handwritten librarys",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return errors.New("add requires an library name")
			}
			name := cmd.Args().Get(0)
			location := cmd.String("location")

			// If location is provided, this is a handwritten library
			if location != "" {
				if cmd.NArg() > 1 {
					return errors.New("cannot specify both --location and API paths")
				}
				return runAdd(name, nil, location)
			}

			// Otherwise, this is a generated library with APIs
			if cmd.NArg() < 2 {
				return errors.New("add requires at least one API path or --location flag")
			}
			apis := cmd.Args().Slice()[1:]
			return runAdd(name, apis, "")
		},
	}
}

func runAdd(name string, apis []string, location string) error {
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err != nil {
		return errConfigNotFound
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	// Download googleapis if we have API paths to parse
	var googleapisRoot string
	if len(apis) > 0 && cfg.Sources.Googleapis != nil {
		var err error
		googleapisRoot, err = fetch.DownloadAndExtractTarball(cfg.Sources.Googleapis)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download googleapis: %v\n", err)
			fmt.Fprintf(os.Stderr, "Library will be added without parsed BUILD.bazel configuration\n")
			googleapisRoot = ""
		} else {
			defer os.RemoveAll(filepath.Dir(googleapisRoot))
		}
	}

	// TODO(https://github.com/julieqiu/librarianx/issues/XXX): Update to use new API-path-based Add method
	if err := cfg.AddLegacy(name, apis, location, googleapisRoot); err != nil {
		return err
	}

	if err := cfg.Write(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if location != "" {
		fmt.Printf("Added handwritten library %q at %s\n", name, location)
	} else {
		fmt.Printf("Added library %q with %d API(s)\n", name, len(apis))
	}
	return nil
}

// createCommand creates a new library with configuration and generation.
func createCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "create and configure a new library",
		UsageText: "librarian create <name> [<api>...] [--location <path>]",
		Description: `Create a new library with configuration and generation.

This command combines two steps into one:
1. Add the library to librarian.yaml (librarian add)
2. Generate client code from API definitions (librarian generate)

For generated libraries, provide API paths:
  librarian create secretmanager google/cloud/secretmanager/v1

For handwritten libraries, use --location:
  librarian create storage --location storage/`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "location",
				Usage: "explicit filesystem path for handwritten libraries",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return errors.New("create requires a library name")
			}
			name := cmd.Args().Get(0)
			location := cmd.String("location")

			var apis []string
			if location == "" {
				if cmd.NArg() < 2 {
					return errors.New("create requires at least one API path or --location flag")
				}
				apis = cmd.Args().Slice()[1:]
			}

			return runCreate(ctx, name, apis, location)
		},
	}
}

func runCreate(ctx context.Context, name string, apis []string, location string) error {
	// Step 1: Add the library to config
	if err := runAdd(name, apis, location); err != nil {
		return err
	}

	// Handwritten libraries don't need generation
	if location != "" {
		return nil
	}

	// Step 2: Generate the library code (which includes configuration)
	if err := runGenerate(ctx, name); err != nil {
		return fmt.Errorf("failed to generate library: %w", err)
	}

	fmt.Printf("Successfully created library %q\n", name)
	return nil
}

// configureLibrary performs language-specific library configuration.
// This creates initial files like go.mod, README.md, CHANGES.md, etc.
func configureLibrary(cfg *config.Config, libraryName string) error {
	// Find the library
	entry, _ := cfg.FindLibraryByName(libraryName)
	if entry == nil {
		return fmt.Errorf("library %q not found in librarian.yaml", libraryName)
	}

	// Convert to Library for backward compatibility
	library := entry.ToLibrary()

	// Dispatch to language-specific configurator
	switch cfg.Language {
	case "go":
		return configureGo(cfg, library)
	case "rust":
		// Rust libraries are configured by sidekick during generation
		return nil
	case "python":
		return configurePython(cfg, library)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}

func configureGo(cfg *config.Config, library *config.Library) error {
	// Get the library's output location
	var outputDir string
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		var err error
		outputDir, err = library.GeneratedLocation(cfg.Generate.Output)
		if err != nil {
			return err
		}
	} else {
		// Default output location if generate.output is not set
		outputDir = library.Name + "/"
	}

	// Create the library directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	fmt.Printf("Configured Go library at %s\n", outputDir)
	// TODO(https://github.com/googleapis/librarian/issues/XXX): Create go.mod, README.md, CHANGES.md
	return nil
}

func configurePython(cfg *config.Config, library *config.Library) error {
	// Get the library's output location
	var outputDir string
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		var err error
		outputDir, err = library.GeneratedLocation(cfg.Generate.Output)
		if err != nil {
			return err
		}
	} else {
		// Default output location if generate.output is not set
		outputDir = library.Name + "/"
	}

	// Create the library directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	fmt.Printf("Configured Python library at %s\n", outputDir)
	// TODO(https://github.com/googleapis/librarian/issues/XXX): Create setup.py, README.md
	return nil
}

// configCommand manages configuration.
func configCommand() *cli.Command {
	return &cli.Command{
		Name:      "config",
		Usage:     "manage configuration",
		UsageText: "librarian config <command>",
		Commands: []*cli.Command{
			configSetCommand(),
			configUnsetCommand(),
		},
	}
}

// configSetCommand sets a configuration key value.
func configSetCommand() *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     "set a configuration key",
		UsageText: "librarian config set <key> <value>",
		Description: `Set configuration key values in librarian.yaml.

Supported keys:
  release.tag_format  - Git tag format template
  generate.output     - Output directory for generated code

Example:
  librarian config set release.tag_format '{id}/v{version}'
  librarian config set generate.output packages/`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return errors.New("set requires key and value")
			}
			key := cmd.Args().Get(0)
			value := cmd.Args().Get(1)
			return runSet(key, value)
		},
	}
}

func runSet(key, value string) error {
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err != nil {
		return errConfigNotFound
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	if err := cfg.Set(key, value); err != nil {
		return fmt.Errorf("%w: %s", errInvalidKey, key)
	}

	if err := cfg.Write(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Set %s to %q\n", key, value)
	return nil
}

// configUnsetCommand removes a configuration key value.
func configUnsetCommand() *cli.Command {
	return &cli.Command{
		Name:      "unset",
		Usage:     "unset a configuration key",
		UsageText: "librarian config unset <key>",
		Description: `Unset configuration key values in librarian.yaml.

This removes the key from the configuration file.

Supported keys:
  release.tag_format  - Git tag format template
  generate.output     - Output directory for generated code

Example:
  librarian config unset release.tag_format
  librarian config unset generate.output`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return errors.New("unset requires a key")
			}
			key := cmd.Args().Get(0)
			return runUnset(key)
		},
	}
}

func runUnset(key string) error {
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err != nil {
		return errConfigNotFound
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	if err := cfg.Unset(key); err != nil {
		return fmt.Errorf("%w: %s", errInvalidKey, key)
	}

	if err := cfg.Write(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Unset %s\n", key)
	return nil
}

// generateCommand generates code for librarys.
func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate code for an library",
		UsageText: "librarian generate <library>",
		Description: `Generate code for an library from API definitions.

This command generates client library code from googleapis API definitions
based on the configuration in librarian.yaml.

Example:
  librarian generate secretmanager`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return errors.New("generate requires an library name")
			}
			library := cmd.Args().Get(0)
			return runGenerate(ctx, library)
		},
	}
}

func runGenerate(ctx context.Context, libraryName string) error {
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err != nil {
		return errConfigNotFound
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	// Find the library
	entry, _ := cfg.FindLibraryByName(libraryName)
	if entry == nil {
		return fmt.Errorf("library %q not found in librarian.yaml", libraryName)
	}

	// Convert to Library for backward compatibility
	library := entry.ToLibrary()

	// Check if this is a handwritten library (no APIs)
	if len(library.Apis) == 0 {
		return fmt.Errorf("library %q is handwritten (no apis field), nothing to generate", libraryName)
	}

	// Configure the library if not already configured
	// This creates the directory and initial files (go.mod, README.md, etc.)
	if err := configureLibrary(cfg, libraryName); err != nil {
		return fmt.Errorf("failed to configure library: %w", err)
	}

	// Download and extract googleapis if available (used by rust and python)
	var googleapisRoot string
	if cfg.Sources.Googleapis == nil {
		return fmt.Errorf("googleapis source is not configured in librarian.yaml")
	}

	googleapisRoot, err = fetch.DownloadAndExtractTarball(cfg.Sources.Googleapis)
	if err != nil {
		return fmt.Errorf("failed to download and extract googleapis: %w", err)
	}
	defer func() {
		cerr := os.RemoveAll(filepath.Dir(googleapisRoot))
		if err == nil {
			err = cerr
		}
	}()

	// Dispatch to language-specific generator
	switch cfg.Language {
	case "go":
		return golang.Generate(ctx, cfg, library, googleapisRoot)
	case "rust":
		return rust.Generate(ctx, cfg, library, googleapisRoot)
	case "python":
		return python.Generate(ctx, cfg, library, googleapisRoot)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}
