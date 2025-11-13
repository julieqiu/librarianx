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

// Command convert-sidekick-toml converts sidekick.toml configuration to librarian.yaml format.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// SidekickRootConfig represents the root .sidekick.toml configuration file structure.
type SidekickRootConfig struct {
	General GeneralConfig  `toml:"general"`
	Source  SourceConfig   `toml:"source"`
	Codec   map[string]any `toml:"codec"`
	Release ReleaseConfig  `toml:"release"`
}

// SidekickLibraryConfig represents a library-level .sidekick.toml configuration.
type SidekickLibraryConfig struct {
	General   LibraryGeneralConfig `toml:"general"`
	Source    map[string]any       `toml:"source"`
	Discovery map[string]any       `toml:"discovery"`
	Codec     map[string]any       `toml:"codec"`
}

// GeneralConfig represents the [general] section of root config.
type GeneralConfig struct {
	Language            string   `toml:"language"`
	SpecificationFormat string   `toml:"specification-format"`
	IgnoredDirectories  []string `toml:"ignored-directories"`
}

// LibraryGeneralConfig represents the [general] section of library config.
type LibraryGeneralConfig struct {
	Language            string `toml:"language"`
	SpecificationFormat string `toml:"specification-format"`
	SpecificationSource string `toml:"specification-source"`
	ServiceConfig       string `toml:"service-config"`
}

// SourceConfig represents the [source] section.
type SourceConfig struct {
	Roots                    string `toml:"roots"`
	ShowcaseExtractedName    string `toml:"showcase-extracted-name"`
	ShowcaseRoot             string `toml:"showcase-root"`
	ShowcaseSHA256           string `toml:"showcase-sha256"`
	GoogleapisRoot           string `toml:"googleapis-root"`
	GoogleapisSHA256         string `toml:"googleapis-sha256"`
	DiscoveryExtractedName   string `toml:"discovery-extracted-name"`
	DiscoveryRoot            string `toml:"discovery-root"`
	DiscoverySHA256          string `toml:"discovery-sha256"`
	ProtobufSrcExtractedName string `toml:"protobuf-src-extracted-name"`
	ProtobufSrcRoot          string `toml:"protobuf-src-root"`
	ProtobufSrcSHA256        string `toml:"protobuf-src-sha256"`
	ProtobufSrcSubdir        string `toml:"protobuf-src-subdir"`
	ConformanceExtractedName string `toml:"conformance-extracted-name"`
	ConformanceRoot          string `toml:"conformance-root"`
	ConformanceSHA256        string `toml:"conformance-sha256"`
}

// ReleaseConfig represents the [release] section.
type ReleaseConfig struct {
	Remote         string                  `toml:"remote"`
	Branch         string                  `toml:"branch"`
	IgnoredChanges []string                `toml:"ignored-changes"`
	Tools          map[string][]ToolConfig `toml:"tools"`
	PreInstalled   map[string]string       `toml:"pre-installed"`
}

// ToolConfig represents a tool configuration.
type ToolConfig struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

func main() {
	rootDir := flag.String("root", "", "Root directory of google-cloud-rust repository")
	outputPath := flag.String("output", "", "Path to output librarian.yaml file")
	flag.Parse()

	if *rootDir == "" || *outputPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -root <google-cloud-rust-dir> -output <librarian.yaml>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := convertAll(*rootDir, *outputPath); err != nil {
		log.Fatalf("conversion failed: %v", err)
	}

	fmt.Printf("Successfully converted to %s\n", *outputPath)
}

