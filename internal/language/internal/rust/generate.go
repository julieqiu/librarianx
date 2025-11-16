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
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir, serviceConfigPath, defaultOutput string) error {
	outdir := filepath.Join(defaultOutput, strings.TrimPrefix(library.API, "google/"))
	sidekickConfig := toSidekickConfig(library, googleapisDir, serviceConfigPath)
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickrust.Generate(model, outdir, sidekickConfig); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "cargo", "fmt", "--manifest-path", filepath.Join(outdir, "Cargo.toml"))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cargo fmt failed: %w\n%s", err, output)
	}

	return nil
}

func toSidekickConfig(library *config.Library, googleapisDir, serviceConfig string) *sidekickconfig.Config {
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
	return sidekickCfg
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
	if rust.NotForPublication {
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
	if rust.CopyrightYear != "" {
		codec["copyright-year"] = rust.CopyrightYear
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

	return strings.Join(parts, ",")
}
