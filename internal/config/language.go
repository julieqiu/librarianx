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

	// ModulePath is the module path for the crate.
	ModulePath string `yaml:"module_path,omitempty"`

	// TemplateOverride overrides the default template.
	TemplateOverride string `yaml:"template_override,omitempty"`

	// TitleOverride overrides the crate title.
	TitleOverride string `yaml:"title_override,omitempty"`

	// DescriptionOverride overrides the crate description.
	DescriptionOverride string `yaml:"description_override,omitempty"`

	// PackageNameOverride overrides the package name.
	PackageNameOverride string `yaml:"package_name_override,omitempty"`

	// RootName is the root name for the crate.
	RootName string `yaml:"root_name,omitempty"`

	// Roots is a list of root names.
	Roots []string `yaml:"roots,omitempty"`

	// DefaultFeatures is a list of default features to enable.
	DefaultFeatures []string `yaml:"default_features,omitempty"`

	// ExtraModules is a list of extra modules to include.
	ExtraModules []string `yaml:"extra_modules,omitempty"`

	// IncludeList is a list of items to include.
	IncludeList []string `yaml:"include_list,omitempty"`

	// IncludedIds is a list of IDs to include.
	IncludedIds []string `yaml:"included_ids,omitempty"`

	// SkippedIds is a list of IDs to skip.
	SkippedIds []string `yaml:"skipped_ids,omitempty"`

	// NameOverrides contains name overrides.
	NameOverrides string `yaml:"name_overrides,omitempty"`

	// PackageDependencies is a list of package dependencies.
	PackageDependencies []RustPackageDependency `yaml:"package_dependencies,omitempty"`

	// DisabledRustdocWarnings is a list of rustdoc warnings to disable.
	DisabledRustdocWarnings []string `yaml:"disabled_rustdoc_warnings,omitempty"`

	// DisabledClippyWarnings is a list of clippy warnings to disable.
	DisabledClippyWarnings []string `yaml:"disabled_clippy_warnings,omitempty"`

	// HasVeneer indicates whether the crate has a veneer.
	HasVeneer bool `yaml:"has_veneer,omitempty"`

	// RoutingRequired indicates whether routing is required.
	RoutingRequired bool `yaml:"routing_required,omitempty"`

	// IncludeGrpcOnlyMethods indicates whether to include gRPC-only methods.
	IncludeGrpcOnlyMethods bool `yaml:"include_grpc_only_methods,omitempty"`

	// GenerateSetterSamples indicates whether to generate setter samples.
	GenerateSetterSamples bool `yaml:"generate_setter_samples,omitempty"`

	// PostProcessProtos indicates whether to post-process protos.
	PostProcessProtos bool `yaml:"post_process_protos,omitempty"`

	// DetailedTracingAttributes indicates whether to include detailed tracing attributes.
	DetailedTracingAttributes bool `yaml:"detailed_tracing_attributes,omitempty"`

	// DocumentationOverrides contains overrides for element documentation.
	DocumentationOverrides []RustDocumentationOverride `yaml:"documentation_overrides,omitempty"`

	// PaginationOverrides contains overrides for pagination configuration.
	PaginationOverrides []RustPaginationOverride `yaml:"pagination_overrides,omitempty"`
}

// RustPackageDependency represents a package dependency configuration.
type RustPackageDependency struct {
	// Name is the dependency name.
	Name string `yaml:"name"`

	// Package is the package name.
	Package string `yaml:"package"`

	// Source is the dependency source.
	Source string `yaml:"source,omitempty"`

	// ForceUsed forces the dependency to be used even if not referenced.
	ForceUsed bool `yaml:"force_used,omitempty"`

	// UsedIf specifies a condition for when the dependency is used.
	UsedIf string `yaml:"used_if,omitempty"`

	// Feature is the feature name for the dependency.
	Feature string `yaml:"feature,omitempty"`
}

// RustDocumentationOverride represents a documentation override for a specific element.
type RustDocumentationOverride struct {
	// ID is the fully qualified element ID (e.g., .google.cloud.dialogflow.v2.Message.field).
	ID string `yaml:"id"`

	// Match is the text to match in the documentation.
	Match string `yaml:"match"`

	// Replace is the replacement text.
	Replace string `yaml:"replace"`
}

// RustPaginationOverride represents a pagination override for a specific method.
type RustPaginationOverride struct {
	// ID is the fully qualified method ID (e.g., .google.cloud.sql.v1.Service.Method).
	ID string `yaml:"id"`

	// ItemField is the name of the field used for items.
	ItemField string `yaml:"item_field"`
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

	// IsProtoOnly indicates this library only contains proto files (no GAPIC generation).
	IsProtoOnly bool `yaml:"is_proto_only,omitempty"`

	// APIDescription overrides the API description in .repo-metadata.json.
	// If not set, the description is derived from the service YAML's documentation.summary field.
	APIDescription string `yaml:"api_description,omitempty"`
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

	// ModulePathVersion is the module path version suffix (e.g., "v2").
	ModulePathVersion string `yaml:"module_path_version,omitempty"`

	// DeleteGenerationOutputPaths specifies paths to delete after generation.
	DeleteGenerationOutputPaths []string `yaml:"delete_generation_output_paths,omitempty"`

	// APIs contains per-API Go-specific configuration.
	APIs []GoAPI `yaml:"apis,omitempty"`
}

// GoAPI contains Go-specific configuration for a single API.
type GoAPI struct {
	// Path is the API path (e.g., "google/cloud/secretmanager/v1").
	Path string `yaml:"path,omitempty"`

	// ClientDirectory is the client output directory relative to the module root.
	ClientDirectory string `yaml:"client_directory,omitempty"`

	// DisableGapic disables GAPIC generation for this API.
	DisableGapic bool `yaml:"disable_gapic,omitempty"`

	// NestedProtos specifies additional proto files to include.
	NestedProtos []string `yaml:"nested_protos,omitempty"`

	// ProtoPackage specifies the proto package name override.
	ProtoPackage string `yaml:"proto_package,omitempty"`
}

// DartPackage contains Dart-specific library configuration.
type DartPackage struct {
	// APIKeysEnvironmentVariables is a comma-separated list of environment variable names for API keys.
	APIKeysEnvironmentVariables string `yaml:"api_keys_environment_variables,omitempty"`

	// DevDependencies is a list of development dependencies.
	DevDependencies []string `yaml:"dev_dependencies,omitempty"`
}
