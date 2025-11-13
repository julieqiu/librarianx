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

// Package rust provides functionality for generating, releasing, and publishing Rust client libraries.
package rust

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/sidekick"
)

// Generate generates Rust client libraries from API definitions.
// It calls sidekick.Run() with rust-generate internally.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, googleapisRoot string) error {
	// Validate Rust-specific requirements
	if len(library.Apis) != 1 {
		return fmt.Errorf("rust generation requires exactly one API per library, got %d for library %q", len(library.Apis), library.Name)
	}

	// Determine output directory
	outputDir := "{name}/"
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		outputDir = cfg.Generate.Output
	}

	location, err := library.GeneratedLocation(outputDir)
	if err != nil {
		return fmt.Errorf("failed to determine output location: %w", err)
	}

	// Build sidekick command line arguments
	args := []string{
		"rust-generate",
		"--specification-source", library.Apis[0],
		"--output", location,
		"--language", "rust",
	}

	// Add googleapis source if available
	if googleapisRoot != "" {
		args = append(args, "--source-option", fmt.Sprintf("googleapis-root=%s", googleapisRoot))
	}

	// Add copyright year if specified
	if library.CopyrightYear > 0 {
		args = append(args, "--codec-option", fmt.Sprintf("copyright-year=%d", library.CopyrightYear))
	}

	// Run sidekick
	if err := sidekick.Run(args); err != nil {
		return fmt.Errorf("sidekick rust-generate failed: %w", err)
	}

	fmt.Printf("Generated Rust library %q at %s\n", library.Name, location)
	return nil
}

// Release performs Rust-specific release preparation.
// It runs cargo test --all-features, updates Cargo.toml version, and creates/updates CHANGELOG.md.
func Release(ctx context.Context, repoRoot string, lib *config.Library, version string) error {
	// This will call sidekick rust-bump-versions which:
	// 1. Finds all crates that changed since last release
	// 2. Updates Cargo.toml versions
	// 3. Runs cargo semver-checks
	// 4. Updates CHANGELOG.md
	args := []string{
		"rust-bump-versions",
	}

	if err := sidekick.Run(args); err != nil {
		return fmt.Errorf("sidekick rust-bump-versions failed: %w", err)
	}

	fmt.Printf("Prepared Rust library %q for release version %s\n", lib.Name, version)
	return nil
}

// Publish publishes to crates.io.
// It verifies git tags exist (local and remote), runs cargo semver-checks (optional), and executes cargo publish.
func Publish(ctx context.Context, repoRoot string, lib *config.Library, version string) error {
	// This will call sidekick rust-publish which:
	// 1. Verifies git tags exist
	// 2. Runs cargo workspaces plan to validate publication plan
	// 3. Runs cargo semver-checks (unless --skip-semver-checks)
	// 4. Runs cargo workspaces publish
	args := []string{
		"rust-publish",
	}

	if err := sidekick.Run(args); err != nil {
		return fmt.Errorf("sidekick rust-publish failed: %w", err)
	}

	fmt.Printf("Published Rust library %q version %s to crates.io\n", lib.Name, version)
	return nil
}
