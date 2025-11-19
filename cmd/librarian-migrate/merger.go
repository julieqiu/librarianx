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

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/julieqiu/librarianx/internal/config"
	"gopkg.in/yaml.v3"
)

// merge combines data from all sources and returns a config.Config.
func merge(state *LegacyState, legacyConfig *LegacyConfig, buildData *BuildBazelData, generatorInput *LegacyGeneratorInputData, language string) *config.Config {
	// Create a map of config overrides for quick lookup
	configMap := make(map[string]*LegacyConfigLibrary)
	for i := range legacyConfig.Libraries {
		lib := &legacyConfig.Libraries[i]
		configMap[lib.ID] = lib
	}

	// Parse generator-input data (repo-config.yaml for Go)
	repoConfig, err := parseRepoConfig(generatorInput)
	if err != nil {
		// Log warning but continue - generator input is optional
		fmt.Fprintf(os.Stderr, "Warning: failed to parse repo-config.yaml: %v\n", err)
	}

	// Merge each library and track tag formats for default identification
	var libraries []*config.Library
	var tagFormats []string
	for _, stateLib := range state.Libraries {
		// Only filter keep patterns for Python repositories
		keepPatterns := stateLib.PreserveRegex
		if language == "python" {
			keepPatterns = filterKeepPatterns(stateLib.PreserveRegex, stateLib.ID)
		}

		lib := &config.Library{
			Name: stateLib.ID,
			Keep: keepPatterns,
		}

		// Extract API paths
		for _, api := range stateLib.APIs {
			lib.APIs = append(lib.APIs, api.Path)
		}

		// Track tag format for default identification
		if stateLib.TagFormat != "" {
			// Replace {id} with {name} for new format
			tagFormat := strings.ReplaceAll(stateLib.TagFormat, "{id}", "{name}")
			tagFormats = append(tagFormats, tagFormat)
		}

		// Merge config overrides if present
		if configLib, ok := configMap[stateLib.ID]; ok {
			if configLib.GenerateBlocked {
				lib.Generate = &config.LibraryGenerate{Disabled: true}
			}
			if configLib.ReleaseBlocked {
				lib.Release = &config.LibraryRelease{Disabled: true}
			}
		}

		// Merge BUILD.bazel data if present
		if buildLib, ok := buildData.Libraries[stateLib.ID]; ok {
			lib.Transport = buildLib.Transport
			lib.GRPCServiceConfig = buildLib.GRPCServiceConfig
			if buildLib.RestNumericEnums {
				lib.RestNumericEnums = &buildLib.RestNumericEnums
			}

			// Merge language-specific fields
			switch language {
			case "python":
				if len(buildLib.OptArgs) > 0 || buildLib.IsProtoOnly {
					if lib.Python == nil {
						lib.Python = &config.PythonPackage{}
					}
					lib.Python.OptArgs = buildLib.OptArgs
					lib.Python.IsProtoOnly = buildLib.IsProtoOnly
				}
			case "go":
				if buildLib.ImportPath != "" || buildLib.Metadata {
					if lib.Go == nil {
						lib.Go = &config.GoModule{}
					}
					if buildLib.ImportPath != "" {
						lib.Go.ImportPath = buildLib.ImportPath
					}
					if buildLib.Metadata {
						lib.Go.Metadata = buildLib.Metadata
					}
				}
				if buildLib.ReleaseLevel != "" {
					lib.ReleaseLevel = buildLib.ReleaseLevel
				}
			}
		}

		// Merge generator-input (repo-config.yaml) data for Go
		if language == "go" && repoConfig != nil {
			if moduleConfig, ok := repoConfig.Modules[stateLib.ID]; ok {
				if lib.Go == nil {
					lib.Go = &config.GoModule{}
				}
				lib.Go.ModulePathVersion = moduleConfig.ModulePathVersion
				lib.Go.DeleteGenerationOutputPaths = moduleConfig.DeleteGenerationOutputPaths
				lib.Go.APIs = moduleConfig.APIs
			}
		}

		libraries = append(libraries, lib)
	}

	// Determine output path and repo based on language
	outputPath := "{name}"
	repo := ""
	switch language {
	case "python":
		outputPath = "packages/{name}/"
		repo = "googleapis/google-cloud-python"
	case "go":
		repo = "googleapis/google-cloud-go"
	}

	// Create config with defaults
	cfg := &config.Config{
		Version:   "v1",
		Language:  language,
		Repo:      repo,
		Libraries: libraries,
		Default: &config.Default{
			Output: outputPath,
			Generate: &config.DefaultGenerate{
				All:              true,
				OneLibraryPer:    "api",
				RestNumericEnums: true,
			},
			Release: &config.DefaultRelease{},
		},
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "9fcfbea0aa5b50fa22e190faceb073d74504172b",
			},
		},
	}

	// Identify and populate common defaults
	identifyDefaults(cfg, tagFormats)

	return cfg
}

