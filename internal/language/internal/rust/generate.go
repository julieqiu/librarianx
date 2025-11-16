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

package rust

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	sidekick "github.com/googleapis/librarian/internal/sidekick/sidekick"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
	if err := sidekick.VerifyRustTools(); err != nil {
		return err
	}

	if defaults.Rust != nil {
		if library.Rust == nil {
			library.Rust = &config.RustCrate{}
		}
		if len(library.Rust.DisabledRustdocWarnings) == 0 {
			library.Rust.DisabledRustdocWarnings = defaults.Rust.DisabledRustdocWarnings
		}
		// Merge default package dependencies with library-specific ones
		library.Rust.PackageDependencies = mergePackageDependencies(defaults.Rust.PackageDependencies, library.Rust.PackageDependencies)
	}

	outdir := filepath.Join(defaultOutput, strings.TrimPrefix(library.API, "google/"))
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		if err := sidekick.PrepareCargoWorkspace(outdir); err != nil {
			return err
		}
	}

	sidekickConfig, err := toSidekickConfig(library, googleapisDir, serviceConfigPath)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickrust.Generate(model, outdir, sidekickConfig); err != nil {
		return err
	}
	return sidekick.PostGenerate(outdir)
}

func toSidekickConfig(library *config.Library, googleapisDir, serviceConfig string) (*sidekickconfig.Config, error) {
	sidekickCfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			Language:            "rust",
			SpecificationFormat: "protobuf",
			ServiceConfig:       serviceConfig,
			SpecificationSource: library.API,
		},
		Source: map[string]string{
			"googleapis-root": googleapisDir,
		},
		Codec: buildCodec(library),
	}

	// Add documentation overrides
	// Start with global overrides from googleapis, filtered to only include
	// overrides that are relevant to this API
	globalOverrides, err := config.ReadDocumentationOverrides()
	if err != nil {
		return nil, err
	}

	var allOverrides []config.RustDocumentationOverride
	// Filter global overrides to only include ones for this API
	// Convert API path (e.g., "google/cloud/developerconnect/v1") to proto package prefix (e.g., ".google.cloud.developerconnect.v1.")
	apiPrefix := "." + strings.ReplaceAll(library.API, "/", ".") + "."
	for _, override := range globalOverrides {
		if strings.HasPrefix(override.ID, apiPrefix) {
			allOverrides = append(allOverrides, override)
		}
	}

	// Add library-specific overrides (these can override global ones)
	if library.Rust != nil {
		allOverrides = append(allOverrides, library.Rust.DocumentationOverrides...)
	}

	if len(allOverrides) > 0 {
		sidekickCfg.CommentOverrides = make([]sidekickconfig.DocumentationOverride, len(allOverrides))
		for i, override := range allOverrides {
			sidekickCfg.CommentOverrides[i] = sidekickconfig.DocumentationOverride{
				ID:      override.ID,
				Match:   override.Match,
				Replace: override.Replace,
			}
		}
	}

	// Add pagination overrides if any
	if library.Rust != nil && len(library.Rust.PaginationOverrides) > 0 {
		sidekickCfg.PaginationOverrides = make([]sidekickconfig.PaginationOverride, len(library.Rust.PaginationOverrides))
		for i, override := range library.Rust.PaginationOverrides {
			sidekickCfg.PaginationOverrides[i] = sidekickconfig.PaginationOverride{
				ID:        override.ID,
				ItemField: override.ItemField,
			}
		}
	}

	return sidekickCfg, nil
}

