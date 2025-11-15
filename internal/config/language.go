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

// RustDefault contains Rust-specific default configuration.
type RustDefault struct {
	// DisabledRustdocWarnings is a list of rustdoc warnings to disable.
	DisabledRustdocWarnings []string `yaml:"disabled_rustdoc_warnings,omitempty"`

	// PackageDependencies is a list of default package dependencies.
	PackageDependencies []*RustPackageDependency `yaml:"package_dependencies,omitempty"`
}

// RustCrate contains Rust-specific library configuration.
type RustCrate struct {
	// PerServiceFeatures enables per-service feature flags.
	PerServiceFeatures bool `yaml:"per_service_features,omitempty"`

	// Additional fields can be added as needed
	ModulePath                string                  `yaml:"module_path,omitempty"`
	TemplateOverride          string                  `yaml:"template_override,omitempty"`
	TitleOverride             string                  `yaml:"title_override,omitempty"`
	DescriptionOverride       string                  `yaml:"description_override,omitempty"`
	PackageNameOverride       string                  `yaml:"package_name_override,omitempty"`
	RootName                  string                  `yaml:"root_name,omitempty"`
	Roots                     []string                `yaml:"roots,omitempty"`
	DefaultFeatures           []string                `yaml:"default_features,omitempty"`
	ExtraModules              []string                `yaml:"extra_modules,omitempty"`
	IncludeList               []string                `yaml:"include_list,omitempty"`
	IncludedIds               []string                `yaml:"included_ids,omitempty"`
	SkippedIds                []string                `yaml:"skipped_ids,omitempty"`
	NameOverrides             string                  `yaml:"name_overrides,omitempty"`
	PackageDependencies       []RustPackageDependency `yaml:"package_dependencies,omitempty"`
	DisabledRustdocWarnings   []string                `yaml:"disabled_rustdoc_warnings,omitempty"`
	DisabledClippyWarnings    []string                `yaml:"disabled_clippy_warnings,omitempty"`
	HasVeneer                 bool                    `yaml:"has_veneer,omitempty"`
	RoutingRequired           bool                    `yaml:"routing_required,omitempty"`
	IncludeGrpcOnlyMethods    bool                    `yaml:"include_grpc_only_methods,omitempty"`
	GenerateSetterSamples     bool                    `yaml:"generate_setter_samples,omitempty"`
	PostProcessProtos         bool                    `yaml:"post_process_protos,omitempty"`
	DetailedTracingAttributes bool                    `yaml:"detailed_tracing_attributes,omitempty"`
	NotForPublication         bool                    `yaml:"not_for_publication,omitempty"`
}

// RustPackageDependency represents a package dependency configuration.
type RustPackageDependency struct {
	Name      string `yaml:"name"`
	Package   string `yaml:"package"`
	Source    string `yaml:"source,omitempty"`
	ForceUsed bool   `yaml:"force_used,omitempty"`
	UsedIf    string `yaml:"used_if,omitempty"`
	Feature   string `yaml:"feature,omitempty"`
}

// PythonPackage contains Python-specific library configuration.
type PythonPackage struct {
	// RestAsyncIOEnabled enables async I/O for REST transport.
	RestAsyncIOEnabled bool `yaml:"rest_async_io_enabled,omitempty"`

	// UnversionedPackageDisabled disables unversioned package generation.
	UnversionedPackageDisabled bool `yaml:"unversioned_package_disabled,omitempty"`

	// OptArgs contains additional options passed to the generator.
	// Example: ["warehouse-package-name=google-cloud-batch"]
	OptArgs []string `yaml:"opt_args,omitempty"`
}

// GoModule contains Go-specific library configuration.
type GoModule struct {
	// RenamedServices maps original service names to renamed versions.
	// Example: {"Publisher": "TopicAdmin", "Subscriber": "SubscriptionAdmin"}
	RenamedServices map[string]string `yaml:"renamed_services,omitempty"`

	// ImportPath is the Go package import path.
	// Example: "cloud.google.com/go/batch/apiv1;batchpb"
	ImportPath string `yaml:"import_path,omitempty"`

	// Metadata indicates whether to generate gapic_metadata.json.
	Metadata bool `yaml:"metadata,omitempty"`

	// Diregapic indicates whether this is a DIREGAPIC (Discovery REST GAPIC).
	Diregapic bool `yaml:"diregapic,omitempty"`

	// ServiceYAML is the client config file in the API version directory.
	ServiceYAML string `yaml:"service_yaml,omitempty"`

	// HasGoGRPC indicates whether a go_grpc_library rule is used.
	HasGoGRPC bool `yaml:"has_go_grpc,omitempty"`

	// HasLegacyGRPC indicates whether the go_proto_library rule uses the legacy gRPC plugin.
	HasLegacyGRPC bool `yaml:"has_legacy_grpc,omitempty"`
}

// DartPackage contains Dart-specific library configuration.
type DartPackage struct {
	// APIKeysEnvironmentVariables is a comma-separated list of environment variable names for API keys.
	APIKeysEnvironmentVariables string `yaml:"api_keys_environment_variables,omitempty"`

	// DevDependencies is a list of development dependencies.
	DevDependencies []string `yaml:"dev_dependencies,omitempty"`
}
