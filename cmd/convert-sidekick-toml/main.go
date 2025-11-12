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
	General GeneralConfig        `toml:"general"`
	Source  SourceConfig         `toml:"source"`
	Codec   map[string]any       `toml:"codec"`
	Release ReleaseConfig        `toml:"release"`
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
	sources := config.SourcesExtended{}
	if rootConfig.Source.GoogleapisRoot != "" {
		sources.Googleapis = &config.SourceExtended{
			Source: config.Source{
				URL:    rootConfig.Source.GoogleapisRoot,
				SHA256: rootConfig.Source.GoogleapisSHA256,
			},
		}
	}
	if rootConfig.Source.ShowcaseRoot != "" {
		sources.Showcase = &config.SourceExtended{
			Source: config.Source{
				URL:    rootConfig.Source.ShowcaseRoot,
				SHA256: rootConfig.Source.ShowcaseSHA256,
			},
			ExtractedName: rootConfig.Source.ShowcaseExtractedName,
		}
	}
	if rootConfig.Source.DiscoveryRoot != "" {
		sources.Discovery = &config.SourceExtended{
			Source: config.Source{
				URL:    rootConfig.Source.DiscoveryRoot,
				SHA256: rootConfig.Source.DiscoverySHA256,
			},
			ExtractedName: rootConfig.Source.DiscoveryExtractedName,
		}
	}
	if rootConfig.Source.ProtobufSrcRoot != "" {
		sources.ProtobufSrc = &config.SourceExtended{
			Source: config.Source{
				URL:    rootConfig.Source.ProtobufSrcRoot,
				SHA256: rootConfig.Source.ProtobufSrcSHA256,
			},
			ExtractedName: rootConfig.Source.ProtobufSrcExtractedName,
			Subdir:        rootConfig.Source.ProtobufSrcSubdir,
		}
	}

	// Extract default package dependencies from codec
	defaultPackageDeps := extractPackageDependencies(rootConfig.Codec)

	// Build Rust defaults
	rustDefaults := &config.RustDefaults{
		PackageDependencies: defaultPackageDeps,
	}
	if disabledRustdoc, ok := rootConfig.Codec["disabled-rustdoc-warnings"].(string); ok {
		rustDefaults.DisabledRustdocWarnings = strings.Split(disabledRustdoc, ",")
	}

	// Build release config
	rustRelease := &config.RustReleaseDefaults{
		Tools:        convertTools(rootConfig.Release.Tools),
		PreInstalled: rootConfig.Release.PreInstalled,
	}
	if len(rustRelease.Tools) > 0 || len(rustRelease.PreInstalled) > 0 {
		rustDefaults.Release = rustRelease
	}

	// Create librarian config
	libConfig := &config.ConfigExtended{
		Version:  "v1",
		Language: rootConfig.General.Language,
		Container: &config.Container{
			Image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/rust-librarian-generator",
			Tag:   "latest",
		},
		Sources: sources,
		Defaults: &config.DefaultsExtended{
			Defaults: config.Defaults{
				ReleaseLevel: getString(rootConfig.Codec, "release-level"),
			},
			Rust: rustDefaults,
		},
		Generate: &config.Generate{
			Output: "src/generated/{api.path}",
		},
		Release: &config.ReleaseExtended{
			Release: config.Release{
				TagFormat: "{name}/v{version}",
			},
			Remote:         rootConfig.Release.Remote,
			Branch:         rootConfig.Release.Branch,
			IgnoredChanges: rootConfig.Release.IgnoredChanges,
		},
		Libraries: []config.LibraryExtended{},
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
		library, err := processLibraryConfig(configPath, rootDir)
		if err != nil {
			log.Printf("Warning: failed to process %s: %v", configPath, err)
			continue
		}
		if library != nil {
			libConfig.Libraries = append(libConfig.Libraries, *library)
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

func processLibraryConfig(configPath, rootDir string) (*config.LibraryExtended, error) {
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

	// Derive library name from path
	relPath, err := filepath.Rel(rootDir, filepath.Dir(configPath))
	if err != nil {
		return nil, err
	}

	// Convert path to library name (e.g., src/generated/cloud/kms/v1 -> cloud-kms-v1)
	name := strings.ReplaceAll(strings.TrimPrefix(relPath, "src/generated/"), "/", "-")
	name = strings.ReplaceAll(name, "src-", "")

	// Build API configuration - use singular 'api' field for single APIs
	api := &config.SingleAPI{}
	if libConfig.General.SpecificationFormat == "" {
		// Simple case: just a path string
		api.StringValue = libConfig.General.SpecificationSource
	} else {
		// Has specification_format: use object form
		api.ObjectValue = &config.APIExtended{
			Path:                libConfig.General.SpecificationSource,
			SpecificationFormat: libConfig.General.SpecificationFormat,
		}
	}

	// Build generate configuration
	generate := &config.LibraryGenerateExtended{
		API: api,
	}

	// Add discovery configuration if present
	if len(libConfig.Discovery) > 0 {
		generate.Discovery = &config.DiscoveryConfig{
			OperationID: getStringFromMap(libConfig.Discovery, "operation-id"),
			Pollers:     extractPollers(libConfig.Discovery),
		}
	}

	// Add Rust-specific configuration (including source filtering)
	generate.Rust = extractRustGenerate(libConfig.Codec, libConfig.Source)

	// Use package-name-override if present, otherwise use derived name
	packageName := getStringFromMap(libConfig.Codec, "package-name-override")
	if packageName == "" {
		packageName = name
	}

	// Only set path if it's in src/ (like src/storage/, src/firestore/, etc.)
	// Default is src/generated/{api.path}, so we only need explicit path for non-default locations
	var explicitPath string
	if strings.HasPrefix(relPath, "src/") && !strings.HasPrefix(relPath, "src/generated/") {
		// This is in src/ but not src/generated/, so it needs explicit path
		explicitPath = relPath
	}
	// Otherwise omit path - will use default from generate.output template

	library := &config.LibraryExtended{
		Name:     packageName,
		Version:  getStringFromMap(libConfig.Codec, "version"),
		Path:     explicitPath,
		Generate: generate,
	}

	return library, nil
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
			}
		}
	}
	return dep
}

