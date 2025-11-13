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
	"strings"

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
			publishCommand(),
			releaseCommand(),
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

	if err := cfg.Add(name, apis, location, googleapisRoot); err != nil {
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
	var library *config.Library
	for i := range cfg.Libraries {
		if cfg.Libraries[i].Name == libraryName {
			library = &cfg.Libraries[i]
			break
		}
	}
	if library == nil {
		return fmt.Errorf("library %q not found in librarian.yaml", libraryName)
	}

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
	var library *config.Library
	for i := range cfg.Libraries {
		if cfg.Libraries[i].Name == libraryName {
			library = &cfg.Libraries[i]
			break
		}
	}
	if library == nil {
		return fmt.Errorf("library %q not found in librarian.yaml", libraryName)
	}

	// Check if this is a handwritten library (no APIs)
	if len(library.Apis) == 0 {
		return fmt.Errorf("library %q is handwritten (no apis field), nothing to generate", libraryName)
	}

	// Configure the library if not already configured
	// This creates the directory and initial files (go.mod, README.md, etc.)
	if err := configureLibrary(cfg, libraryName); err != nil {
		return fmt.Errorf("failed to configure library: %w", err)
	}

	// Dispatch to language-specific generator
	switch cfg.Language {
	case "go":
		if err := golang.Generate(ctx, cfg, library); err != nil {
			return err
		}
		fmt.Printf("Generated Go library %q at %s\n", library.Name, library.Name)
		return nil
	case "rust":
		// Download and extract googleapis if available
		var googleapisRoot string
		if cfg.Sources.Googleapis != nil {
			var err error
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
		}
		return rust.Generate(ctx, cfg, library, googleapisRoot)
	case "python":
		return generatePython(ctx, cfg, library)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}

func generatePython(ctx context.Context, cfg *config.Config, library *config.Library) (err error) {
	// Determine output directory
	outputDir := "{name}/"
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		outputDir = cfg.Generate.Output
	}

	location, err := library.GeneratedLocation(outputDir)
	if err != nil {
		return fmt.Errorf("failed to determine output location: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(location, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Download googleapis if available
	if cfg.Sources.Googleapis == nil {
		return fmt.Errorf("googleapis source is not configured in librarian.yaml")
	}

	var googleapisRoot string
	googleapisRoot, err = fetch.DownloadAndExtractTarball(cfg.Sources.Googleapis)
	if err != nil {
		err = fmt.Errorf("failed to download and extract googleapis: %w", err)
		return
	}
	defer func() {
		cerr := os.RemoveAll(filepath.Dir(googleapisRoot))
		if err == nil {
			err = cerr
		}
	}()

	// Run Python generator (with post-processor disabled by default)
	// TODO: Make post-processor configurable via librarian.yaml
	if err := python.Generate(ctx, library, location, googleapisRoot, true); err != nil {
		return fmt.Errorf("python generation failed: %w", err)
	}

	fmt.Printf("Generated Python library %q at %s\n", library.Name, location)
	return nil
}

// releaseCommand creates a release for a library.
func releaseCommand() *cli.Command {
	return &cli.Command{
		Name:      "release",
		Usage:     "create a release for a library",
		UsageText: "librarian release <library> --version <version>",
		Description: `Create a release for a library.

This command:
1. Runs language-specific tests
2. Updates version files and changelogs
3. Creates a git commit
4. Creates a git tag
5. Pushes the tag to remote

Example:
  librarian release secretmanager --version 1.16.0
  librarian release google-cloud-secret-manager --version 1.16.0`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "version",
				Usage:    "version to release (required)",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return errors.New("release requires a library name")
			}
			libraryName := cmd.Args().Get(0)
			version := cmd.String("version")
			return runRelease(ctx, libraryName, version)
		},
	}
}

func runRelease(ctx context.Context, libraryName, version string) error {
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err != nil {
		return errConfigNotFound
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	// Find the library
	var library *config.Library
	for i := range cfg.Libraries {
		if cfg.Libraries[i].Name == libraryName {
			library = &cfg.Libraries[i]
			break
		}
	}
	if library == nil {
		return fmt.Errorf("library %q not found in librarian.yaml", libraryName)
	}

	// Get repository root (current directory)
	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// TODO: Parse git commits and create Change structs
	// For now, create an empty changes slice
	changes := make([]interface{}, 0)

	// Dispatch to language-specific release
	switch cfg.Language {
	case "go":
		if err := golang.Release(ctx, repoRoot, library, version, nil); err != nil {
			return fmt.Errorf("go release failed: %w", err)
		}
	case "python":
		if err := python.Release(ctx, library, version, nil, repoRoot); err != nil {
			return fmt.Errorf("python release failed: %w", err)
		}
	case "rust":
		if err := rust.Release(ctx, repoRoot, library, version); err != nil {
			return fmt.Errorf("rust release failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}

	// Create tag name
	tagFormat := "{name}/v{version}"
	if cfg.Release != nil && cfg.Release.TagFormat != "" {
		tagFormat = cfg.Release.TagFormat
	}
	tagName := strings.ReplaceAll(tagFormat, "{name}", libraryName)
	tagName = strings.ReplaceAll(tagName, "{version}", version)

	fmt.Printf("Release preparation complete for %q version %s\n", libraryName, version)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Review the changes\n")
	fmt.Printf("  2. Commit: git add . && git commit -m \"chore(release): %s v%s\"\n", libraryName, version)
	fmt.Printf("  3. Tag: git tag %s\n", tagName)
	fmt.Printf("  4. Push: git push origin %s\n", tagName)
	fmt.Printf("  5. Publish: librarian publish %s\n", libraryName)

	_ = changes // Suppress unused warning
	return nil
}

// publishCommand publishes a released library to package registries.
func publishCommand() *cli.Command {
	return &cli.Command{
		Name:      "publish",
		Usage:     "publish a library to package registries",
		UsageText: "librarian publish <library>",
		Description: `Publish a library to package registries.

For Go: Verifies pkg.go.dev indexing
For Python: Publishes to PyPI (typically handled by CI/CD)
For Rust: Publishes to crates.io

Example:
  librarian publish secretmanager
  librarian publish google-cloud-secret-manager`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return errors.New("publish requires a library name")
			}
			libraryName := cmd.Args().Get(0)
			return runPublish(ctx, libraryName)
		},
	}
}

func runPublish(ctx context.Context, libraryName string) error {
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err != nil {
		return errConfigNotFound
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	// Find the library
	var library *config.Library
	for i := range cfg.Libraries {
		if cfg.Libraries[i].Name == libraryName {
			library = &cfg.Libraries[i]
			break
		}
	}
	if library == nil {
		return fmt.Errorf("library %q not found in librarian.yaml", libraryName)
	}

	// Get repository root (current directory)
	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// TODO: Find the latest tag for this library
	version := "latest"

	// Dispatch to language-specific publish
	switch cfg.Language {
	case "go":
		if err := golang.Publish(ctx, repoRoot, library, version); err != nil {
			return fmt.Errorf("go publish failed: %w", err)
		}
		fmt.Printf("Published Go library %q\n", libraryName)
	case "python":
		if err := python.Publish(ctx, library, repoRoot); err != nil {
			return fmt.Errorf("python publish failed: %w", err)
		}
		fmt.Printf("Published Python library %q\n", libraryName)
	case "rust":
		if err := rust.Publish(ctx, repoRoot, library, version); err != nil {
			return fmt.Errorf("rust publish failed: %w", err)
		}
		fmt.Printf("Published Rust library %q\n", libraryName)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}

	return nil
}
