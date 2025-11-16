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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BazelConfig contains configuration extracted from BUILD.bazel files.
type BazelConfig struct {
	// GRPCServiceConfig is the path to the gRPC service config JSON file (relative to API directory).
	GRPCServiceConfig string

	// ServiceYAML is the path to the service YAML file (relative to API directory).
	ServiceYAML string

	// Transport specifies the transport(s) to generate (e.g., "grpc+rest", "grpc", "rest").
	Transport string

	// RestNumericEnums indicates whether to use numeric enums in REST.
	RestNumericEnums bool

	// OptArgs contains additional generator options.
	OptArgs []string

	// IsProtoOnly indicates this API has no GAPIC rule (proto-only library).
	IsProtoOnly bool
}

// ReadBuildBazel reads and parses a BUILD.bazel file for the given API path.
// Returns BazelConfig with extracted configuration, or an error if the file cannot be read.
func ReadBuildBazel(googleapisDir, apiPath string) (*BazelConfig, error) {
	buildFilePath := filepath.Join(googleapisDir, apiPath, "BUILD.bazel")

	content, err := os.ReadFile(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read BUILD.bazel: %w", err)
	}

	return parseBuildBazel(string(content))
}

// parseBuildBazel parses BUILD.bazel content and extracts py_gapic_library configuration.
func parseBuildBazel(content string) (*BazelConfig, error) {
	config := &BazelConfig{}

	// Look for py_gapic_library rule
	pyGapicPattern := regexp.MustCompile(`py_gapic_library\s*\([^)]+\)`)
	match := pyGapicPattern.FindString(content)

	if match == "" {
		// No py_gapic_library rule found - this is a proto-only library
		config.IsProtoOnly = true
		return config, nil
	}

	// Extract grpc_service_config
	if val := extractStringValue(match, "grpc_service_config"); val != "" {
		config.GRPCServiceConfig = val
	}

	// Extract service_yaml
	if val := extractStringValue(match, "service_yaml"); val != "" {
		config.ServiceYAML = val
	}

	// Extract transport
	if val := extractStringValue(match, "transport"); val != "" {
		config.Transport = val
	}

	// Extract rest_numeric_enums
	config.RestNumericEnums = extractBoolValue(match, "rest_numeric_enums")

	// Extract opt_args
	config.OptArgs = extractListValue(match, "opt_args")

	return config, nil
}

// extractStringValue extracts a string attribute value from BUILD.bazel content.
// Example: grpc_service_config = "file.json" -> returns "file.json".
func extractStringValue(content, key string) string {
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*"([^"]+)"`, regexp.QuoteMeta(key)))
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// extractBoolValue extracts a boolean attribute value from BUILD.bazel content.
// Example: rest_numeric_enums = True -> returns true.
func extractBoolValue(content, key string) bool {
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*(True|False)`, regexp.QuoteMeta(key)))
	matches := pattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1] == "True"
	}
	return false
}

// extractListValue extracts a list attribute value from BUILD.bazel content.
// Example: opt_args = ["arg1", "arg2"] -> returns ["arg1", "arg2"].
func extractListValue(content, key string) []string {
	// Match: key = [...] with multiline support
	// Use (?s) for dot to match newlines
	listPattern := regexp.MustCompile(fmt.Sprintf(`(?s)%s\s*=\s*\[(.*?)\]`, regexp.QuoteMeta(key)))
	listMatch := listPattern.FindStringSubmatch(content)
	if len(listMatch) < 2 {
		return nil
	}

	listContent := listMatch[1]

	// Extract quoted strings from the list
	stringPattern := regexp.MustCompile(`"([^"]+)"`)
	matches := stringPattern.FindAllStringSubmatch(listContent, -1)

	var result []string
	for _, match := range matches {
		if len(match) >= 2 {
			result = append(result, match[1])
		}
	}

	return result
}

// GetAbsolutePath returns the absolute path for a relative path from BUILD.bazel.
// Relative paths in BUILD.bazel are relative to the API directory.
func (b *BazelConfig) GetAbsolutePath(googleapisDir, apiPath, relativePath string) string {
	if relativePath == "" {
		return ""
	}
	return filepath.Join(googleapisDir, apiPath, relativePath)
}

// GetGRPCServiceConfigPath returns the absolute path to the gRPC service config file.
func (b *BazelConfig) GetGRPCServiceConfigPath(googleapisDir, apiPath string) string {
	return b.GetAbsolutePath(googleapisDir, apiPath, b.GRPCServiceConfig)
}

// GetServiceYAMLPath returns the absolute path to the service YAML file.
func (b *BazelConfig) GetServiceYAMLPath(googleapisDir, apiPath string) string {
	return b.GetAbsolutePath(googleapisDir, apiPath, b.ServiceYAML)
}

// MergeWithLibrary merges BUILD.bazel config with library config.
// Library config takes precedence over BUILD.bazel config.
func (b *BazelConfig) MergeWithLibrary(lib *Library, defaults *Default) {
	// Transport: library > BUILD.bazel > defaults
	if lib.Transport == "" && b.Transport != "" {
		lib.Transport = b.Transport
	}

	// RestNumericEnums: library > BUILD.bazel > defaults
	if lib.RestNumericEnums == nil && b.RestNumericEnums {
		lib.RestNumericEnums = &b.RestNumericEnums
	}

	// OptArgs: merge BUILD.bazel opt_args with library Python.OptArgs
	if len(b.OptArgs) > 0 {
		if lib.Python == nil {
			lib.Python = &PythonPackage{}
		}
		// Append BUILD.bazel opt_args to library opt_args (library takes precedence)
		seen := make(map[string]bool)
		for _, arg := range lib.Python.OptArgs {
			seen[arg] = true
		}
		for _, arg := range b.OptArgs {
			if !seen[arg] {
				lib.Python.OptArgs = append(lib.Python.OptArgs, arg)
			}
		}
	}
}

// String returns a string representation for debugging.
func (b *BazelConfig) String() string {
	if b.IsProtoOnly {
		return "proto-only library"
	}

	var parts []string
	if b.GRPCServiceConfig != "" {
		parts = append(parts, fmt.Sprintf("grpc_service_config=%s", b.GRPCServiceConfig))
	}
	if b.ServiceYAML != "" {
		parts = append(parts, fmt.Sprintf("service_yaml=%s", b.ServiceYAML))
	}
	if b.Transport != "" {
		parts = append(parts, fmt.Sprintf("transport=%s", b.Transport))
	}
	if b.RestNumericEnums {
		parts = append(parts, "rest_numeric_enums=True")
	}
	if len(b.OptArgs) > 0 {
		parts = append(parts, fmt.Sprintf("opt_args=%v", b.OptArgs))
	}

	return strings.Join(parts, ", ")
}
