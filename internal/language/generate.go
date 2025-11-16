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

package language

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language/internal/rust"
)

func Generate(ctx context.Context, name, googleapisDir string) error {
	cfg, err := config.Read("librarian.yaml")
	if err != nil {
		return err
	}

	// Parse name to get API path (for Rust: "google-cloud-secretmanager-v1" → "google/cloud/secretmanager/v1")
	apiPath := strings.ReplaceAll(name, "-", "/")

	// Check if library is explicitly configured
	var library *config.Library
	for _, lib := range cfg.Libraries {
		if lib.API == apiPath {
			library = lib
			break
		}
	}

	// If not found, check if wildcard mode is enabled
	if library == nil {
		if !cfg.Default.Generate.All {
			return fmt.Errorf("library %q not found in configuration and generate.all is false", name)
		}

		// Create minimal library config with defaults
		library = &config.Library{
			API: apiPath,
		}
		applyDefaults(library, cfg.Default)
	}

	// Validate API path exists in googleapis
	apiDir := filepath.Join(googleapisDir, apiPath)
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return fmt.Errorf("API path %q not found in googleapis", apiPath)
	} else if err != nil {
		return err
	}

	// Find service config file
	serviceConfigPath, err := FindServiceConfig(googleapisDir, apiPath)
	if err != nil {
		return err
	}

	// Compute output directory (for Rust: strip "google/" prefix)
	// "google/cloud/secretmanager/v1" → "src/generated/cloud/secretmanager/v1"
	outdir := filepath.Join(cfg.Default.Output, strings.TrimPrefix(apiPath, "google/"))

	return rust.Generate(ctx, library, googleapisDir, outdir, serviceConfigPath)
}

func applyDefaults(library *config.Library, defaults *config.Default) {
	if defaults.Generate != nil {
		if library.Transport == "" {
			library.Transport = defaults.Generate.Transport
		}
		if library.ReleaseLevel == "" {
			library.ReleaseLevel = defaults.Generate.ReleaseLevel
		}
		if library.RestNumericEnums == nil {
			b := defaults.Generate.RestNumericEnums
			library.RestNumericEnums = &b
		}
	}

	// Apply Rust-specific defaults
	if defaults.Rust != nil && library.Rust == nil {
		library.Rust = &config.RustCrate{
			DisabledRustdocWarnings: defaults.Rust.DisabledRustdocWarnings,
		}
	}
}

// FindServiceConfig returns the path to the service config file for the given API.
// For example: (googleapis, "google/cloud/secretmanager/v1") → "google/cloud/secretmanager/v1/secretmanager_v1.yaml"
func FindServiceConfig(googleapisDir, apiPath string) (string, error) {
	parts := strings.Split(apiPath, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid API path: %q", apiPath)
	}
	version := parts[len(parts)-1]

	dir := filepath.Join(googleapisDir, apiPath)
	pattern := filepath.Join(dir, "*_"+version+".yaml")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	// Filter out *_gapic.yaml files
	var configs []string
	for _, m := range matches {
		if !strings.HasSuffix(m, "_gapic.yaml") {
			configs = append(configs, m)
		}
	}

	if len(configs) == 0 {
		return "", fmt.Errorf("no service config found in %q", dir)
	}

	return configs[0], nil
}
