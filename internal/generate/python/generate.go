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
	"github.com/googleapis/librarian/internal/generate/golang/execv"
)

// Test substitution vars.
var (
	execvRun = execv.Run
)

// Generate is the main entrypoint for the Python generate command.
// It orchestrates the entire generation process:
//
//  1. For each API in the library:
//     - Construct protoc command
//     - Run protoc with --python_gapic_out to generate code directly to output
//  2. Generate .repo-metadata.json from service_yaml
//  3. Run synthtool post-processor in place
//  4. Copy README.rst to docs/
func Generate(ctx context.Context, lib *config.Library, outputDir, sourceDir string, disablePostProcessor bool) error {
	slog.Debug("python generate: started")

	// Generate code for each API
	for _, apiPath := range lib.Apis {
		if err := generateAPI(ctx, sourceDir, outputDir, apiPath); err != nil {
			return fmt.Errorf("failed to generate API %s: %w", apiPath, err)
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

// generateAPI generates code for a single API.
func generateAPI(ctx context.Context, sourceDir, outputDir, apiPath string) error {
	apiServiceDir := filepath.Join(sourceDir, apiPath)
	slog.Info("processing api", "service_dir", apiServiceDir)

	// Determine if this is a GAPIC or proto-only library
	// For now, assume all APIs are GAPIC libraries
	// TODO: Parse BUILD.bazel to determine library type
	isGapic := true

	var cmd *ProtocCommand
	var err error

	if isGapic {
		// Build GAPIC command
		opts := &GapicOptions{
			Transport:        "grpc+rest",
			RestNumericEnums: true,
		}
		cmd, err = BuildGapicCommand(apiPath, sourceDir, outputDir, opts)
	} else {
		// Build proto-only command
		cmd, err = BuildProtoCommand(apiPath, sourceDir, outputDir)
	}

	if err != nil {
		return fmt.Errorf("failed to build protoc command: %w", err)
	}

	// Run protoc
	args := append([]string{cmd.Command}, cmd.Args...)
	if err := execvRun(ctx, args, outputDir); err != nil {
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
