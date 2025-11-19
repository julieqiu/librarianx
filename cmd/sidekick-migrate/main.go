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
	"path/filepath"
	"sort"
	"strings"

	"github.com/julieqiu/librarianx/internal/config"
	"github.com/pelletier/go-toml/v2"
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

	flag.StringVar(&repoPath, "repo", "", "Path to the google-cloud-rust repository (required)")
	flag.StringVar(&outputPath, "output", "", "Output file path (default: stdout)")
	flag.StringVar(&googleapisPath, "googleapis", "", "Path to googleapis repository")
	flag.Parse()

	if repoPath == "" {
		return fmt.Errorf("-repo flag is required")
	}

	fmt.Fprintf(os.Stderr, "Reading sidekick.toml files from %s...\n", repoPath)

	// Read existing librarian.yaml to preserve library-specific configuration and versions
	existingConfig, err := readExistingLibrarian(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read existing librarian.yaml: %w", err)
	}

	// Read root .sidekick.toml for defaults
	rootDefaults, err := readRootSidekick(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read root .sidekick.toml: %w", err)
	}

	// Find all .sidekick.toml files
	sidekickFiles, err := findSidekickFiles(repoPath)
	if err != nil {
		return fmt.Errorf("failed to find sidekick.toml files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d sidekick.toml files\n", len(sidekickFiles))

	// Read all sidekick.toml files
	libraries, err := readSidekickFiles(sidekickFiles)
	if err != nil {
		return fmt.Errorf("failed to read sidekick files: %w", err)
	}

	// Build config
	cfg := buildConfig(libraries, googleapisPath, rootDefaults, existingConfig)

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
		if err := cfg.Write(outputPath); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Wrote config to %s\n", outputPath)
	}

	return nil
}

// RootSidekickConfig represents the structure of the root .sidekick.toml file.
type RootSidekickConfig struct {
	Codec struct {
		DisabledRustdocWarnings string            `toml:"disabled-rustdoc-warnings"`
		Packages                map[string]string `toml:",remain"`
	} `toml:"codec"`
	Release struct {
		Remote string `toml:"remote"`
		Branch string `toml:"branch"`
	} `toml:"release"`
}

// SidekickConfig represents the structure of a .sidekick.toml file.
type SidekickConfig struct {
	General struct {
		SpecificationSource string `toml:"specification-source"`
		ServiceConfig       string `toml:"service-config"`
	} `toml:"general"`
	Codec struct {
		Version       string `toml:"version"`
		CopyrightYear string `toml:"copyright-year"`
	} `toml:"codec"`
}

// RootDefaults contains defaults extracted from root .sidekick.toml.
type RootDefaults struct {
	DisabledRustdocWarnings []string
	PackageDependencies     []*config.RustPackageDependency
	Remote                  string
	Branch                  string
}

// ExistingConfig contains data from existing librarian.yaml.
type ExistingConfig struct {
	Libraries map[string]*config.Library
	Versions  map[string]string
}

// readExistingLibrarian reads the existing librarian.yaml to preserve library-specific configuration and versions.
func readExistingLibrarian(repoPath string) (*ExistingConfig, error) {
	librarianPath := filepath.Join(repoPath, "librarian.yaml")
	data, err := os.ReadFile(librarianPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing librarian.yaml, return empty config
			return &ExistingConfig{
				Libraries: make(map[string]*config.Library),
				Versions:  make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Build map of libraries by name
	libraries := make(map[string]*config.Library)
	for _, lib := range cfg.Libraries {
		libraries[lib.Name] = lib
	}

	return &ExistingConfig{
		Libraries: libraries,
		Versions:  cfg.Versions,
	}, nil
}

// readRootSidekick reads the root .sidekick.toml file and extracts defaults.
func readRootSidekick(repoPath string) (*RootDefaults, error) {
	rootPath := filepath.Join(repoPath, ".sidekick.toml")
	data, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, err
	}

	// Parse as generic map to handle the dynamic package keys
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	defaults := &RootDefaults{}

	// Extract codec section
	if codec, ok := raw["codec"].(map[string]interface{}); ok {
		// Parse disabled warnings
		if warnings, ok := codec["disabled-rustdoc-warnings"].(string); ok {
			defaults.DisabledRustdocWarnings = strings.Split(warnings, ",")
		}

		// Parse package dependencies
		for key, value := range codec {
			if !strings.HasPrefix(key, "package:") {
				continue
			}
			pkgName := strings.TrimPrefix(key, "package:")
			pkgSpec := value.(string)

			dep := parsePackageDependency(pkgName, pkgSpec)
			if dep != nil {
				defaults.PackageDependencies = append(defaults.PackageDependencies, dep)
			}
		}
	}

	// Extract release section
	if release, ok := raw["release"].(map[string]interface{}); ok {
		if remote, ok := release["remote"].(string); ok {
			defaults.Remote = remote
		}
		if branch, ok := release["branch"].(string); ok {
			defaults.Branch = branch
		}
	}

	// Sort package dependencies by name
	sort.Slice(defaults.PackageDependencies, func(i, j int) bool {
		return defaults.PackageDependencies[i].Name < defaults.PackageDependencies[j].Name
	})

	return defaults, nil
}

// parsePackageDependency parses a package dependency spec.
// Format: "package=name,source=path,force-used=true,used-if=condition".
func parsePackageDependency(name, spec string) *config.RustPackageDependency {
	dep := &config.RustPackageDependency{
		Name: name,
	}

	parts := strings.Split(spec, ",")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}
		key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])

		switch key {
		case "package":
			dep.Package = value
		case "source":
			dep.Source = value
		case "force-used":
			dep.ForceUsed = value == "true"
		case "used-if":
			dep.UsedIf = value
		case "feature":
			dep.Feature = value
		}
	}

	return dep
}