func buildCodec(library *config.Library) map[string]string {
	codec := make(map[string]string)

	// Add version if specified
	if library.Version != "" {
		codec["version"] = library.Version
	}

	// Add release level if specified
	if library.ReleaseLevel != "" {
		codec["release-level"] = library.ReleaseLevel
	}

	// Add package name override if specified
	if library.Name != "" {
		codec["package-name-override"] = library.Name
	}

	// Add copyright year if specified
	if library.CopyrightYear != "" {
		codec["copyright-year"] = library.CopyrightYear
	}

	// Return codec if no Rust config
	if library.Rust == nil {
		return codec
	}

	rust := library.Rust
	if rust.NameOverrides != "" {
		codec["name-overrides"] = rust.NameOverrides
	}
	if rust.ModulePath != "" {
		codec["module-path"] = rust.ModulePath
	}
	if library.Publish != nil && library.Publish.Disabled {
		codec["not-for-publication"] = "true"
	}
	if len(rust.DisabledRustdocWarnings) > 0 {
		codec["disabled-rustdoc-warnings"] = strings.Join(rust.DisabledRustdocWarnings, ",")
	}
	if len(rust.DisabledClippyWarnings) > 0 {
		codec["disabled-clippy-warnings"] = strings.Join(rust.DisabledClippyWarnings, ",")
	}
	if rust.TemplateOverride != "" {
		codec["template-override"] = rust.TemplateOverride
	}
	if rust.IncludeGrpcOnlyMethods {
		codec["include-grpc-only-methods"] = "true"
	}
	if rust.PerServiceFeatures {
		codec["per-service-features"] = "true"
	}
	if len(rust.DefaultFeatures) > 0 {
		codec["default-features"] = strings.Join(rust.DefaultFeatures, ",")
	}
	if rust.DetailedTracingAttributes {
		codec["detailed-tracing-attributes"] = "true"
	}
	if rust.HasVeneer {
		codec["has-veneer"] = "true"
	}
	if len(rust.ExtraModules) > 0 {
		codec["extra-modules"] = strings.Join(rust.ExtraModules, ",")
	}
	if rust.RoutingRequired {
		codec["routing-required"] = "true"
	}
	if rust.GenerateSetterSamples {
		codec["generate-setter-samples"] = "true"
	}

	for _, dep := range rust.PackageDependencies {
		codec["package:"+dep.Name] = formatPackageDependency(&dep)
	}

	return codec
}

func formatPackageDependency(dep *config.RustPackageDependency) string {
	var parts []string

	if dep.Package != "" {
		parts = append(parts, "package="+dep.Package)
	}
	if dep.Source != "" {
		parts = append(parts, "source="+dep.Source)
	}
	if dep.ForceUsed {
		parts = append(parts, "force-used=true")
	}
	if dep.UsedIf != "" {
		parts = append(parts, "used-if="+dep.UsedIf)
	}
	if dep.Feature != "" {
		parts = append(parts, "feature="+dep.Feature)
	}
	// Note: Workspace field is not passed to sidekick as it doesn't support it.
	// Sidekick templates handle workspace dependencies automatically.

	return strings.Join(parts, ",")
}

func convertPackageDependencies(deps []*config.RustPackageDependency) []config.RustPackageDependency {
	result := make([]config.RustPackageDependency, len(deps))
	for i, dep := range deps {
		if dep != nil {
			result[i] = *dep
		}
	}
	return result
}

// mergePackageDependencies merges default package dependencies with library-specific ones.
// Library-specific dependencies override defaults with the same name.
func mergePackageDependencies(defaults []*config.RustPackageDependency, librarySpecific []config.RustPackageDependency) []config.RustPackageDependency {
	// Create a map of library-specific dependencies by name for quick lookup
	libMap := make(map[string]config.RustPackageDependency)
	for _, dep := range librarySpecific {
		libMap[dep.Name] = dep
	}

	// Start with all default dependencies
	result := convertPackageDependencies(defaults)

	// Override with library-specific dependencies and track which names we've seen
	seenNames := make(map[string]bool)
	for i, dep := range result {
		if override, ok := libMap[dep.Name]; ok {
			result[i] = override
			seenNames[override.Name] = true
		} else {
			seenNames[dep.Name] = true
		}
	}

	// Add any library-specific dependencies that weren't in defaults
	for _, dep := range librarySpecific {
		if !seenNames[dep.Name] {
			result = append(result, dep)
		}
	}

	return result
}
