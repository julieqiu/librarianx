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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		repoPath       string
		outputPath     string
		googleapisPath string
	)

	flag.StringVar(&repoPath, "repo", "", "Path to the repository (required)")
	flag.StringVar(&outputPath, "output", "", "Output file path (default: stdout)")
	flag.StringVar(&googleapisPath, "googleapis", "", "Path to googleapis repository for BUILD.bazel files")
	flag.Parse()

	if repoPath == "" {
		return fmt.Errorf("-repo flag is required")
	}

	// Detect language from repository
	language, err := detectLanguage(repoPath)
	if err != nil {
		return fmt.Errorf("failed to detect language: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Detected language: %s\n", language)

	// Read all legacy configuration sources
	reader := &Reader{
		RepoPath:       repoPath,
		GoogleapisPath: googleapisPath,
	}

	fmt.Fprintf(os.Stderr, "Reading legacy configuration from %s...\n", repoPath)
	state, config, buildData, generatorInput, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read legacy configuration: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d libraries in state.yaml\n", len(state.Libraries))

	// Merge all sources into config.Config
	fmt.Fprintf(os.Stderr, "Merging configuration sources...\n")
	cfg, err := merge(state, config, buildData, generatorInput, language)
	if err != nil {
		return fmt.Errorf("failed to merge configuration: %w", err)
	}

	// Deduplicate fields that match defaults
	fmt.Fprintf(os.Stderr, "Deduplicating library-specific fields...\n")
	deduplicate(cfg)

	// Discover all APIs from googleapis
	fmt.Fprintf(os.Stderr, "Discovering APIs from googleapis...\n")
	googleapisAPIs, err := discoverGoogleapisAPIs(googleapisPath)
	if err != nil {
		return fmt.Errorf("failed to discover googleapis APIs: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Found %d APIs in googleapis\n", len(googleapisAPIs))

	// Build name_overrides and libraries based on naming conventions
	fmt.Fprintf(os.Stderr, "Building name_overrides and libraries...\n")
	buildNameOverridesAndLibraries(cfg, googleapisAPIs, language)

	// Sort for reproducibility
	fmt.Fprintf(os.Stderr, "Sorting for reproducibility...\n")
	sortConfig(cfg)

	// Write output
	if outputPath == "" {
		// Write to stdout
		enc := yaml.NewEncoder(os.Stdout)
		enc.SetIndent(2)
		defer enc.Close()

		if err := enc.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode config: %w", err)
		}
	} else {
		// Write to file
		fmt.Fprintf(os.Stderr, "Writing output to %s...\n", outputPath)
		if err := cfg.Write(outputPath); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		// Run yamlfmt if available
		if err := runYamlfmt(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: yamlfmt failed: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "Migration complete!\n")
	}

	return nil
}

// deduplicate removes library-specific fields that match the defaults.
func deduplicate(cfg *config.Config) {
	defaultTransport := ""
	if cfg.Default != nil && cfg.Default.Generate != nil {
		defaultTransport = cfg.Default.Generate.Transport
	}

	for _, lib := range cfg.Libraries {
		// Remove transport if it matches the default
		if defaultTransport != "" && lib.Transport == defaultTransport {
			lib.Transport = ""
		}

		// Simplify API/APIs field
		if len(lib.APIs) == 1 {
			lib.API = lib.APIs[0]
			lib.APIs = nil
		}

		// Remove empty Python section
		if lib.Python != nil && len(lib.Python.OptArgs) == 0 {
			lib.Python = nil
		}
	}
}

