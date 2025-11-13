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
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/golang"
	"github.com/googleapis/librarian/internal/golang/generate"
	"github.com/googleapis/librarian/internal/python"
	"github.com/googleapis/librarian/internal/sidekick/sidekick"
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
		googleapisRoot, err = downloadAndExtractTarball(cfg.Sources.Googleapis)
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
		return configureRust(cfg, library)
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

func configureRust(cfg *config.Config, library *config.Library) error {
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

	fmt.Printf("Configured Rust library at %s\n", outputDir)
	// TODO(https://github.com/googleapis/librarian/issues/XXX): Create Cargo.toml, README.md
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

func downloadAndExtractTarball(source *config.Source) (string, error) {
	// Create a temporary directory for extraction
	tmpDir, err := os.MkdirTemp("", "googleapis-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Download the tarball
	resp, err := http.Get(source.URL)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download tarball: HTTP %d - %s", resp.StatusCode, resp.Status)
	}

	// Verify SHA256
	hasher := sha256.New()
	teeReader := io.TeeReader(resp.Body, hasher)

	tarballPath := filepath.Join(tmpDir, "source.tar.gz")
	file, err := os.Create(tarballPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create tarball file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, teeReader); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write tarball to file: %w", err)
	}

	if fmt.Sprintf("%x", hasher.Sum(nil)) != source.SHA256 {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("SHA256 checksum mismatch for %s", source.URL)
	}

	// Extract the tarball
	if err := extractTarball(tarballPath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}

	// The tarball usually extracts into a directory named after the archive, e.g., googleapis-sha
	// Find the actual extracted directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to read temp directory after extraction: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "googleapis-") {
			return filepath.Join(tmpDir, entry.Name()), nil
		}
	}

	os.RemoveAll(tmpDir)
	return "", fmt.Errorf("could not find extracted googleapis directory in %s", tmpDir)
}

// extractTarball extracts a gzipped tarball to a destination directory.
func extractTarball(tarballPath, destDir string) error {
	file, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}

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
		return generateGo(ctx, cfg, library)
	case "rust":
		return generateRust(cfg, library)
	case "python":
		return generatePython(ctx, cfg, library)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}

func generateGo(ctx context.Context, cfg *config.Config, library *config.Library) (err error) {
	// Determine output directory
	outputDir := "{name}/"
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		outputDir = cfg.Generate.Output
	}

	location, err := library.GeneratedLocation(outputDir)
	if err != nil {
		return fmt.Errorf("failed to determine output location: %w", err)
	}

	// Convert to absolute path
	absLocation, err := filepath.Abs(location)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", location, err)
	}

	// Download googleapis if available
	if cfg.Sources.Googleapis == nil {
		return fmt.Errorf("googleapis source is not configured in librarian.yaml")
	}

	var googleapisRoot string
	googleapisRoot, err = downloadAndExtractTarball(cfg.Sources.Googleapis)
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

	// Create temporary librarian directory structure
	librarianDir, err := os.MkdirTemp("", "librarian-go-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary librarian directory: %w", err)
	}
	defer func() {
		cerr := os.RemoveAll(librarianDir)
		if err == nil {
			err = cerr
		}
	}()

	// Create generator-input directory
	generatorInputDir := filepath.Join(librarianDir, "generator-input")
	if err := os.MkdirAll(generatorInputDir, 0755); err != nil {
		return fmt.Errorf("failed to create generator-input directory: %w", err)
	}

	// Create minimal repo-config.yaml
	repoConfigPath := filepath.Join(generatorInputDir, "repo-config.yaml")
	repoConfigContent := "modules: []\n"
	if err := os.WriteFile(repoConfigPath, []byte(repoConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write repo-config.yaml: %w", err)
	}

	// Create generate-request.json
	generateRequest := struct {
		ID   string `json:"id"`
		APIs []struct {
			Path string `json:"path"`
		} `json:"apis"`
	}{
		ID: library.Name,
	}

	for _, apiPath := range library.Apis {
		generateRequest.APIs = append(generateRequest.APIs, struct {
			Path string `json:"path"`
		}{Path: apiPath})
	}

	requestJSON, err := json.Marshal(generateRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal generate request: %w", err)
	}

	requestPath := filepath.Join(librarianDir, "generate-request.json")
	if err := os.WriteFile(requestPath, requestJSON, 0644); err != nil {
		return fmt.Errorf("failed to write generate-request.json: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(absLocation, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Call golang.Generate
	genCfg := &generate.Config{
		LibrarianDir:         librarianDir,
		InputDir:             generatorInputDir,
		OutputDir:            absLocation,
		SourceDir:            googleapisRoot,
		DisablePostProcessor: false,
	}

	if err := golang.Generate(ctx, genCfg); err != nil {
		return fmt.Errorf("go generation failed: %w", err)
	}

	fmt.Printf("Generated Go library %q at %s\n", library.Name, location)
	return nil
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
	googleapisRoot, err = downloadAndExtractTarball(cfg.Sources.Googleapis)
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

func generateRust(cfg *config.Config, library *config.Library) (err error) {
	// Validate Rust-specific requirements
	if len(library.Apis) != 1 {
		return fmt.Errorf("rust generation requires exactly one API per library, got %d for library %q", len(library.Apis), library.Name)
	}

	// Determine output directory
	outputDir := "{name}/"
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		outputDir = cfg.Generate.Output
	}

	location, err := library.GeneratedLocation(outputDir)
	if err != nil {
		return fmt.Errorf("failed to determine output location: %w", err)
	}

	// Build sidekick command line arguments
	args := []string{
		"sidekick",
		"rust-generate",
		"--specification-source", library.Apis[0],
		"--output", location,
		"--language", "rust",
	}

	// Add googleapis source if available
	if cfg.Sources.Googleapis != nil {
		var googleapisRoot string
		googleapisRoot, err = downloadAndExtractTarball(cfg.Sources.Googleapis)
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
		args = append(args, "--source-option", fmt.Sprintf("googleapis-root=%s", googleapisRoot))
	}

	// Add copyright year if specified
	if library.CopyrightYear > 0 {
		args = append(args, "--codec-option", fmt.Sprintf("copyright-year=%d", library.CopyrightYear))
	}

	// Run sidekick
	if runErr := sidekick.Run(args[1:]); runErr != nil {
		err = fmt.Errorf("sidekick rust-generate failed: %w", runErr)
		return
	}

	fmt.Printf("Generated Rust library %q at %s\n", library.Name, location)
	return
}
