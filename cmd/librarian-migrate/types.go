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

// LegacyState represents the old .librarian/state.yaml file.
type LegacyState struct {
	// Image is the generator container image reference.
	Image string `yaml:"image"`

	// Libraries is the list of libraries in the repository.
	Libraries []LegacyStateLibrary `yaml:"libraries"`
}

// LegacyStateLibrary represents a library in state.yaml.
type LegacyStateLibrary struct {
	// ID is the unique identifier for the library.
	ID string `yaml:"id"`

	// Version is the last released version.
	Version string `yaml:"version,omitempty"`

	// LastGeneratedCommit is the googleapis commit hash at last generation.
	LastGeneratedCommit string `yaml:"last_generated_commit,omitempty"`

	// APIs is the list of APIs that are part of this library.
	APIs []LegacyAPI `yaml:"apis"`

	// SourceRoots is the list of directories where code is generated.
	SourceRoots []string `yaml:"source_roots"`

	// PreserveRegex is the list of patterns for files to preserve.
	PreserveRegex []string `yaml:"preserve_regex,omitempty"`

	// RemoveRegex is the list of patterns for files to remove.
	RemoveRegex []string `yaml:"remove_regex,omitempty"`

	// ReleaseExcludePaths is the list of paths to exclude from releases.
	ReleaseExcludePaths []string `yaml:"release_exclude_paths,omitempty"`

	// TagFormat is the format string for release tags.
	TagFormat string `yaml:"tag_format,omitempty"`
}

// LegacyAPI represents an API in state.yaml.
type LegacyAPI struct {
	// Path is the API path relative to googleapis root.
	Path string `yaml:"path"`

	// ServiceConfig is the service config file name.
	ServiceConfig string `yaml:"service_config,omitempty"`
}

// LegacyConfig represents the old .librarian/config.yaml file.
type LegacyConfig struct {
	// GlobalFilesAllowlist is the list of global files the container can access.
	GlobalFilesAllowlist []GlobalFile `yaml:"global_files_allowlist,omitempty"`

	// Libraries contains library-specific overrides.
	Libraries []LegacyConfigLibrary `yaml:"libraries,omitempty"`
}

// GlobalFile represents a global file configuration.
type GlobalFile struct {
	// Path is the file path from repository root.
	Path string `yaml:"path"`

	// Permissions is the access mode (read-only, write-only, read-write).
	Permissions string `yaml:"permissions"`
}

// LegacyConfigLibrary represents a library override in config.yaml.
type LegacyConfigLibrary struct {
	// ID is the library identifier.
	ID string `yaml:"id"`

	// NextVersion is the next version to release.
	NextVersion string `yaml:"next_version,omitempty"`

	// GenerateBlocked prevents generation.
	GenerateBlocked bool `yaml:"generate_blocked,omitempty"`

	// ReleaseBlocked prevents release.
	ReleaseBlocked bool `yaml:"release_blocked,omitempty"`
}

// BuildBazelData represents parsed data from BUILD.bazel files.
type BuildBazelData struct {
	// Libraries maps library ID to BUILD.bazel metadata.
	Libraries map[string]*BuildLibrary
}

// BuildLibrary represents BUILD.bazel metadata for a library.
type BuildLibrary struct {
	// ID is the library identifier.
	ID string

	// Transport is the transport protocol (grpc, rest, grpc+rest).
	Transport string

	// OptArgs are additional generator options (Python-specific).
	OptArgs []string

	// ServiceYAML is the service config file.
	ServiceYAML string

	// GRPCServiceConfig is the gRPC service config JSON file.
	GRPCServiceConfig string

	// RestNumericEnums indicates whether to use numeric enums in REST.
	RestNumericEnums bool

	// IsProtoOnly indicates this API has no GAPIC rule (proto-only library).
	IsProtoOnly bool

	// Go-specific fields
	// ImportPath is the Go package import path from go_gapic_library.
	ImportPath string

	// Metadata indicates whether to generate gapic_metadata.json (Go).
	Metadata bool

	// ReleaseLevel is the release level (e.g., "ga", "beta", "alpha") (Go).
	ReleaseLevel string
}

// LegacyGeneratorInputData represents parsed data from .librarian/generator-input/.
type LegacyGeneratorInputData struct {
	// Files maps file names to their content.
	Files map[string][]byte
}