// runYamlfmt runs yamlfmt on the output file if the command is available.
func runYamlfmt(path string) error {
	_, err := exec.LookPath("yamlfmt")
	if err != nil {
		// yamlfmt not available, skip
		return nil
	}

	cmd := exec.Command("yamlfmt", path)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// detectLanguage detects the programming language from the repository path.
func detectLanguage(repoPath string) (string, error) {
	// Extract language from repository name
	// Check longer names first to avoid false matches (e.g., "go" in "googleapis")
	languages := []string{"python", "rust", "dart", "java", "node", "ruby", "php", "go"}

	lowerPath := strings.ToLower(repoPath)
	for _, lang := range languages {
		// Look for language in the final path component (repo name)
		if strings.Contains(lowerPath, "cloud-"+lang) || strings.HasSuffix(lowerPath, "-"+lang) {
			return lang, nil
		}
	}

	return "", fmt.Errorf("could not detect language from repository path: %s", repoPath)
}

// sortConfig sorts all lists in the config for reproducible output.
func sortConfig(cfg *config.Config) {
	// Sort libraries by name
	sort.Slice(cfg.Libraries, func(i, j int) bool {
		return cfg.Libraries[i].Name < cfg.Libraries[j].Name
	})

	// Sort fields within each library
	for _, lib := range cfg.Libraries {
		sort.Strings(lib.APIs)
		sort.Strings(lib.Keep)
		if lib.Python != nil {
			sort.Strings(lib.Python.OptArgs)
		}
	}
}

// discoverGoogleapisAPIs finds all API paths in the googleapis repository.
func discoverGoogleapisAPIs(googleapisPath string) ([]string, error) {
	if googleapisPath == "" {
		return nil, fmt.Errorf("googleapis path is required")
	}

	var apis []string
	err := filepath.Walk(googleapisPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Look for BUILD.bazel files
		if !info.IsDir() && info.Name() == "BUILD.bazel" {
			// Get the directory path relative to googleapis root
			dir := filepath.Dir(path)
			relPath, err := filepath.Rel(googleapisPath, dir)
			if err != nil {
				return err
			}
			// Only include paths under google/
			if strings.HasPrefix(relPath, "google/") || strings.HasPrefix(relPath, "grafeas/") {
				apis = append(apis, relPath)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return apis, nil
}

// getBasePath extracts the base path without the version.
// Example: google/cloud/vision/v1 → google/cloud/vision
func getBasePath(apiPath string) string {
	parts := strings.Split(apiPath, "/")
	// Remove version component (last part if it starts with 'v' followed by a digit)
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if len(lastPart) > 0 && lastPart[0] == 'v' && len(lastPart) > 1 && (lastPart[1] >= '0' && lastPart[1] <= '9') {
			parts = parts[:len(parts)-1]
		}
	}
	return strings.Join(parts, "/")
}

// deriveExpectedName derives the expected library name from an API path based on language.
func deriveExpectedName(apiPath, language string) string {
	if language == "python" {
		// Python: replace / with -
		return strings.ReplaceAll(apiPath, "/", "-")
	}
	// Go: use the service name (last non-version component)
	parts := strings.Split(apiPath, "/")
	// Work backwards to find the first non-version component
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// Skip version components (v1, v1beta1, v2alpha, etc.)
		if !strings.HasPrefix(part, "v") && !strings.Contains(part, "v1") && !strings.Contains(part, "v2") {
			return part
		}
		// If it's a version-like string, skip it
		if len(part) > 0 && part[0] == 'v' && len(part) > 1 && (part[1] >= '0' && part[1] <= '9') {
			continue
		}
		return part
	}
	return parts[len(parts)-1]
}

// buildNameOverridesAndLibraries constructs name_overrides and libraries based on naming conventions.
func buildNameOverridesAndLibraries(cfg *config.Config, googleapisAPIs []string, language string) {
	nameOverrides := make(map[string]string)
	var newLibraries []*config.Library

	// Create a map of API path to library for quick lookup
	apiToLib := make(map[string]*config.Library)
	for _, lib := range cfg.Libraries {
		if lib.API != "" {
			apiToLib[lib.API] = lib
		}
		for _, api := range lib.APIs {
			apiToLib[api] = lib
		}
	}

	// Group APIs by library name to detect multi-API libraries
	libNameToAPIs := make(map[string][]string)
	for _, apiPath := range googleapisAPIs {
		lib, exists := apiToLib[apiPath]
		if !exists {
			continue
		}
		libNameToAPIs[lib.Name] = append(libNameToAPIs[lib.Name], apiPath)
	}

	// Determine which libraries need to be in libraries section vs name_overrides
	libsToKeep := make(map[string]bool)

	// First pass: identify all libraries with extra configuration
	for _, lib := range cfg.Libraries {
		hasExtraConfig := lib.Transport != "" ||
			lib.Python != nil ||
			len(lib.Keep) > 0 ||
			lib.Release != nil ||
			lib.Generate != nil

		if lib.Name == "googleapis-common-protos" || hasExtraConfig {
			libsToKeep[lib.Name] = true
		}
	}

	// Second pass: identify multi-API libraries that need explicit listing
	for libName, apis := range libNameToAPIs {
		// If multiple APIs map to the same library
		if len(apis) > 1 {
			// Check if all APIs share the same base path
			basePath := getBasePath(apis[0])
			allSameBase := true
			for _, api := range apis {
				if getBasePath(api) != basePath {
					allSameBase = false
					break
				}
			}

			// If APIs come from different base paths, need to list explicitly
			if !allSameBase {
				libsToKeep[libName] = true
				continue
			}

			// All APIs share the same base - check if library name follows convention
			expectedName := deriveExpectedName(basePath, language)
			if libName != expectedName {
				// Doesn't follow convention - add to name_overrides
				nameOverrides[basePath] = libName
			}
			continue
		}

		// Single API - check if it follows naming convention
		apiPath := apis[0]
		expectedName := deriveExpectedName(apiPath, language)
		if libName != expectedName {
			// Doesn't follow convention - add to name_overrides
			nameOverrides[apiPath] = libName
		}
	}

	// Keep libraries that need to be in libraries section
	for _, lib := range cfg.Libraries {
		if lib.Name == "googleapis-common-protos" || libsToKeep[lib.Name] {
			// Reconstruct API/APIs fields only when necessary
			if lib.Name == "googleapis-common-protos" {
				// Always list APIs for googleapis-common-protos
				lib.APIs = libNameToAPIs[lib.Name]
				lib.API = ""
			} else if len(libNameToAPIs[lib.Name]) > 1 {
				// Multi-API library - check if APIs should be listed
				apis := libNameToAPIs[lib.Name]
				basePath := getBasePath(apis[0])
				allSameBase := true
				for _, api := range apis {
					if getBasePath(api) != basePath {
						allSameBase = false
						break
					}
				}

				// Only list APIs if from different base paths
				if !allSameBase {
					lib.APIs = libNameToAPIs[lib.Name]
					lib.API = ""
				} else {
					// APIs will be auto-discovered - don't list them
					lib.API = ""
					lib.APIs = nil
				}
			} else {
				// Single API library with extra config - remove API fields
				lib.API = ""
				lib.APIs = nil
			}
			newLibraries = append(newLibraries, lib)
		}
	}

	cfg.Libraries = newLibraries
	if len(nameOverrides) > 0 {
		cfg.NameOverrides = nameOverrides
	}
}
