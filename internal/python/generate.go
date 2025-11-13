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

// Package python provides the core generation logic for creating Python client libraries.
package python

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/golang/execv"
)

// Test substitution vars.
var (
	execvRun = execv.Run
)

// Generate generates Python client library code from configuration.
// It orchestrates the entire generation process:
//
//  1. Determine output directory from configuration
//  2. Create output directory if it doesn't exist
//  3. For each API in the library:
//     - Construct protoc command
//     - Run protoc with --python_gapic_out to generate code directly to output
//  4. Generate .repo-metadata.json from service_yaml
//  5. Run synthtool post-processor in place (if enabled)
//  6. Copy README.rst to docs/
func Generate(ctx context.Context, cfg *config.Config, lib *config.Library, googleapisRoot string) error {
	// Determine output directory
	outputDir := "{name}/"
	if cfg.Generate != nil && cfg.Generate.Output != "" {
		outputDir = cfg.Generate.Output
	}

	location, err := lib.GeneratedLocation(outputDir)
	if err != nil {
		return fmt.Errorf("failed to determine output location: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(location, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run Python generator (with post-processor disabled by default)
	// TODO(https://github.com/googleapis/librarian/issues/XXX): Make post-processor configurable via librarian.yaml
	disablePostProcessor := true
	if err := generate(ctx, lib, location, googleapisRoot, disablePostProcessor); err != nil {
		return fmt.Errorf("python generation failed: %w", err)
	}

	fmt.Printf("Generated Python library %q at %s\n", lib.Name, location)
	return nil
}

// generate is the internal implementation of Python code generation.
// It orchestrates the protoc execution and post-processing steps.
func generate(ctx context.Context, lib *config.Library, outputDir, sourceDir string, disablePostProcessor bool) error {
	slog.Debug("python generate: started")

	// Use parsed API configurations if available, otherwise fall back to API paths
	if lib.Generate != nil && len(lib.Generate.APIs) > 0 {
		// Generate code using parsed API configurations
		for _, apiConfig := range lib.Generate.APIs {
			if err := generateAPIFromConfig(ctx, sourceDir, outputDir, &apiConfig); err != nil {
				return fmt.Errorf("failed to generate API %s: %w", apiConfig.Path, err)
			}
		}
	} else {
		// Fallback: generate code from API paths without parsed config
		for _, apiPath := range lib.Apis {
			if err := generateAPI(ctx, sourceDir, outputDir, apiPath); err != nil {
				return fmt.Errorf("failed to generate API %s: %w", apiPath, err)
			}
		}
	}

	// Run post-processor (synthtool) if enabled
	if !disablePostProcessor {
		if err := runPostProcessor(ctx, outputDir, lib.Name); err != nil {
			return fmt.Errorf("post-processing failed: %w", err)
		}
	}

	slog.Debug("python generate: finished")
	return nil
}

// generateAPIFromConfig generates code for a single API using parsed configuration.
func generateAPIFromConfig(ctx context.Context, sourceDir, outputDir string, apiConfig *config.API) error {
	apiServiceDir := filepath.Join(sourceDir, apiConfig.Path)
	slog.Info("processing api", "service_dir", apiServiceDir, "has_gapic", apiConfig.HasGAPIC)

	// Convert output directory to absolute path to avoid issues with protoc working directory
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}

	var cmd *protocCommand

	if apiConfig.HasGAPIC {
		// Build GAPIC command using parsed configuration
		opts := &gapicOptions{
			GrpcServiceConfig: apiConfig.GRPCServiceConfig,
			ServiceYAML:       apiConfig.ServiceYAML,
			Transport:         apiConfig.Transport,
			RestNumericEnums:  apiConfig.RestNumericEnums,
		}
		if apiConfig.Python != nil {
			opts.OptArgs = apiConfig.Python.OptArgs
		}
		cmd, err = buildGapicCommand(apiConfig.Path, sourceDir, absOutputDir, opts)
	} else {
		// Build proto-only command
		cmd, err = buildProtoCommand(apiConfig.Path, sourceDir, absOutputDir)
	}

	if err != nil {
		return fmt.Errorf("failed to build protoc command: %w", err)
	}

	// Run protoc from current directory, not from output directory
	args := append([]string{cmd.Command}, cmd.Args...)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := execvRun(ctx, args, cwd); err != nil {
		return fmt.Errorf("protoc failed: %w", err)
	}

	return nil
}

// generateAPI generates code for a single API.
func generateAPI(ctx context.Context, sourceDir, outputDir, apiPath string) error {
	apiServiceDir := filepath.Join(sourceDir, apiPath)
	slog.Info("processing api", "service_dir", apiServiceDir)

	// Convert output directory to absolute path to avoid issues with protoc working directory
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}

	// Determine if this is a GAPIC or proto-only library
	// For now, assume all APIs are GAPIC libraries
	// TODO: Parse BUILD.bazel to determine library type
	isGapic := true

	var cmd *protocCommand

	if isGapic {
		// Build GAPIC command
		opts := &gapicOptions{
			Transport:        "grpc+rest",
			RestNumericEnums: true,
		}
		cmd, err = buildGapicCommand(apiPath, sourceDir, absOutputDir, opts)
	} else {
		// Build proto-only command
		cmd, err = buildProtoCommand(apiPath, sourceDir, absOutputDir)
	}

	if err != nil {
		return fmt.Errorf("failed to build protoc command: %w", err)
	}

	// Run protoc from current directory, not from output directory
	args := append([]string{cmd.Command}, cmd.Args...)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := execvRun(ctx, args, cwd); err != nil {
		return fmt.Errorf("protoc failed: %w", err)
	}

	return nil
}

// runPostProcessor runs synthtool to post-process generated code.
func runPostProcessor(ctx context.Context, outputDir, libraryName string) error {
	slog.Debug("python generate: running post-processor")

	// Check if custom owlbot.py exists
	owlbotPath := filepath.Join(outputDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err == nil {
		// Run custom owlbot.py
		args := []string{"python3", owlbotPath}
		if err := execvRun(ctx, args, outputDir); err != nil {
			return fmt.Errorf("owlbot.py failed: %w", err)
		}
		return nil
	}

	// Run default synthtool
	// synthtool.languages.python_mono_repo.owlbot_main expects the relative library path
	libraryPath := filepath.Join("packages", libraryName)
	args := []string{
		"python3", "-c",
		fmt.Sprintf("from synthtool.languages import python_mono_repo; python_mono_repo.owlbot_main('%s')", libraryPath),
	}
	if err := execvRun(ctx, args, outputDir); err != nil {
		return fmt.Errorf("synthtool failed: %w", err)
	}

	return nil
}