func extractRustGenerate(codec map[string]any, source map[string]any) *config.RustGenerate {
	rust := &config.RustGenerate{}

	// Add source filtering fields directly (flattened, not nested)
	if len(source) > 0 {
		rust.Roots = getStringSlice(source, "roots")
		rust.IncludedIDs = getStringSlice(source, "included-ids")
		rust.SkippedIDs = getStringSlice(source, "skipped-ids")
		rust.IncludeList = getStringSlice(source, "include-list")
		rust.TitleOverride = getStringFromMap(source, "title-override")
		rust.DescriptionOverride = getStringFromMap(source, "description-override")
		rust.ProjectRoot = getStringFromMap(source, "project-root")
	}

	rust.ModulePath = getStringFromMap(codec, "module-path")
	rust.RootName = getStringFromMap(codec, "root-name")
	rust.PerServiceFeatures = getBoolFromMap(codec, "per-service-features")
	rust.DefaultFeatures = getStringSlice(codec, "default-features")
	rust.ExtraModules = getStringSlice(codec, "extra-modules")
	rust.HasVeneer = getBoolFromMap(codec, "has-veneer")
	rust.RoutingRequired = getBoolFromMap(codec, "routing-required")
	rust.IncludeGrpcOnlyMethods = getBoolFromMap(codec, "include-grpc-only-methods")
	rust.GenerateSetterSamples = getBoolFromMap(codec, "generate-setter-samples")
	rust.PostProcessProtos = getBoolFromMap(codec, "post-process-protos")
	rust.DetailedTracingAttributes = getBoolFromMap(codec, "detailed-tracing-attributes")

	if disabledRustdoc, ok := codec["disabled-rustdoc-warnings"].(string); ok {
		rust.DisabledRustdocWarnings = strings.Split(disabledRustdoc, ",")
	}
	if disabledClippy, ok := codec["disabled-clippy-warnings"].(string); ok {
		rust.DisabledClippyWarnings = strings.Split(disabledClippy, ",")
	}

	// Extract name overrides
	if nameOverrides, ok := codec["name-overrides"].(string); ok {
		rust.NameOverrides = parseNameOverrides(nameOverrides)
	}

	// Extract package dependencies
	rust.PackageDependencies = extractPackageDependencies(codec)

	// Return nil if the struct is empty (all fields are zero values)
	if len(rust.Roots) == 0 &&
		len(rust.IncludedIDs) == 0 &&
		len(rust.SkippedIDs) == 0 &&
		len(rust.IncludeList) == 0 &&
		rust.TitleOverride == "" &&
		rust.DescriptionOverride == "" &&
		rust.ProjectRoot == "" &&
		rust.ModulePath == "" &&
		rust.RootName == "" &&
		!rust.PerServiceFeatures &&
		!rust.HasVeneer &&
		!rust.RoutingRequired &&
		!rust.IncludeGrpcOnlyMethods &&
		!rust.GenerateSetterSamples &&
		!rust.PostProcessProtos &&
		!rust.DetailedTracingAttributes &&
		len(rust.DefaultFeatures) == 0 &&
		len(rust.ExtraModules) == 0 &&
		len(rust.DisabledRustdocWarnings) == 0 &&
		len(rust.DisabledClippyWarnings) == 0 &&
		len(rust.NameOverrides) == 0 &&
		len(rust.PackageDependencies) == 0 {
		return nil
	}

	return rust
}

func parseNameOverrides(value string) map[string]string {
	overrides := make(map[string]string)
	parts := strings.Split(value, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			overrides[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return overrides
}

func extractPollers(discovery map[string]any) []config.DiscoveryPoller {
	var pollers []config.DiscoveryPoller
	if pollersRaw, ok := discovery["pollers"].([]any); ok {
		for _, p := range pollersRaw {
			if pollerMap, ok := p.(map[string]any); ok {
				poller := config.DiscoveryPoller{
					Prefix:   getStringFromMap(pollerMap, "prefix"),
					MethodID: getStringFromMap(pollerMap, "method-id"),
				}
				pollers = append(pollers, poller)
			}
		}
	}
	return pollers
}

func convertTools(tools map[string][]ToolConfig) map[string][]config.Tool {
	result := make(map[string][]config.Tool)
	for key, toolList := range tools {
		converted := make([]config.Tool, len(toolList))
		for i, t := range toolList {
			converted[i] = config.Tool{
				Name:    t.Name,
				Version: t.Version,
			}
		}
		result[key] = converted
	}
	return result
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

func getIntFromMap(m map[string]any, key string) int {
	if v, ok := m[key].(string); ok {
		var result int
		fmt.Sscanf(v, "%d", &result)
		return result
	}
	return 0
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
