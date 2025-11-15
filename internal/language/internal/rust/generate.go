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
func Generate(ctx context.Context, library *config.Library, googleapisDir, outdir string) error {
	sidekickConfig := toSidekickConfig(library, googleapisDir)
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	return sidekickrust.Generate(model, outdir, sidekickConfig)
}

func toSidekickConfig(library *config.Library, source, serviceConfig, googleapisDir string) *sidekickconfig.Config {
	sidekickCfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			Language:            "rust",
			SpecificationFormat: "protobuf",
			ServiceConfig:       serviceConfig,
			SpecificationSource: source,
		},
		Source: map[string]string{
			"googleapis-root": googleapisDir,
		},
		Codec: map[string]string{
			"name-overrides":              rust.NameOverrides,
			"module-path":                 rust.ModulePath,
			"not-for-publication":         boolToString(rust.NotForPublication),
			"disabled-rustdoc-warnings":   strings.Join(rust.DisabledRustdocWarnings, ","),
			"disabled-clippy-warnings":    strings.Join(rust.DisabledClippyWarnings, ","),
			"template-override":           rust.TemplateOverride,
			"include-grpc-only-methods":   boolToString(rust.IncludeGrpcOnlyMethods),
			"per-service-features":        boolToString(rust.PerServiceFeatures),
			"default-features":            strings.Join(rust.DefaultFeatures, ","),
			"detailed-tracing-attributes": boolToString(rust.DetailedTracingAttributes),
			"has-veneer":                  boolToString(rust.HasVeneer),
			"extra-modules":               strings.Join(rust.ExtraModules, ","),
			"routing-required":            boolToString(rust.RoutingRequired),
			"generate-setter-samples":     boolToString(rust.GenerateSetterSamples),
		},
	}
	if library.Rust != nil {
		mapRustCodecOptions(sidekickCfg.Codec, library.Rust)
	}
	return sidekickCfg
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return ""
}
