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

package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config/internal/bazel"
)

// EnrichWithBazelMetadata enriches a configuration with metadata extracted from BUILD.bazel files.
// For each library with an API field, it parses the corresponding BUILD.bazel file in googleapis
// and stores the metadata in the library's BazelMetadata field.
//
// This function silently skips libraries that don't have BUILD.bazel files or where parsing fails,
// as the metadata is supplementary and not required.
func EnrichWithBazelMetadata(cfg *Config, googleapisRoot string) error {
	if googleapisRoot == "" {
		slog.Debug("skipping bazel metadata enrichment: no googleapis root provided")
		return nil
	}

	language := cfg.Language
	if language == "" {
		slog.Debug("skipping bazel metadata enrichment: no language specified")
		return nil
	}

	for i := range cfg.Libraries {
		entry := &cfg.Libraries[i]

		// Skip wildcard entries
		if entry.Name == "*" {
			continue
		}

		// Skip if no config (shouldn't happen for non-wildcard entries)
		if entry.Config == nil {
			continue
		}

		// Get API paths to process
		var apiPaths []string
		if entry.Config.API != nil {
			if apiStr, ok := entry.Config.API.(string); ok {
				apiPaths = append(apiPaths, apiStr)
			}
		}
		apiPaths = append(apiPaths, entry.Config.APIs...)

		// Skip if no API paths
		if len(apiPaths) == 0 {
			continue
		}

		// Process the first API path (for libraries with multiple APIs, they typically
		// share the same metadata, so we just use the first one)
		apiPath := apiPaths[0]

		// Parse BUILD.bazel file
		apiConfig, err := bazel.ParseAPI(googleapisRoot, apiPath, language)
		if err != nil {
			// Silently skip - BUILD.bazel files are supplementary
			slog.Debug("skipping bazel metadata", "library", entry.Name, "api", apiPath, "error", err)
			continue
		}

		// Skip if no GAPIC (proto-only libraries don't have metadata)
		if !apiConfig.HasGAPIC {
			slog.Debug("skipping bazel metadata: no GAPIC found", "library", entry.Name, "api", apiPath)
			continue
		}

		// Convert bazel.APIConfig to config.BazelMetadata
		metadata := &BazelMetadata{}

		// Find grpc_service_config file in the API directory (only for Go)
		if language == "go" {
			apiDir := filepath.Join(googleapisRoot, apiPath)
			if grpcConfig := findGRPCServiceConfig(apiDir); grpcConfig != "" {
				metadata.GRPCServiceConfig = grpcConfig
			}
		}

		// Only include transport if it's not the default "grpc+rest"
		if apiConfig.Transport != "" && apiConfig.Transport != "grpc+rest" {
			metadata.Transport = apiConfig.Transport
		}

		// Only include rest_numeric_enums if not the default (true)
		if !apiConfig.RestNumericEnums {
			metadata.RestNumericEnums = false
		}

		// Only include release_level if specified
		if apiConfig.ReleaseLevel != "" {
			metadata.ReleaseLevel = apiConfig.ReleaseLevel
		}

		// Add language-specific metadata
		switch language {
		case "go":
			if apiConfig.Go != nil && apiConfig.Go.ImportPath != "" {
				metadata.Go = &BazelGoMetadata{
					ImportPath:    apiConfig.Go.ImportPath,
					Metadata:      apiConfig.Go.Metadata,
					Diregapic:     apiConfig.Go.Diregapic,
					ServiceYAML:   apiConfig.ServiceYAML,
					HasGoGRPC:     apiConfig.Go.HasGoGRPC,
					HasLegacyGRPC: apiConfig.Go.HasLegacyGRPC,
				}
			}
		case "python":
			if apiConfig.Python != nil && len(apiConfig.Python.OptArgs) > 0 {
				metadata.Python = &BazelPythonMetadata{
					OptArgs: apiConfig.Python.OptArgs,
				}
			}
		}

		// Store metadata only if it has content
		if !isEmptyBazelMetadata(metadata) {
			entry.Config.BazelMetadata = metadata
			slog.Debug("enriched library with bazel metadata", "library", entry.Name, "api", apiPath)
		}
	}

	return nil
}

// isEmptyBazelMetadata checks if BazelMetadata has any non-zero fields.
func isEmptyBazelMetadata(m *BazelMetadata) bool {
	if m == nil {
		return true
	}
	return m.Transport == "" &&
		!m.RestNumericEnums &&
		m.ReleaseLevel == "" &&
		m.GRPCServiceConfig == "" &&
		m.Go == nil &&
		m.Python == nil
}

// findGRPCServiceConfig looks for a *_grpc_service_config.json file in the given directory.
// Returns the filename (not full path) if found, empty string otherwise.
func findGRPCServiceConfig(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_grpc_service_config.json") {
			return name
		}
	}

	return ""
}

// GetAPIPath returns the first API path for a library, or empty string if none.
func (l *LibraryConfig) GetAPIPath() string {
	if l.API != nil {
		if apiStr, ok := l.API.(string); ok {
			return apiStr
		}
	}
	if len(l.APIs) > 0 {
		return l.APIs[0]
	}
	return ""
}

// GetAllAPIPaths returns all API paths for a library.
func (l *LibraryConfig) GetAllAPIPaths() []string {
	var paths []string
	if l.API != nil {
		if apiStr, ok := l.API.(string); ok {
			paths = append(paths, apiStr)
		}
	}
	paths = append(paths, l.APIs...)
	return paths
}

