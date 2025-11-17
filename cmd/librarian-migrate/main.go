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
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	// Only support Python and Go
	if language != "python" && language != "go" {
		return fmt.Errorf("unsupported language: %s (only python and go are supported)", language)
	}

	fmt.Fprintf(os.Stderr, "Detected language: %s\n", language)

	// Read all legacy configuration sources
	reader := &Reader{
		RepoPath:       repoPath,
		GoogleapisPath: googleapisPath,
	}

	fmt.Fprintf(os.Stderr, "Reading legacy configuration from %s...\n", repoPath)
	state, config, buildData, generatorInput, err := reader.ReadAll(language)
	if err != nil {
		return fmt.Errorf("failed to read legacy configuration: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d libraries in state.yaml\n", len(state.Libraries))

	// Merge all sources into config.Config
	fmt.Fprintf(os.Stderr, "Merging configuration sources...\n")
	cfg := merge(state, config, buildData, generatorInput, language)

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

	// Discover versions from packages
	if language == "python" {
		fmt.Fprintf(os.Stderr, "Discovering package versions...\n")
		versions, err := discoverVersions(repoPath, cfg)
		if err != nil {
			return fmt.Errorf("failed to discover versions: %w", err)
		}
		if len(versions) > 0 {
			cfg.Versions = versions
			fmt.Fprintf(os.Stderr, "Found %d package versions\n", len(versions))
		}
	}

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
// Only supports Python and Go.
func detectLanguage(repoPath string) (string, error) {
	// Extract language from repository name
	// Check python first, then go (check longer names first to avoid false matches)
	languages := []string{"python", "go"}

	lowerPath := strings.ToLower(repoPath)
	for _, lang := range languages {
		// Look for language in the final path component (repo name)
		if strings.Contains(lowerPath, "cloud-"+lang) || strings.HasSuffix(lowerPath, "-"+lang) {
			return lang, nil
		}
	}

	return "", fmt.Errorf("could not detect language from repository path: %s (only python and go are supported)", repoPath)
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
// Example: google/cloud/vision/v1 → google/cloud/vision.
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

// deriveExpectedName derives the expected library name from a service path based on language.
// The service path should NOT include the version (e.g., google/ai/generativelanguage, not google/ai/generativelanguage/v1).
func deriveExpectedName(servicePath, language string) string {
	if language == "python" {
		// Python: replace / with -
		return strings.ReplaceAll(servicePath, "/", "-")
	}
	// Go: use the service name (last component)
	parts := strings.Split(servicePath, "/")
	return parts[len(parts)-1]
}

// buildNameOverridesAndLibraries constructs name_overrides and libraries based on naming conventions.
// When one_library_per: service, all versions of a service are grouped into one library by default.
// For example, google/ai/generativelanguage includes v1, v1alpha, v1beta, v1beta2.
// Expected library names:
//   - Python: google-ai-generativelanguage (replace / with -)
//   - Go: generativelanguage (last component)
//
// Only list in name_overrides if actual name differs from expected.
// Only list in libraries if there's extra config OR APIs from multiple services.
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

	// Group APIs by service (base path) and track which library uses them
	serviceToAPIs := make(map[string][]string)
	serviceToLib := make(map[string]*config.Library)
	for _, apiPath := range googleapisAPIs {
		lib, exists := apiToLib[apiPath]
		if !exists {
			continue
		}
		servicePath := getBasePath(apiPath)
		serviceToAPIs[servicePath] = append(serviceToAPIs[servicePath], apiPath)
		serviceToLib[servicePath] = lib
	}

	// For each library, find which services it covers
	libToServices := make(map[string][]string)
	for servicePath, lib := range serviceToLib {
		libToServices[lib.Name] = append(libToServices[lib.Name], servicePath)
	}

	// Deduplicate services for each library
	for libName := range libToServices {
		services := libToServices[libName]
		uniqueServices := make(map[string]bool)
		for _, s := range services {
			uniqueServices[s] = true
		}
		libToServices[libName] = make([]string, 0, len(uniqueServices))
		for s := range uniqueServices {
			libToServices[libName] = append(libToServices[libName], s)
		}
		sort.Strings(libToServices[libName])
	}

	// Determine which libraries need to be in libraries section vs name_overrides
	for _, lib := range cfg.Libraries {
		services := libToServices[lib.Name]

		// Check if library has extra configuration
		hasExtraConfig := lib.Transport != "" ||
			lib.Python != nil ||
			len(lib.Keep) > 0 ||
			lib.Release != nil ||
			lib.Generate != nil

		// Always keep googleapis-common-protos in libraries section
		if lib.Name == "googleapis-common-protos" {
			// List all APIs for googleapis-common-protos
			var allAPIs []string
			for _, service := range services {
				allAPIs = append(allAPIs, serviceToAPIs[service]...)
			}
			sort.Strings(allAPIs)
			lib.APIs = allAPIs
			lib.API = ""
			newLibraries = append(newLibraries, lib)
			continue
		}

		// If library covers exactly one service
		if len(services) == 1 {
			servicePath := services[0]
			expectedName := deriveExpectedName(servicePath, language)

			// Check if name matches convention
			if lib.Name == expectedName {
				// Name matches convention
				if hasExtraConfig {
					// Has extra config - add to libraries without api/apis fields
					lib.API = ""
					lib.APIs = nil
					newLibraries = append(newLibraries, lib)
				}
				// else: auto-discovered, don't list anywhere
			} else {
				// Name doesn't match convention - add to name_overrides
				nameOverrides[servicePath] = lib.Name
				if hasExtraConfig {
					// Also add to libraries without api/apis fields
					lib.API = ""
					lib.APIs = nil
					newLibraries = append(newLibraries, lib)
				}
			}
		} else if len(services) > 1 {
			// Library covers multiple services - must list in libraries with explicit apis
			var allAPIs []string
			for _, service := range services {
				allAPIs = append(allAPIs, serviceToAPIs[service]...)
			}
			sort.Strings(allAPIs)
			lib.APIs = allAPIs
			lib.API = ""
			newLibraries = append(newLibraries, lib)

			// Check if name matches convention for primary service
			// Use first service alphabetically as primary
			primaryService := services[0]
			expectedName := deriveExpectedName(primaryService, language)
			if lib.Name != expectedName {
				// Name doesn't match convention - add to name_overrides
				nameOverrides[primaryService] = lib.Name
			}
		}
	}

	cfg.Libraries = newLibraries
	if len(nameOverrides) > 0 {
		cfg.NameOverrides = nameOverrides
	}
}

// discoverVersions discovers package versions from gapic_version.py files.
// For Python packages, it looks for packages/{library-name}/google/.../gapic_version.py
// and extracts the __version__ string.
func discoverVersions(repoPath string, cfg *config.Config) (map[string]string, error) {
	versions := make(map[string]string)
	packagesDir := filepath.Join(repoPath, "packages")

	// Check if packages directory exists
	if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
		return versions, nil
	}

	// Get all library names from the config
	libraryNames := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		libraryNames[lib.Name] = true
	}
	// Also check name_overrides values
	for _, name := range cfg.NameOverrides {
		libraryNames[name] = true
	}

	// Iterate through each package directory
	err := filepath.Walk(packagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only look for gapic_version.py files
		if info.IsDir() || info.Name() != "gapic_version.py" {
			return nil
		}

		// Extract the library name from the path
		// Path format: packages/{library-name}/google/.../gapic_version.py
		relPath, err := filepath.Rel(packagesDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) < 2 {
			return nil
		}

		libraryName := parts[0]

		// Only process libraries we know about
		if !libraryNames[libraryName] {
			return nil
		}

		// Skip if we already found a version for this library
		if _, exists := versions[libraryName]; exists {
			return nil
		}

		// Check if this is the main gapic_version.py (not a versioned one)
		// The main one is typically in a path without version suffix like _v1, _v1beta1, etc.
		// Example: google/cloud/secretmanager/gapic_version.py (not secretmanager_v1/gapic_version.py)
		// The directory containing gapic_version.py should not have a version suffix
		if len(parts) >= 2 {
			// Get the directory name that contains gapic_version.py
			dirName := parts[len(parts)-2]
			// Skip if directory name contains version pattern like _v1, _v1beta1, etc.
			if strings.Contains(dirName, "_v") {
				return nil
			}
		}

		// Read the version from the file
		version, err := readVersionFromFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read version from %s: %v\n", path, err)
			return nil
		}

		if version != "" && version != "0.0.0" {
			versions[libraryName] = version
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return versions, nil
}

// readVersionFromFile reads the __version__ string from a gapic_version.py file.
func readVersionFromFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Regexp to match: __version__ = "1.2.3"
	versionRegex := regexp.MustCompile(`__version__\s*=\s*"([^"]+)"`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}
