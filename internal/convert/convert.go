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

// Package convert provides functionality for converting old .librarian format to new librarian.yaml format.
package convert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// OldConfig represents the old .librarian/config.yaml format.
type OldConfig struct {
	GlobalFilesAllowlist []GlobalFileAllowlist `yaml:"global_files_allowlist"`
	Libraries            []OldConfigLibrary    `yaml:"libraries"`
}

// GlobalFileAllowlist represents a global file allowlist entry.
type GlobalFileAllowlist struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions"`
}

// OldConfigLibrary represents a library entry in config.yaml.
type OldConfigLibrary struct {
	ID             string `yaml:"id"`
	ReleaseBlocked bool   `yaml:"release_blocked"`
}

// OldState represents the old .librarian/state.yaml format.
type OldState struct {
	Image     string       `yaml:"image"`
	Libraries []OldLibrary `yaml:"libraries"`
}

// OldLibrary represents a library in the old format.
type OldLibrary struct {
	ID                  string   `yaml:"id"`
	Version             string   `yaml:"version"`
	LastGeneratedCommit string   `yaml:"last_generated_commit"`
	APIs                []OldAPI `yaml:"apis"`
	SourceRoots         []string `yaml:"source_roots"`
	PreserveRegex       []string `yaml:"preserve_regex"`
	RemoveRegex         []string `yaml:"remove_regex"`
	TagFormat           string   `yaml:"tag_format"`
}

// OldAPI represents an API in the old format.
type OldAPI struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config"`
}

// OldRepoConfig represents the old .librarian/generator-input/repo-config.yaml format.
type OldRepoConfig struct {
	Modules []OldModule `yaml:"modules"`
}

// OldModule represents a module in repo-config.yaml.
type OldModule struct {
	Name                        string         `yaml:"name"`
	ModulePathVersion           string         `yaml:"module_path_version"`
	APIs                        []OldModuleAPI `yaml:"apis"`
	DeleteGenerationOutputPaths []string       `yaml:"delete_generation_output_paths"`
}

// OldModuleAPI represents an API in a module.
type OldModuleAPI struct {
	Path            string   `yaml:"path"`
	ClientDirectory string   `yaml:"client_directory"`
	DisableGapic    bool     `yaml:"disable_gapic"`
	ProtoPackage    string   `yaml:"proto_package"`
	NestedProtos    []string `yaml:"nested_protos"`
}

// Run executes the convert command with the given arguments.
func Run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Name:      "convert",
		Usage:     "convert old .librarian format to new librarian.yaml format",
		UsageText: "convert <input-dir> <output-file>",
		Description: `Convert old .librarian format to new librarian.yaml format.

Reads .librarian/config.yaml and .librarian/state.yaml from the input directory
and outputs a librarian.yaml file to the specified output path.

Example:
  convert /path/to/google-cloud-go data/go/librarian.yaml`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 2 {
				return fmt.Errorf("requires exactly 2 arguments: <input-dir> <output-file>")
			}

			inputDir := cmd.Args().Get(0)
			outputFile := cmd.Args().Get(1)

			return Convert(inputDir, outputFile)
		},
	}

	return cmd.Run(ctx, args)
}

// Convert reads the old .librarian format and converts it to the new librarian.yaml format.
func Convert(inputDir, outputFile string) error {
	librarianDir := filepath.Join(inputDir, ".librarian")

	// Read config.yaml
	oldConfig, err := readConfigYAML(librarianDir)
	if err != nil {
		return err
	}

	// Read state.yaml
	oldState, err := readStateYAML(librarianDir)
	if err != nil {
		return err
	}

	// Read repo-config.yaml
	oldRepoConfig, err := readRepoConfigYAML(librarianDir)
	if err != nil {
		return err
	}

	// Convert to new format
	newConfig := convertToNewFormat(oldConfig, oldState, oldRepoConfig)

	// Create output directory if it does not exist
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write new config
	if err := newConfig.Write(outputFile); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully converted to %s\n", outputFile)
	return nil
}

// readConfigYAML reads the .librarian/config.yaml file.
func readConfigYAML(librarianDir string) (*OldConfig, error) {
	configPath := filepath.Join(librarianDir, "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	var oldConfig OldConfig
	if err := yaml.Unmarshal(configData, &oldConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config.yaml: %w", err)
	}

	return &oldConfig, nil
}

// readStateYAML reads the .librarian/state.yaml file.
func readStateYAML(librarianDir string) (*OldState, error) {
	statePath := filepath.Join(librarianDir, "state.yaml")
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", statePath, err)
	}

	var oldState OldState
	if err := yaml.Unmarshal(stateData, &oldState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state.yaml: %w", err)
	}

	return &oldState, nil
}