// identifyDefaults identifies common patterns across libraries to extract as defaults.
func identifyDefaults(cfg *config.Config, tagFormats []string) {
	// Count transport occurrences
	transportCounts := make(map[string]int)
	for _, lib := range cfg.Libraries {
		if lib.Transport != "" {
			transportCounts[lib.Transport]++
		}
	}

	// Find most common transport (80%+ threshold)
	totalWithTransport := 0
	for _, count := range transportCounts {
		totalWithTransport += count
	}

	for transport, count := range transportCounts {
		if totalWithTransport > 0 && float64(count)/float64(totalWithTransport) >= 0.8 {
			cfg.Default.Generate.Transport = transport
			fmt.Fprintf(os.Stderr, "Identified default transport: %s (%.1f%% of libraries)\n", transport, 100*float64(count)/float64(totalWithTransport))
			break
		}
	}

	// Count tag format occurrences
	tagFormatCounts := make(map[string]int)
	for _, tagFormat := range tagFormats {
		tagFormatCounts[tagFormat]++
	}

	// Find most common tag format
	maxCount := 0
	var defaultTagFormat string
	for tagFormat, count := range tagFormatCounts {
		if count > maxCount {
			maxCount = count
			defaultTagFormat = tagFormat
		}
	}

	if defaultTagFormat != "" {
		cfg.Default.Release.TagFormat = defaultTagFormat
		fmt.Fprintf(os.Stderr, "Identified default tag_format: %s (%d libraries)\n", defaultTagFormat, maxCount)
	}
}

// filterKeepPatterns filters out unwanted patterns from the keep list.
func filterKeepPatterns(patterns []string, libraryName string) []string {
	// Patterns to exclude (with {name} placeholder for library name)
	excludePatterns := []string{
		"packages/{name}/CHANGELOG.md",
		"docs/CHANGELOG.md",
		"docs/README.rst",
		"samples/README.txt",
		"scripts/client-post-processing",
		"samples/snippets/README.rst",
		"tests/system",
	}

	// Replace {name} with actual library name
	var expandedExcludePatterns []string
	for _, pattern := range excludePatterns {
		expandedExcludePatterns = append(expandedExcludePatterns, strings.ReplaceAll(pattern, "{name}", libraryName))
	}

	var filtered []string
	for _, pattern := range patterns {
		exclude := false
		for _, excludePattern := range expandedExcludePatterns {
			if strings.Contains(pattern, excludePattern) {
				exclude = true
				break
			}
		}
		if !exclude {
			filtered = append(filtered, pattern)
		}
	}
	return filtered
}

// RepoConfig represents the structure of repo-config.yaml from generator-input.
type RepoConfig struct {
	Modules map[string]*RepoModule
}

// RepoModule represents a module entry in repo-config.yaml.
type RepoModule struct {
	ModulePathVersion           string         `yaml:"module_path_version"`
	DeleteGenerationOutputPaths []string       `yaml:"delete_generation_output_paths"`
	APIs                        []config.GoAPI `yaml:"apis"`
}

// parseRepoConfig parses repo-config.yaml from generator-input data.
func parseRepoConfig(generatorInput *LegacyGeneratorInputData) (*RepoConfig, error) {
	if generatorInput == nil || len(generatorInput.Files) == 0 {
		return nil, nil
	}

	// Look for repo-config.yaml
	content, ok := generatorInput.Files["repo-config.yaml"]
	if !ok {
		return nil, nil
	}

	// Parse the YAML structure
	var rawConfig struct {
		Modules []struct {
			Name                        string         `yaml:"name"`
			ModulePathVersion           string         `yaml:"module_path_version"`
			DeleteGenerationOutputPaths []string       `yaml:"delete_generation_output_paths"`
			APIs                        []config.GoAPI `yaml:"apis"`
		} `yaml:"modules"`
	}

	if err := yaml.Unmarshal(content, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repo-config.yaml: %w", err)
	}

	// Convert to map for easier lookup
	repoConfig := &RepoConfig{
		Modules: make(map[string]*RepoModule),
	}

	for _, module := range rawConfig.Modules {
		repoConfig.Modules[module.Name] = &RepoModule{
			ModulePathVersion:           module.ModulePathVersion,
			DeleteGenerationOutputPaths: module.DeleteGenerationOutputPaths,
			APIs:                        module.APIs,
		}
	}

	return repoConfig, nil
}