// findSidekickFiles finds all .sidekick.toml files in the repository.
func findSidekickFiles(repoPath string) ([]string, error) {
	var files []string

	generatedPath := filepath.Join(repoPath, "src", "generated")
	err := filepath.Walk(generatedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == ".sidekick.toml" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// CargoConfig represents relevant fields from Cargo.toml.
type CargoConfig struct {
	Package struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"package"`
}

// readSidekickFiles reads all sidekick.toml files and extracts library information.
func readSidekickFiles(files []string) (map[string]*config.Library, error) {
	libraries := make(map[string]*config.Library)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		var sidekick SidekickConfig
		if err := toml.Unmarshal(data, &sidekick); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", file, err)
		}

		// Get API path
		apiPath := sidekick.General.SpecificationSource
		if apiPath == "" {
			continue
		}

		// Read Cargo.toml in the same directory to get the actual library name
		dir := filepath.Dir(file)
		cargoPath := filepath.Join(dir, "Cargo.toml")
		cargoData, err := os.ReadFile(cargoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", cargoPath, err)
		}

		var cargo CargoConfig
		if err := toml.Unmarshal(cargoData, &cargo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", cargoPath, err)
		}

		libraryName := cargo.Package.Name
		if libraryName == "" {
			continue
		}

		// Create or update library
		lib, exists := libraries[libraryName]
		if !exists {
			lib = &config.Library{
				Name: libraryName,
			}
			libraries[libraryName] = lib
		}

		// Add API path
		if lib.API == "" && len(lib.APIs) == 0 {
			lib.API = apiPath
		} else if lib.API != "" && lib.API != apiPath {
			// Convert to multi-API library
			lib.APIs = []string{lib.API, apiPath}
			lib.API = ""
		} else if len(lib.APIs) > 0 && !contains(lib.APIs, apiPath) {
			lib.APIs = append(lib.APIs, apiPath)
		}

		// Set version from Cargo.toml (more authoritative than sidekick)
		if cargo.Package.Version != "" {
			lib.Version = cargo.Package.Version
		} else if sidekick.Codec.Version != "" && lib.Version == "" {
			lib.Version = sidekick.Codec.Version
		}
	}

	return libraries, nil
}

// deriveLibraryName derives a library name from an API path.
// For Rust: google/cloud/secretmanager/v1 -> google-cloud-secretmanager-v1.
func deriveLibraryName(apiPath string) string {
	return strings.ReplaceAll(apiPath, "/", "-")
}

// buildConfig builds the complete config from libraries.
func buildConfig(libraries map[string]*config.Library, googleapisPath string, rootDefaults *RootDefaults, existingConfig *ExistingConfig) *config.Config {
	cfg := &config.Config{
		Version:  "v1",
		Language: "rust",
		Default: &config.Default{
			Output: "src/generated/",
			Generate: &config.DefaultGenerate{
				All:           true,
				OneLibraryPer: "channel",
				ReleaseLevel:  "stable",
			},
			Release: &config.DefaultRelease{
				TagFormat: "{name}/v{version}",
				Remote:    rootDefaults.Remote,
				Branch:    rootDefaults.Branch,
			},
			Rust: &config.RustDefault{
				DisabledRustdocWarnings: rootDefaults.DisabledRustdocWarnings,
				PackageDependencies:     rootDefaults.PackageDependencies,
			},
		},
	}

	// Add googleapis source if provided
	if googleapisPath != "" {
		// Try to get the current commit
		commit, err := getGitCommit(googleapisPath)
		if err == nil {
			cfg.Sources = &config.Sources{
				Googleapis: &config.Source{
					Commit: commit,
				},
			}
		}
	}

	// Convert libraries map to sorted slice, applying new schema logic
	var libList []*config.Library
	versions := make(map[string]string)

	// Start with existing versions
	for name, version := range existingConfig.Versions {
		versions[name] = version
	}

	for name, lib := range libraries {
		// Track versions for ALL libraries (override existing if present)
		if lib.Version != "" {
			versions[name] = lib.Version
		}

		// Merge with existing library configuration
		if existingLib, exists := existingConfig.Libraries[name]; exists {
			// Preserve existing configuration fields
			if existingLib.Generate != nil {
				lib.Generate = existingLib.Generate
			}
			if existingLib.Publish != nil {
				lib.Publish = existingLib.Publish
			}
			if existingLib.CopyrightYear != "" {
				lib.CopyrightYear = existingLib.CopyrightYear
			}
			if existingLib.Rust != nil {
				if lib.Rust == nil {
					lib.Rust = &config.RustCrate{}
				}
				// Merge Rust-specific config
				if existingLib.Rust.PerServiceFeatures {
					lib.Rust.PerServiceFeatures = true
				}
				if len(existingLib.Rust.DisabledRustdocWarnings) > 0 {
					lib.Rust.DisabledRustdocWarnings = existingLib.Rust.DisabledRustdocWarnings
				}
				if len(existingLib.Rust.PackageDependencies) > 0 {
					lib.Rust.PackageDependencies = existingLib.Rust.PackageDependencies
				}
				if existingLib.Rust.GenerateSetterSamples {
					lib.Rust.GenerateSetterSamples = true
				}
				if len(existingLib.Rust.PaginationOverrides) > 0 {
					lib.Rust.PaginationOverrides = existingLib.Rust.PaginationOverrides
				}
				if existingLib.Rust.NameOverrides != "" {
					lib.Rust.NameOverrides = existingLib.Rust.NameOverrides
				}
			}
		}

		// Get the API path for this library
		apiPath := lib.API
		if len(lib.APIs) > 0 {
			apiPath = lib.APIs[0]
		}

		// Derive expected library name from API path
		// For Rust with one_library_per: channel, expected name is the API path with / replaced by -
		expectedName := deriveLibraryName(apiPath)

		// Check if library has extra configuration beyond just name/api/version
		hasExtraConfig := lib.Generate != nil || lib.Publish != nil || lib.CopyrightYear != "" ||
			(lib.Rust != nil && (lib.Rust.PerServiceFeatures || len(lib.Rust.DisabledRustdocWarnings) > 0 ||
				len(lib.Rust.PackageDependencies) > 0 || lib.Rust.GenerateSetterSamples ||
				len(lib.Rust.PaginationOverrides) > 0 || lib.Rust.NameOverrides != ""))

		// Only include in libraries section if:
		// 1. Name doesn't match expected naming convention (name override)
		// 2. Library has extra configuration
		// 3. Library spans multiple APIs
		nameMatchesConvention := lib.Name == expectedName

		if !nameMatchesConvention || hasExtraConfig || len(lib.APIs) > 1 {
			// Clear version from library (it goes in versions section)
			libCopy := *lib
			libCopy.Version = ""
			libList = append(libList, &libCopy)
		}
	}

	// Add libraries from existing config that weren't found in sidekick files
	for name, existingLib := range existingConfig.Libraries {
		if _, found := libraries[name]; !found {
			// This library exists in the config but not in sidekick files
			// Preserve it as-is
			libCopy := *existingLib
			libCopy.Version = "" // Version goes in versions section
			libList = append(libList, &libCopy)
		}
	}

	// Sort libraries by name
	sort.Slice(libList, func(i, j int) bool {
		return libList[i].Name < libList[j].Name
	})

	cfg.Libraries = libList
	cfg.Versions = versions

	return cfg
}

// getGitCommit gets the current git commit hash from a repository.
func getGitCommit(repoPath string) (string, error) {
	// Read .git/HEAD to get current commit
	headPath := filepath.Join(repoPath, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", err
	}

	head := strings.TrimSpace(string(data))
	if strings.HasPrefix(head, "ref: ") {
		// HEAD points to a branch
		refPath := strings.TrimPrefix(head, "ref: ")
		refFullPath := filepath.Join(repoPath, ".git", refPath)
		commitData, err := os.ReadFile(refFullPath)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(commitData)), nil
	}

	// HEAD is a direct commit hash
	return head, nil
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
