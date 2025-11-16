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

package python

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Generate generates a Python client library.
func Generate(ctx context.Context, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
	// Determine output directory
	outdir := library.Path
	if outdir == "" {
		// Use default output pattern if no explicit path
		if defaults != nil && defaults.Output != "" {
			outdir = strings.ReplaceAll(defaults.Output, "{name}", library.Name)
		} else {
			outdir = filepath.Join("packages", library.Name)
		}
	}
	outdir = filepath.Join(defaultOutput, outdir)

	// Create output directory
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get API paths to generate
	apiPaths := library.APIs
	if len(apiPaths) == 0 && library.API != "" {
		apiPaths = []string{library.API}
	}

	if len(apiPaths) == 0 {
		return fmt.Errorf("no APIs specified for library %s", library.Name)
	}

	// Get transport from library or defaults
	transport := library.Transport
	if transport == "" && defaults != nil && defaults.Generate != nil {
		transport = defaults.Generate.Transport
	}

	// Get rest_numeric_enums from library or defaults
	restNumericEnums := defaults != nil && defaults.Generate != nil && defaults.Generate.RestNumericEnums
	if library.RestNumericEnums != nil {
		restNumericEnums = *library.RestNumericEnums
	}

	// Generate each API
	for _, apiPath := range apiPaths {
		if err := generateAPI(ctx, apiPath, library, googleapisDir, serviceConfigPath, outdir, transport, restNumericEnums); err != nil {
			return fmt.Errorf("failed to generate API %s: %w", apiPath, err)
		}
	}

	return nil
}

// generateAPI generates code for a single API.
func generateAPI(ctx context.Context, apiPath string, library *config.Library, googleapisDir, serviceConfigPath, outdir, transport string, restNumericEnums bool) error {
	// Build generator options
	var opts []string

	// Add transport option
	if transport != "" {
		opts = append(opts, fmt.Sprintf("transport=%s", transport))
	}

	// Add rest_numeric_enums option
	if restNumericEnums {
		opts = append(opts, "rest-numeric-enums")
	}

	// Add Python-specific options
	if library.Python != nil && len(library.Python.OptArgs) > 0 {
		opts = append(opts, library.Python.OptArgs...)
	}

	// Add service config if provided
	if serviceConfigPath != "" {
		opts = append(opts, fmt.Sprintf("grpc-service-config=%s", serviceConfigPath))
	}

	// Build protoc command
	protoPattern := filepath.Join(apiPath, "*.proto")

	args := []string{
		protoPattern,
		fmt.Sprintf("--python_gapic_out=%s", outdir),
	}

	// Add options if any
	if len(opts) > 0 {
		optString := "metadata," + strings.Join(opts, ",")
		args = append(args, fmt.Sprintf("--python_gapic_opt=%s", optString))
	}

	// Construct the full command as a shell command
	// We need shell=true because of the glob pattern *.proto
	cmdStr := "protoc " + strings.Join(args, " ")

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = googleapisDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("protoc command failed: %w", err)
	}

	return nil
}