// readRepoConfigYAML reads the .librarian/generator-input/repo-config.yaml file.
func readRepoConfigYAML(librarianDir string) (*OldRepoConfig, error) {
	repoConfigPath := filepath.Join(librarianDir, "generator-input", "repo-config.yaml")
	repoConfigData, err := os.ReadFile(repoConfigPath)
	if err != nil {
		// repo-config.yaml is optional
		if os.IsNotExist(err) {
			return &OldRepoConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", repoConfigPath, err)
	}

	var oldRepoConfig OldRepoConfig
	if err := yaml.Unmarshal(repoConfigData, &oldRepoConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repo-config.yaml: %w", err)
	}

	return &oldRepoConfig, nil
}

// convertToNewFormat converts the old format to the new format.
func convertToNewFormat(oldConfig *OldConfig, oldState *OldState, oldRepoConfig *OldRepoConfig) *config.Config {
	// Parse container image and tag
	containerImage, containerTag := parseImage(oldState.Image)

	// Auto-detect language from container image
	language := detectLanguageFromImage(containerImage)

	// Create new config
	newConfig := &config.Config{
		Version:  "v1",
		Language: language,
		Container: &config.Container{
			Image: containerImage,
			Tag:   containerTag,
		},
		Defaults: &config.Defaults{
			Output:           "./",
			OneLibraryPer:    "service",
			Transport:        "grpc+rest",
			RestNumericEnums: true,
		},
		Release: &config.Release{
			TagFormat: "{name}/v{version}",
		},
	}

	// Add wildcard to generate everything
	newConfig.Libraries = append(newConfig.Libraries, config.LibraryEntry{
		APIPath: "*",
	})

	// Add global files allowlist
	if len(oldConfig.GlobalFilesAllowlist) > 0 {
		newConfig.Global = &config.Global{
			FilesAllowlist: make([]config.FileAllowlist, len(oldConfig.GlobalFilesAllowlist)),
		}
		for i, fa := range oldConfig.GlobalFilesAllowlist {
			newConfig.Global.FilesAllowlist[i] = config.FileAllowlist{
				Path:        fa.Path,
				Permissions: fa.Permissions,
			}
		}
	}

	// Create a map of release_blocked libraries
	releaseBlockedMap := make(map[string]bool)
	for _, lib := range oldConfig.Libraries {
		if lib.ReleaseBlocked {
			releaseBlockedMap[lib.ID] = true
		}
	}

	// Create a map of module configs
	moduleConfigMap := make(map[string]*OldModule)
	for i := range oldRepoConfig.Modules {
		moduleConfigMap[oldRepoConfig.Modules[i].Name] = &oldRepoConfig.Modules[i]
	}

	// Convert libraries
	for _, oldLib := range oldState.Libraries {
		newLib := config.Library{
			Name:    oldLib.ID,
			Version: oldLib.Version,
		}

		// Add source_roots only if they differ from the standard pattern
		// Standard pattern: [{name}, internal/generated/snippets/{name}]
		if len(oldLib.SourceRoots) > 0 {
			isStandardPattern := len(oldLib.SourceRoots) == 2 &&
				oldLib.SourceRoots[0] == oldLib.ID &&
				oldLib.SourceRoots[1] == "internal/generated/snippets/"+oldLib.ID
			if !isStandardPattern {
				newLib.SourceRoots = oldLib.SourceRoots
			}
		}

		// Check if this library has module-specific config
		if moduleConfig, ok := moduleConfigMap[oldLib.ID]; ok {
			// Add module_path_version
			if moduleConfig.ModulePathVersion != "" {
				newLib.ModulePathVersion = moduleConfig.ModulePathVersion
			}

			// Add delete_generation_output_paths
			if len(moduleConfig.DeleteGenerationOutputPaths) > 0 {
				if newLib.Generate == nil {
					newLib.Generate = &config.LibraryGenerate{}
				}
				newLib.Generate.DeleteOutputPaths = moduleConfig.DeleteGenerationOutputPaths
			}
		}

		// Check if this library has release_blocked
		if releaseBlockedMap[oldLib.ID] {
			if newLib.Release == nil {
				newLib.Release = &config.LibraryRelease{}
			}
			newLib.Release.Disabled = true
		}

		// Convert APIs
		if len(oldLib.APIs) > 0 {
			if newLib.Generate == nil {
				newLib.Generate = &config.LibraryGenerate{}
			}
			newLib.Generate.APIs = make([]config.API, len(oldLib.APIs))

			for i, api := range oldLib.APIs {
				newAPI := config.API{
					Path: api.Path,
				}

				// Check for API-specific overrides in module config
				if moduleConfig, ok := moduleConfigMap[oldLib.ID]; ok {
					for _, moduleAPI := range moduleConfig.APIs {
						if moduleAPI.Path == api.Path {
							// Initialize Go config if any overrides are present
							if moduleAPI.ClientDirectory != "" || moduleAPI.DisableGapic ||
								moduleAPI.ProtoPackage != "" || len(moduleAPI.NestedProtos) > 0 {
								newAPI.Go = &config.GoAPI{}
								if moduleAPI.ClientDirectory != "" {
									newAPI.Go.ClientDirectory = moduleAPI.ClientDirectory
								}
								if moduleAPI.DisableGapic {
									newAPI.Go.DisableGapic = true
								}
								if moduleAPI.ProtoPackage != "" {
									newAPI.Go.ProtoPackage = moduleAPI.ProtoPackage
								}
								if len(moduleAPI.NestedProtos) > 0 {
									newAPI.Go.NestedProtos = moduleAPI.NestedProtos
								}
							}
							break
						}
					}
				}

				newLib.Generate.APIs[i] = newAPI
			}

			// Convert preserve_regex to keep
			if len(oldLib.PreserveRegex) > 0 {
				newLib.Generate.Keep = oldLib.PreserveRegex
			}
		}

		// Set release tag format if present
		if oldLib.TagFormat != "" {
			if newConfig.Release == nil {
				newConfig.Release = &config.Release{}
			}
			// Use the first tag format found
			if newConfig.Release.TagFormat == "" {
				// Convert {id} to {name} since we renamed the field
				tagFormat := strings.ReplaceAll(oldLib.TagFormat, "{id}", "{name}")
				newConfig.Release.TagFormat = tagFormat
			}
		}

		// Convert Library to LibraryEntry
		// Use source_roots to determine filesystem path
		// For Python: packages/google-cloud-vision
		// For Go: batch
		libraryPath := oldLib.ID
		if len(oldLib.SourceRoots) > 0 {
			// Use first source root as the library path
			libraryPath = oldLib.SourceRoots[0]
		}
		// All library paths should have trailing slash to indicate they're directories
		if !strings.HasSuffix(libraryPath, "/") {
			libraryPath = libraryPath + "/"
		}

		// Build library config for exceptions
		var cfg *config.LibraryConfig

		// Add keep patterns if present
		if len(oldLib.PreserveRegex) > 0 {
			// Process keep patterns to remove redundancy
			deduped := deduplicateKeepPatterns(oldLib.PreserveRegex, libraryPath)
			// Only add if there are non-empty patterns after deduplication
			if len(deduped) > 0 {
				if cfg == nil {
					cfg = &config.LibraryConfig{}
				}
				cfg.Keep = deduped
			}
		}

		// Add source roots if present and different from library path
		// Since we use source_roots[0] as the library path identifier,
		// only add source_roots if there are multiple or if they're unusual
		if len(newLib.SourceRoots) > 1 {
			if cfg == nil {
				cfg = &config.LibraryConfig{}
			}
			cfg.SourceRoots = newLib.SourceRoots
		}

		// Add release config if present
		if newLib.Release != nil {
			if cfg == nil {
				cfg = &config.LibraryConfig{}
			}
			cfg.Release = newLib.Release
		}

		// Only add library entry if it has config (is an exception to wildcard)
		if cfg != nil {
			entry := config.LibraryEntry{
				APIPath: libraryPath,
				Config:  cfg,
			}
			newConfig.Libraries = append(newConfig.Libraries, entry)
		}
	}

	return newConfig
}

// parseImage splits a container image string into image and tag.
// Example: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go:latest"
// Returns: ("us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go", "latest").
func parseImage(image string) (string, string) {
	parts := strings.Split(image, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	// Default to latest if no tag specified
	return image, "latest"
}

// deduplicateKeepPatterns removes redundant keep patterns.
// For Python libraries, many have identical keep patterns that can be omitted.
func deduplicateKeepPatterns(patterns []string, libraryPath string) []string {
	// Common patterns that appear in most Python libraries
	// These are the shared repository paths that don't need to be listed per-library
	commonPatterns := map[string]bool{
		"docs/CHANGELOG.md":              true,
		"docs/README.rst":                true,
		"samples/README.txt":             true,
		"scripts/client-post-processing": true,
		"samples/snippets/README.rst":    true,
		"tests/system":                   true,
		"CHANGELOG.md":                   true, // Also skip the relative CHANGELOG.md
	}

	var result []string
	for _, pattern := range patterns {
		// For library-specific patterns, make them relative if they start with the library path
		processedPattern := pattern
		if strings.HasPrefix(pattern, libraryPath+"/") {
			// Make it relative to the library
			processedPattern = strings.TrimPrefix(pattern, libraryPath+"/")
		}

		// Skip common patterns (check after making relative)
		if commonPatterns[processedPattern] {
			continue
		}

		result = append(result, processedPattern)
	}

	return result
}

// detectLanguageFromImage extracts the language from the container image name.
// Example: "librarian-go" -> "go", "python-librarian-generator" -> "python"
func detectLanguageFromImage(image string) string {
	// Get the last path component
	parts := strings.Split(image, "/")
	imageName := parts[len(parts)-1]

	// Common patterns in container image names
	if strings.Contains(imageName, "python") {
		return "python"
	}
	if strings.Contains(imageName, "-go") || strings.HasSuffix(imageName, "go") {
		return "go"
	}
	if strings.Contains(imageName, "rust") {
		return "rust"
	}
	if strings.Contains(imageName, "dart") {
		return "dart"
	}

	// Default to go if we can't determine
	return "go"
}