// IsHandwritten returns true if this library has no API configuration
// (i.e., it's a handwritten library, not generated from googleapis).
func (l *LibraryConfig) IsHandwritten() bool {
	if l.API != nil {
		if apiStr, ok := l.API.(string); ok && apiStr != "" {
			return false
		}
	}
	return len(l.APIs) == 0
}

// HasCustomConfig returns true if this library has any custom configuration
// beyond just specifying an API path.
func (l *LibraryConfig) HasCustomConfig() bool {
	// Check for non-API fields
	if l.Path != "" {
		return true
	}
	if len(l.Keep) > 0 {
		return true
	}
	if l.Release != nil {
		return true
	}
	if l.Disabled {
		return true
	}
	if l.Transport != "" {
		return true
	}
	if l.RestNumericEnums != nil {
		return true
	}
	if l.ReleaseLevel != "" {
		return true
	}
	if l.Rust != nil {
		return true
	}
	if l.Dart != nil {
		return true
	}
	if l.Python != nil {
		return true
	}
	if l.Go != nil {
		return true
	}
	if l.Java != nil {
		return true
	}
	if l.Node != nil {
		return true
	}
	if l.Dotnet != nil {
		return true
	}
	if l.LaunchStage != "" {
		return true
	}
	if len(l.Destinations) > 0 {
		return true
	}
	// BazelMetadata doesn't count as "custom config" since it's auto-generated
	return false
}

// NormalizeAPIPath normalizes an API path by removing leading/trailing slashes.
func NormalizeAPIPath(path string) string {
	return strings.Trim(path, "/")
}

// ParseBazelForGo parses a BUILD.bazel file in the given directory and returns
// configuration for Go generation. This is a wrapper around the internal bazel
// parser to allow access from other packages.
func ParseBazelForGo(apiServiceDir string) (BazelConfig, error) {
	cfg, err := bazel.Parse(apiServiceDir)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// BazelConfig provides configuration extracted from BUILD.bazel files.
// This interface allows other packages to use bazel configuration without
// directly importing the internal bazel package.
type BazelConfig interface {
	GAPICImportPath() string
	ServiceYAML() string
	GRPCServiceConfig() string
	Transport() string
	ReleaseLevel() string
	HasMetadata() bool
	HasDiregapic() bool
	HasRESTNumericEnums() bool
	HasGoGRPC() bool
	HasGAPIC() bool
	HasLegacyGRPC() bool
	DisableGAPIC()
}

// EnrichWithServiceConfigSettings enriches a configuration with language-specific settings
// extracted from service config YAML files in googleapis.
// For each library with an API field, it extracts the appropriate language settings from
// the service config and merges them into the library's configuration.
//
// This function silently skips libraries that don't have service configs or where extraction fails,
// as the settings are supplementary and not required for all libraries.
func EnrichWithServiceConfigSettings(cfg *Config, googleapisRoot string) error {
	if googleapisRoot == "" {
		slog.Debug("skipping service config enrichment: no googleapis root provided")
		return nil
	}

	language := cfg.Language
	if language == "" {
		slog.Debug("skipping service config enrichment: no language specified")
		return nil
	}

	for i := range cfg.Libraries {
		entry := &cfg.Libraries[i]

		// Skip wildcard entries
		if entry.Name == "*" {
			continue
		}

		// Skip if no config (shouldn't happen for non-wildcard entries)
		if entry.Config == nil {
			continue
		}

		// Get the first API path to process
		apiPath := entry.Config.GetAPIPath()
		if apiPath == "" {
			continue
		}

		// Extract language-specific settings from service config
		settings, err := ExtractServiceConfigSettings(googleapisRoot, apiPath, language)
		if err != nil {
			// Log but don't fail - service config extraction is best-effort
			slog.Debug("failed to extract service config settings", "library", entry.Name, "api", apiPath, "error", err)
			continue
		}

		// Skip if no settings were extracted
		if settings == nil {
			slog.Debug("no service config settings found", "library", entry.Name, "api", apiPath)
			continue
		}

		// Merge extracted settings into library config
		// Only set fields if they're not already set in the library config
		if settings.Java != nil && entry.Config.Java == nil {
			entry.Config.Java = settings.Java
			slog.Debug("enriched library with Java settings from service config", "library", entry.Name, "api", apiPath)
		}

		if settings.Python != nil && entry.Config.Python == nil {
			entry.Config.Python = settings.Python
			slog.Debug("enriched library with Python settings from service config", "library", entry.Name, "api", apiPath)
		}

		if settings.Go != nil && entry.Config.Go == nil {
			entry.Config.Go = settings.Go
			slog.Debug("enriched library with Go settings from service config", "library", entry.Name, "api", apiPath)
		}

		if settings.Node != nil && entry.Config.Node == nil {
			entry.Config.Node = settings.Node
			slog.Debug("enriched library with Node settings from service config", "library", entry.Name, "api", apiPath)
		}

		if settings.Dotnet != nil && entry.Config.Dotnet == nil {
			entry.Config.Dotnet = settings.Dotnet
			slog.Debug("enriched library with Dotnet settings from service config", "library", entry.Name, "api", apiPath)
		}

		// Don't override launch_stage and destinations if already set
		// These are typically derived or explicitly configured
	}

	return nil
}
