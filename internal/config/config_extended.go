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

import "fmt"

// Extended types for full sidekick.toml compatibility.

// SourceExtended represents an external source repository with additional fields.
type SourceExtended struct {
	Source        `yaml:",inline"`
	ExtractedName string `yaml:"extracted_name,omitempty"`
	Subdir        string `yaml:"subdir,omitempty"`
}

// SourcesExtended contains references to all external source repositories.
type SourcesExtended struct {
	Googleapis  *SourceExtended `yaml:"googleapis,omitempty"`
	Showcase    *SourceExtended `yaml:"showcase,omitempty"`
	Discovery   *SourceExtended `yaml:"discovery,omitempty"`
	ProtobufSrc *SourceExtended `yaml:"protobuf_src,omitempty"`
	Conformance *SourceExtended `yaml:"conformance,omitempty"`
}

// RustDefaults contains Rust-specific default configuration.
type RustDefaults struct {
	DisabledRustdocWarnings []string             `yaml:"disabled_rustdoc_warnings,omitempty"`
	DisabledClippyWarnings  []string             `yaml:"disabled_clippy_warnings,omitempty"`
	PackageDependencies     []PackageDependency  `yaml:"package_dependencies,omitempty"`
	Release                 *RustReleaseDefaults `yaml:"release,omitempty"`
}

// RustReleaseDefaults contains Rust-specific release tool requirements.
type RustReleaseDefaults struct {
	Tools        map[string][]Tool `yaml:"tools,omitempty"`
	PreInstalled map[string]string `yaml:"pre_installed,omitempty"`
}

// Tool represents a required tool with version.
type Tool struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// PackageDependency represents a package dependency configuration.
type PackageDependency struct {
	Name      string `yaml:"name"`
	Package   string `yaml:"package"`
	Source    string `yaml:"source,omitempty"`
	ForceUsed bool   `yaml:"force_used,omitempty"`
	UsedIf    string `yaml:"used_if,omitempty"`
	Feature   string `yaml:"feature,omitempty"`
}

// DefaultsExtended contains extended default generation settings.
type DefaultsExtended struct {
	Defaults `yaml:",inline"`
	Rust     *RustDefaults `yaml:"rust,omitempty"`
}

// ReleaseExtended contains extended release configuration.
type ReleaseExtended struct {
	Release        `yaml:",inline"`
	Remote         string   `yaml:"remote,omitempty"`
	Branch         string   `yaml:"branch,omitempty"`
	IgnoredChanges []string `yaml:"ignored_changes,omitempty"`
}

// APIExtended represents an extended API configuration.
type APIExtended struct {
	Path                string `yaml:"path"`
	SpecificationFormat string `yaml:"specification_format,omitempty"`
}

// SingleAPI represents a single API that can be either a string path or an object with additional fields.
type SingleAPI struct {
	// For simple string paths
	StringValue string
	// For objects with specification_format or other fields
	ObjectValue *APIExtended
}

// UnmarshalYAML implements custom unmarshaling for SingleAPI to support both string and object forms.
func (a *SingleAPI) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as string first
	var str string
	if err := unmarshal(&str); err == nil {
		a.StringValue = str
		return nil
	}

	// Try to unmarshal as object
	var obj APIExtended
	if err := unmarshal(&obj); err == nil {
		a.ObjectValue = &obj
		return nil
	}

	return fmt.Errorf("api must be either a string or an object with path field")
}

// MarshalYAML implements custom marshaling for SingleAPI.
func (a SingleAPI) MarshalYAML() (interface{}, error) {
	if a.StringValue != "" {
		return a.StringValue, nil
	}
	if a.ObjectValue != nil {
		return a.ObjectValue, nil
	}
	return nil, nil
}

// GetPath returns the path regardless of whether it's a string or object.
func (a *SingleAPI) GetPath() string {
	if a.StringValue != "" {
		return a.StringValue
	}
	if a.ObjectValue != nil {
		return a.ObjectValue.Path
	}
	return ""
}

// GetSpecificationFormat returns the specification format if present.
func (a *SingleAPI) GetSpecificationFormat() string {
	if a.ObjectValue != nil {
		return a.ObjectValue.SpecificationFormat
	}
	return ""
}

// SourceFiltering contains source filtering options.
type SourceFiltering struct {
	Roots               []string `yaml:"roots,omitempty"`
	IncludedIDs         []string `yaml:"included_ids,omitempty"`
	SkippedIDs          []string `yaml:"skipped_ids,omitempty"`
	IncludeList         []string `yaml:"include_list,omitempty"`
	TitleOverride       string   `yaml:"title_override,omitempty"`
	DescriptionOverride string   `yaml:"description_override,omitempty"`
	ProjectRoot         string   `yaml:"project_root,omitempty"`
}

