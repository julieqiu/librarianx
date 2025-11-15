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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language/golang/generate"
	"github.com/googleapis/librarian/internal/language/golang/release"
)

// Generate generates Go client libraries from API definitions.
// It downloads googleapis, sets up the generation environment, and runs the generator.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, googleapisRoot string) (err error) {
	// Determine output directory
	outputDir := "{name}/"
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		outputDir = cfg.Generate.Output
	}

	location, err := library.GeneratedLocation(outputDir)
	if err != nil {
		return fmt.Errorf("failed to determine output location: %w", err)
	}

	// Convert to absolute path
	absLocation, err := filepath.Abs(location)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", location, err)
	}

	// Create temporary librarian directory structure
	librarianDir, err := os.MkdirTemp("", "librarian-go-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary librarian directory: %w", err)
	}
	defer func() {
		cerr := os.RemoveAll(librarianDir)
		if err == nil {
			err = cerr
		}
	}()

	// Create generator-input directory
	generatorInputDir := filepath.Join(librarianDir, "generator-input")
	if err := os.MkdirAll(generatorInputDir, 0755); err != nil {
		return fmt.Errorf("failed to create generator-input directory: %w", err)
	}

	// Create minimal repo-config.yaml
	repoConfigPath := filepath.Join(generatorInputDir, "repo-config.yaml")
	repoConfigContent := "modules: []\n"
	if err := os.WriteFile(repoConfigPath, []byte(repoConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write repo-config.yaml: %w", err)
	}

	// Create generate-request.json
	generateRequest := struct {
		ID   string `json:"id"`
		APIs []struct {
			Path string `json:"path"`
		} `json:"apis"`
	}{
		ID: library.Name,
	}

	for _, apiPath := range library.Apis {
		generateRequest.APIs = append(generateRequest.APIs, struct {
			Path string `json:"path"`
		}{Path: apiPath})
	}

	requestJSON, err := json.Marshal(generateRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal generate request: %w", err)
	}

	requestPath := filepath.Join(librarianDir, "generate-request.json")
	if err := os.WriteFile(requestPath, requestJSON, 0644); err != nil {
		return fmt.Errorf("failed to write generate-request.json: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(absLocation, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Call generate.Generate
	genCfg := &generate.Config{
		LibrarianDir:         librarianDir,
		InputDir:             generatorInputDir,
		OutputDir:            absLocation,
		SourceDir:            googleapisRoot,
		DisablePostProcessor: false,
	}

	if err := generate.Generate(ctx, genCfg); err != nil {
		return fmt.Errorf("go generation failed: %w", err)
	}

	return nil
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
