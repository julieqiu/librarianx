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

package golang

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/julieqiu/librarianx/internal/config"
)

// Generate generates a Go client library.
// Files and directories specified in library.Keep will be preserved during regeneration.
func Generate(ctx context.Context, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
	// Determine output directory
	outdir := library.Path
	if outdir == "" {
		// Use default output pattern if no explicit path
		if defaults != nil {
			outdir = strings.ReplaceAll(defaults.Output, "{name}", library.Name)
		}
	}

	// Convert to absolute path since protoc runs from a different directory
	var err error
	outdir, err = filepath.Abs(outdir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}
	fmt.Println(outdir)

	// Clean output directory before generation
	if err := cleanOutputDirectory(outdir, library.Keep); err != nil {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get APIs to generate
	apis := config.GetLibraryAPIs(library)
	if len(apis) == 0 {
		return fmt.Errorf("no APIs found for library %s", library.Name)
	}

	// Determine transport from library or defaults
	transport := library.Transport
	if transport == "" && defaults != nil && defaults.Generate != nil {
		transport = defaults.Generate.Transport
	}

	// Determine rest_numeric_enums setting
	restNumericEnums := false
	if library.RestNumericEnums != nil {
		restNumericEnums = *library.RestNumericEnums
	} else if defaults != nil && defaults.Generate != nil {
		restNumericEnums = defaults.Generate.RestNumericEnums
	}

	// Generate each API
	for _, apiPath := range apis {
		// Get service config for this API
		apiServiceConfig := library.APIServiceConfigs[apiPath]
		if err := generateAPI(ctx, apiPath, library, googleapisDir, apiServiceConfig, outdir, transport, restNumericEnums); err != nil {
			return fmt.Errorf("failed to generate API %s: %w", apiPath, err)
		}
	}

	// Run post-processing
	if err := postProcess(ctx, outdir); err != nil {
		return fmt.Errorf("failed to post-process: %w", err)
	}

	// Create scaffolding files if they don't exist
	if err := createScaffoldingFiles(outdir, library); err != nil {
		return fmt.Errorf("failed to create scaffolding files: %w", err)
	}

	return nil
}

// generateAPI generates code for a single API using protoc with Go plugins.
func generateAPI(ctx context.Context, apiPath string, library *config.Library, googleapisDir, serviceConfigPath, outdir, transport string, restNumericEnums bool) error {
	// Build protoc command arguments
	protoPattern := filepath.Join(apiPath, "*.proto")

	// Base arguments for all Go generation
	args := []string{
		protoPattern,
		fmt.Sprintf("--go_out=%s", outdir),
		fmt.Sprintf("--go-grpc_out=%s", outdir),
		fmt.Sprintf("--go_gapic_out=%s", outdir),
	}

	// Build GAPIC options
	var gapicOpts []string

	// Import path from library config
	if library.Go != nil && library.Go.ImportPath != "" {
		gapicOpts = append(gapicOpts, fmt.Sprintf("go-gapic-package=%s", library.Go.ImportPath))
	}

	// Transport option
	if transport != "" {
		gapicOpts = append(gapicOpts, fmt.Sprintf("transport=%s", transport))
	}

	// Rest numeric enums option
	if restNumericEnums {
		gapicOpts = append(gapicOpts, "rest-numeric-enums")
	}

	// gRPC service config (retry/timeout settings)
	grpcConfigPath := ""
	if library.GRPCServiceConfig != "" {
		// GRPCServiceConfig is relative to the API directory
		grpcConfigPath = filepath.Join(googleapisDir, apiPath, library.GRPCServiceConfig)
	} else {
		// Auto-discover: look for *_grpc_service_config.json in the API directory
		apiDir := filepath.Join(googleapisDir, apiPath)
		matches, err := filepath.Glob(filepath.Join(apiDir, "*_grpc_service_config.json"))
		if err == nil && len(matches) > 0 {
			grpcConfigPath = matches[0]
		}
	}
	if grpcConfigPath != "" {
		gapicOpts = append(gapicOpts, fmt.Sprintf("grpc-service-config=%s", grpcConfigPath))
	}

	// Service YAML (API metadata) if provided
	if serviceConfigPath != "" {
		gapicOpts = append(gapicOpts, fmt.Sprintf("api-service-config=%s", serviceConfigPath))
	}

	// Metadata generation
	if library.Go != nil && library.Go.Metadata {
		gapicOpts = append(gapicOpts, "metadata")
	}

	// Add GAPIC options to command
	if len(gapicOpts) > 0 {
		for _, opt := range gapicOpts {
			args = append(args, fmt.Sprintf("--go_gapic_opt=%s", opt))
		}
	}

	cmdStr := "protoc " + strings.Join(args, " ")

	// Debug: print the protoc command
	fmt.Fprintf(os.Stderr, "\nRunning: %s\n", cmdStr)

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = googleapisDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("protoc command failed: %w", err)
	}

	return nil
}

// postProcess runs Go-specific post-processing steps.
func postProcess(ctx context.Context, outdir string) error {
	// Run goimports to format imports
	cmd := exec.CommandContext(ctx, "goimports", "-w", ".")
	cmd.Dir = outdir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Non-fatal: goimports might not be installed
		fmt.Fprintf(os.Stderr, "Warning: goimports failed (skipping): %v\n", err)
	}

	// Run go mod tidy if go.mod exists
	goModPath := filepath.Join(outdir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
		cmd.Dir = outdir
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go mod tidy failed: %w", err)
		}
	}

	return nil
}

// createScaffoldingFiles creates initial files for a new library if they don't exist.
func createScaffoldingFiles(outdir string, library *config.Library) error {
	// Create README.md if it doesn't exist
	readmePath := filepath.Join(outdir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		readme := fmt.Sprintf(`# %s

## Installation

`+"```"+`bash
go get cloud.google.com/go/%s
`+"```"+`
`, library.Name, library.Name)
		if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
			return fmt.Errorf("failed to write README.md: %w", err)
		}
	}

	// Create CHANGES.md if it doesn't exist
	changesPath := filepath.Join(outdir, "CHANGES.md")
	if _, err := os.Stat(changesPath); os.IsNotExist(err) {
		changes := "# Changes\n"
		if err := os.WriteFile(changesPath, []byte(changes), 0644); err != nil {
			return fmt.Errorf("failed to write CHANGES.md: %w", err)
		}
	}

	// Create internal/version.go if it doesn't exist
	internalDir := filepath.Join(outdir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return fmt.Errorf("failed to create internal directory: %w", err)
	}

	versionPath := filepath.Join(internalDir, "version.go")
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		version := `package internal

// Version is the current version of this client library.
const Version = "0.0.0"
`
		if err := os.WriteFile(versionPath, []byte(version), 0644); err != nil {
			return fmt.Errorf("failed to write internal/version.go: %w", err)
		}
	}

	return nil
}
