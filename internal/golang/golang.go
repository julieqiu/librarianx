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

// Package golang provides the main entry points for Go library generation and release.
package golang

import (
	"context"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/golang/generate"
	"github.com/googleapis/librarian/internal/golang/release"
)

// Generate generates Go client libraries from API definitions.
// It is a thin wrapper around generate.Generate.
func Generate(ctx context.Context, cfg *generate.Config) error {
	return generate.Generate(ctx, cfg)
}

// Release performs Go-specific release preparation.
// It is a thin wrapper around release.Release.
func Release(ctx context.Context, repoRoot string, lib *config.Library, version string, changes []*release.Change) error {
	return release.Release(ctx, repoRoot, lib, version, changes)
}

// Publish verifies pkg.go.dev indexing.
// It is a thin wrapper around release.Publish.
func Publish(ctx context.Context, repoRoot string, lib *config.Library, version string) error {
	return release.Publish(ctx, repoRoot, lib, version)
}