func convertAll(rootDir, outputPath string) error {
	// Read root .sidekick.toml
	rootConfigPath := filepath.Join(rootDir, ".sidekick.toml")
	rootConfig, err := readRootConfig(rootConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read root config: %w", err)
	}

	// Build sources
	var sources config.Sources
	if rootConfig.Source.GoogleapisRoot != "" {
		sources.Googleapis = &config.Source{
			URL:    rootConfig.Source.GoogleapisRoot,
			SHA256: rootConfig.Source.GoogleapisSHA256,
		}
	}
	if rootConfig.Source.ShowcaseRoot != "" {
		sources.Showcase = &config.Source{
			URL:           rootConfig.Source.ShowcaseRoot,
			SHA256:        rootConfig.Source.ShowcaseSHA256,
			ExtractedName: rootConfig.Source.ShowcaseExtractedName,
		}
	}
	if rootConfig.Source.DiscoveryRoot != "" {
		sources.Discovery = &config.Source{
			URL:           rootConfig.Source.DiscoveryRoot,
			SHA256:        rootConfig.Source.DiscoverySHA256,
			ExtractedName: rootConfig.Source.DiscoveryExtractedName,
		}
	}
	if rootConfig.Source.ProtobufSrcRoot != "" {
		sources.ProtobufSrc = &config.Source{
			URL:           rootConfig.Source.ProtobufSrcRoot,
			SHA256:        rootConfig.Source.ProtobufSrcSHA256,
			ExtractedName: rootConfig.Source.ProtobufSrcExtractedName,
			Subdir:        rootConfig.Source.ProtobufSrcSubdir,
		}
	}
	if rootConfig.Source.ConformanceRoot != "" {
		sources.Conformance = &config.Source{
			URL:           rootConfig.Source.ConformanceRoot,
			SHA256:        rootConfig.Source.ConformanceSHA256,
			ExtractedName: rootConfig.Source.ConformanceExtractedName,
		}
	}

	// Determine output directory and one_library_per based on language
	output := "src/generated/"
	oneLibraryPer := "version"
	if rootConfig.General.Language == "dart" {
		output = "generated/"
	}

	// Extract default Rust settings
	var rustDefaults *config.RustDefaults
	if rootConfig.General.Language == "rust" {
		rustDefaults = &config.RustDefaults{}

		// Extract default package dependencies
		if deps := extractPackageDependencies(rootConfig.Codec); len(deps) > 0 {
			rustDefaults.PackageDependencies = deps
		}

		// Extract default disabled rustdoc warnings
		if warnings := getStringSlice(rootConfig.Codec, "disabled-rustdoc-warnings"); len(warnings) > 0 {
			rustDefaults.DisabledRustdocWarnings = warnings
		}
	}

	// Create librarian config (simple format)
	libConfig := &config.Config{
		Version:  "v1",
		Language: rootConfig.General.Language,
		Container: &config.Container{
			Image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/rust-librarian-generator",
			Tag:   "latest",
		},
		Sources: sources,
		Defaults: &config.Defaults{
			Output:        output,
			OneLibraryPer: oneLibraryPer,
			ReleaseLevel:  getString(rootConfig.Codec, "release-level"),
			Rust:          rustDefaults,
		},
		Release: &config.Release{
			TagFormat:      "{name}/v{version}",
			Remote:         rootConfig.Release.Remote,
			Branch:         rootConfig.Release.Branch,
			IgnoredChanges: rootConfig.Release.IgnoredChanges,
		},
		Libraries: []config.LibraryEntry{
			{Name: "*"}, // Add wildcard to generate everything
		},
	}

	// Find all library .sidekick.toml files
	var libraryConfigs []string
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == ".sidekick.toml" && path != rootConfigPath {
			libraryConfigs = append(libraryConfigs, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find library configs: %w", err)
	}

	// Sort for consistent output
	sort.Strings(libraryConfigs)

	// Process each library config
	for _, configPath := range libraryConfigs {
		entry, err := processLibraryConfig(configPath, rootDir, rootConfig.General.Language)
		if err != nil {
			log.Printf("Warning: failed to process %s: %v", configPath, err)
			continue
		}
		if entry != nil {
			libConfig.Libraries = append(libConfig.Libraries, *entry)
		}
	}

	// Write librarian.yaml
	data, err := yaml.Marshal(libConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func readRootConfig(path string) (*SidekickRootConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg SidekickRootConfig
	if err := toml.Unmarshal(contents, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func processLibraryConfig(configPath, rootDir, language string) (*config.LibraryEntry, error) {
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var libConfig SidekickLibraryConfig
	if err := toml.Unmarshal(contents, &libConfig); err != nil {
		return nil, err
	}

	// Skip if no specification source
	if libConfig.General.SpecificationSource == "" {
		return nil, nil
	}

	// API path comes from specification-source
	apiPath := libConfig.General.SpecificationSource

	// Derive library name from API path using Rust conventions (one_library_per: version)
	// Example: google/longrunning -> google-longrunning
	// Example: google/cloud/secretmanager/v1 -> google-cloud-secretmanager-v1
	libraryName := strings.ReplaceAll(apiPath, "/", "-")

	// Derive filesystem path from directory relative to root
	relPath, err := filepath.Rel(rootDir, filepath.Dir(configPath))
	if err != nil {
		return nil, err
	}

	// All library paths should have trailing slash to indicate they're directories
	libraryPath := relPath
	if !strings.HasSuffix(libraryPath, "/") {
		libraryPath = libraryPath + "/"
	}

	// Build library config if there are any overrides
	// Always add API field for generated libraries
	cfg := &config.LibraryConfig{
		API: apiPath,
	}

	// Add path override if it differs from expected location
	// Expected: defaults.output (src/generated/) + API path pattern
	// We always add path since we're converting from existing structure
	cfg.Path = libraryPath

	// Extract Rust-specific configuration
	if language == "rust" {
		rustCfg := extractRustLibraryConfig(libConfig.Codec, libConfig.Source)
		if rustCfg != nil {
			cfg.Rust = rustCfg
		}
	}

	// Extract Dart-specific configuration
	if language == "dart" {
		dartCfg := extractDartLibraryConfig(libConfig.Codec)
		if dartCfg != nil {
			cfg.Dart = dartCfg
		}
	}

	// Return library entry with name as identifier
	entry := &config.LibraryEntry{
		Name:   libraryName,
		Config: cfg,
	}

	return entry, nil
}

func extractRustLibraryConfig(codec map[string]any, source map[string]any) *config.RustLibrary {
	rust := &config.RustLibrary{}
	hasConfig := false

	if perServiceFeatures := getBoolFromMap(codec, "per-service-features"); perServiceFeatures {
		rust.PerServiceFeatures = true
		hasConfig = true
	}

	if modulePath := getStringFromMap(codec, "module-path"); modulePath != "" {
		rust.ModulePath = modulePath
		hasConfig = true
	}

	if templateOverride := getStringFromMap(codec, "template-override"); templateOverride != "" {
		rust.TemplateOverride = templateOverride
		hasConfig = true
	}

	if titleOverride := getStringFromMap(codec, "title-override"); titleOverride != "" {
		rust.TitleOverride = titleOverride
		hasConfig = true
	}

	if descriptionOverride := getStringFromMap(source, "description-override"); descriptionOverride != "" {
		rust.DescriptionOverride = descriptionOverride
		hasConfig = true
	}

	// included-ids can be in [source] or [codec] section
	if includedIds := getStringSlice(source, "included-ids"); len(includedIds) > 0 {
		rust.IncludedIds = includedIds
		hasConfig = true
	} else if includedIds := getStringSlice(codec, "included-ids"); len(includedIds) > 0 {
		rust.IncludedIds = includedIds
		hasConfig = true
	}

	if packageNameOverride := getStringFromMap(codec, "package-name-override"); packageNameOverride != "" {
		rust.PackageNameOverride = packageNameOverride
		hasConfig = true
	}

	if rootName := getStringFromMap(codec, "root-name"); rootName != "" {
		rust.RootName = rootName
		hasConfig = true
	}

	if roots := getStringSlice(codec, "roots"); len(roots) > 0 {
		rust.Roots = roots
		hasConfig = true
	}

	if defaultFeatures := getStringSlice(codec, "default-features"); len(defaultFeatures) > 0 {
		rust.DefaultFeatures = defaultFeatures
		hasConfig = true
	}

	if extraModules := getStringSlice(codec, "extra-modules"); len(extraModules) > 0 {
		rust.ExtraModules = extraModules
		hasConfig = true
	}

	if includeList := getStringSlice(codec, "include-list"); len(includeList) > 0 {
		rust.IncludeList = includeList
		hasConfig = true
	}

	if skippedIds := getStringSlice(codec, "skipped-ids"); len(skippedIds) > 0 {
		rust.SkippedIds = skippedIds
		hasConfig = true
	}

	if nameOverrides := getStringFromMap(codec, "name-overrides"); nameOverrides != "" {
		rust.NameOverrides = nameOverrides
		hasConfig = true
	}

	if deps := extractPackageDependencies(codec); len(deps) > 0 {
		rust.PackageDependencies = deps
		hasConfig = true
	}

	if hasVeneer := getBoolFromMap(codec, "has-veneer"); hasVeneer {
		rust.HasVeneer = true
		hasConfig = true
	}

	if routingRequired := getBoolFromMap(codec, "routing-required"); routingRequired {
		rust.RoutingRequired = true
		hasConfig = true
	}

	if includeGrpcOnlyMethods := getBoolFromMap(codec, "include-grpc-only-methods"); includeGrpcOnlyMethods {
		rust.IncludeGrpcOnlyMethods = true
		hasConfig = true
	}

	if generateSetterSamples := getBoolFromMap(codec, "generate-setter-samples"); generateSetterSamples {
		rust.GenerateSetterSamples = true
		hasConfig = true
	}

	if postProcessProtos := getBoolFromMap(codec, "post-process-protos"); postProcessProtos {
		rust.PostProcessProtos = true
		hasConfig = true
	}

	if detailedTracingAttributes := getBoolFromMap(codec, "detailed-tracing-attributes"); detailedTracingAttributes {
		rust.DetailedTracingAttributes = true
		hasConfig = true
	}

	if notForPublication := getBoolFromMap(codec, "not-for-publication"); notForPublication {
		rust.NotForPublication = true
		hasConfig = true
	}

	if disabledRustdoc, ok := codec["disabled-rustdoc-warnings"].(string); ok && disabledRustdoc != "" {
		rust.DisabledRustdocWarnings = strings.Split(disabledRustdoc, ",")
		hasConfig = true
	}

	if disabledClippy, ok := codec["disabled-clippy-warnings"].(string); ok && disabledClippy != "" {
		rust.DisabledClippyWarnings = strings.Split(disabledClippy, ",")
		hasConfig = true
	}

	if !hasConfig {
		return nil
	}
	return rust
}

func extractDartLibraryConfig(codec map[string]any) *config.DartLibrary {
	dart := &config.DartLibrary{}
	hasConfig := false

	if apiKeys := getStringFromMap(codec, "api-keys-environment-variables"); apiKeys != "" {
		dart.APIKeysEnvironmentVariables = apiKeys
		hasConfig = true
	}

	if devDeps := getStringFromMap(codec, "dev-dependencies"); devDeps != "" {
		dart.DevDependencies = strings.Split(devDeps, ",")
		hasConfig = true
	}

	if !hasConfig {
		return nil
	}
	return dart
}

func extractPackageDependencies(codec map[string]any) []config.PackageDependency {
	var deps []config.PackageDependency
	for key, value := range codec {
		if strings.HasPrefix(key, "package:") {
			name := strings.TrimPrefix(key, "package:")
			if strVal, ok := value.(string); ok {
				dep := parsePackageDependency(name, strVal)
				deps = append(deps, dep)
			}
		}
	}
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].Name < deps[j].Name
	})
	return deps
}

func parsePackageDependency(name, value string) config.PackageDependency {
	dep := config.PackageDependency{Name: name}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			switch key {
			case "package":
				dep.Package = val
			case "source":
				dep.Source = val
			case "force-used":
				dep.ForceUsed = val == "true"
			case "used-if":
				dep.UsedIf = val
			case "feature":
				dep.Feature = val
			case "ignore":
				if val == "true" {
					// When ignore=true, set package to empty string
					dep.Package = ""
				}
			}
		}
	}
	return dep
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringFromMap(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBoolFromMap(m map[string]any, key string) bool {
	if v, ok := m[key].(string); ok {
		return v == "true"
	}
	return false
}

func getStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key].(string); ok {
		if v == "" {
			return nil
		}
		// Handle comma-separated strings
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return nil
}