// DiscoveryPoller represents a poller configuration for Discovery APIs.
type DiscoveryPoller struct {
	Prefix   string `yaml:"prefix"`
	MethodID string `yaml:"method_id"`
}

// DiscoveryConfig contains Discovery API specific configuration.
type DiscoveryConfig struct {
	OperationID string            `yaml:"operation_id,omitempty"`
	Pollers     []DiscoveryPoller `yaml:"pollers,omitempty"`
}

// RustGenerate contains Rust-specific code generation options.
type RustGenerate struct {
	// Source filtering fields (flattened from SourceFiltering)
	Roots               []string `yaml:"roots,omitempty"`
	IncludedIDs         []string `yaml:"included_ids,omitempty"`
	SkippedIDs          []string `yaml:"skipped_ids,omitempty"`
	IncludeList         []string `yaml:"include_list,omitempty"`
	TitleOverride       string   `yaml:"title_override,omitempty"`
	DescriptionOverride string   `yaml:"description_override,omitempty"`
	ProjectRoot         string   `yaml:"project_root,omitempty"`
	// Code generation fields
	NameOverrides             map[string]string   `yaml:"name_overrides,omitempty"`
	ModulePath                string              `yaml:"module_path,omitempty"`
	RootName                  string              `yaml:"root_name,omitempty"`
	PerServiceFeatures        bool                `yaml:"per_service_features,omitempty"`
	DefaultFeatures           []string            `yaml:"default_features,omitempty"`
	ExtraModules              []string            `yaml:"extra_modules,omitempty"`
	PackageDependencies       []PackageDependency `yaml:"package_dependencies,omitempty"`
	HasVeneer                 bool                `yaml:"has_veneer,omitempty"`
	RoutingRequired           bool                `yaml:"routing_required,omitempty"`
	IncludeGrpcOnlyMethods    bool                `yaml:"include_grpc_only_methods,omitempty"`
	GenerateSetterSamples     bool                `yaml:"generate_setter_samples,omitempty"`
	PostProcessProtos         bool                `yaml:"post_process_protos,omitempty"`
	DetailedTracingAttributes bool                `yaml:"detailed_tracing_attributes,omitempty"`
	DisabledRustdocWarnings   []string            `yaml:"disabled_rustdoc_warnings,omitempty"`
	DisabledClippyWarnings    []string            `yaml:"disabled_clippy_warnings,omitempty"`
}

// LibraryGenerateExtended contains extended generation configuration for a library.
type LibraryGenerateExtended struct {
	// API for single API (can be string or object)
	API *SingleAPI `yaml:"api,omitempty"`
	// APIs for multiple APIs (array of objects)
	APIs      []APIExtended    `yaml:"apis,omitempty"`
	Template  string           `yaml:"template,omitempty"`
	Discovery *DiscoveryConfig `yaml:"discovery,omitempty"`
	Rust      *RustGenerate    `yaml:"rust,omitempty"`
	Keep      []string         `yaml:"keep,omitempty"`
	Remove    []string         `yaml:"remove,omitempty"`
}

// PublishConfig contains publication configuration.
type PublishConfig struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Registry string `yaml:"registry,omitempty"`
}

// LibraryExtended represents an extended library configuration.
type LibraryExtended struct {
	Name          string                   `yaml:"name"`
	Version       string                   `yaml:"version,omitempty"`
	CopyrightYear int                      `yaml:"copyright_year,omitempty"`
	Title         string                   `yaml:"title,omitempty"`
	Description   string                   `yaml:"description,omitempty"`
	Apis          []string                 `yaml:"apis,omitempty"`
	Path          string                   `yaml:"path,omitempty"`
	Generate      *LibraryGenerateExtended `yaml:"generate,omitempty"`
	Publish       *PublishConfig           `yaml:"publish,omitempty"`
}

// ConfigExtended represents the extended librarian.yaml configuration.
type ConfigExtended struct {
	Version   string            `yaml:"version"`
	Language  string            `yaml:"language,omitempty"`
	Container *Container        `yaml:"container,omitempty"`
	Sources   SourcesExtended   `yaml:"sources,omitempty"`
	Defaults  *DefaultsExtended `yaml:"defaults,omitempty"`
	Generate  *Generate         `yaml:"generate,omitempty"`
	Release   *ReleaseExtended  `yaml:"release,omitempty"`
	Libraries []LibraryExtended `yaml:"libraries,omitempty"`
}
