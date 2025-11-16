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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir, outdir, serviceConfigPath string) error {
	sidekickConfig := toSidekickConfig(library, googleapisDir, serviceConfigPath)
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	return sidekickrust.Generate(model, outdir, sidekickConfig)
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
		Codec: map[string]string{
			"name-overrides":              library.Rust.NameOverrides,
			"module-path":                 library.Rust.ModulePath,
			"not-for-publication":         boolToString(library.Rust.NotForPublication),
			"disabled-rustdoc-warnings":   strings.Join(library.Rust.DisabledRustdocWarnings, ","),
			"disabled-clippy-warnings":    strings.Join(library.Rust.DisabledClippyWarnings, ","),
			"template-override":           library.Rust.TemplateOverride,
			"include-grpc-only-methods":   boolToString(library.Rust.IncludeGrpcOnlyMethods),
			"per-service-features":        boolToString(library.Rust.PerServiceFeatures),
			"default-features":            strings.Join(library.Rust.DefaultFeatures, ","),
			"detailed-tracing-attributes": boolToString(library.Rust.DetailedTracingAttributes),
			"has-veneer":                  boolToString(library.Rust.HasVeneer),
			"extra-modules":               strings.Join(library.Rust.ExtraModules, ","),
			"routing-required":            boolToString(library.Rust.RoutingRequired),
			"generate-setter-samples":     boolToString(library.Rust.GenerateSetterSamples),
		},
	}
	return sidekickCfg
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return ""
}
