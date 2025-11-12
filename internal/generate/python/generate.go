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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/generate/golang/execv"
	"github.com/googleapis/librarian/internal/generate/golang/request"
)

// Test substitution vars.
var (
	execvRun     = execv.Run
	requestParse = request.ParseLibrary
)

// Config holds the configuration for the Python generate command.
type Config struct {
	// LibrarianDir is the path to the librarian-tool input directory.
	// It is expected to contain the generate-request.json file.
	LibrarianDir string
	// OutputDir is the path to the directory where generated code is written.
	OutputDir string
	// SourceDir is the path to a complete checkout of the googleapis repository.
	SourceDir string
	// StagingDir is the path to the owl-bot-staging directory.
	StagingDir string
	// DisablePostProcessor controls whether synthtool is run.
	DisablePostProcessor bool
}

// Validate ensures that the configuration is valid.
func (c *Config) Validate() error {
	if c.LibrarianDir == "" {
		return errors.New("librarian directory must be set")
	}
	if c.OutputDir == "" {
		return errors.New("output directory must be set")
	}
	if c.SourceDir == "" {
		return errors.New("source directory must be set")
	}
	if c.StagingDir == "" {
		return errors.New("staging directory must be set")
	}
	return nil
}

// Generate is the main entrypoint for the Python generate command.
// It orchestrates the entire generation process:
//
//  1. Read generate-request.json
//  2. For each API:
//     - Construct protoc command
//     - Run protoc with --python_gapic_out to generate code
//     - Stage code in owl-bot-staging directory
//  3. Generate .repo-metadata.json from service_yaml
//  4. Run synthtool post-processor
//  5. Copy README.rst to docs/
//  6. Write generated code to output directory
func Generate(ctx context.Context, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	slog.Debug("python generate: started")

	generateReq, err := readGenerateReq(cfg.LibrarianDir)
	if err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}

	// Create staging directory
	if err := os.MkdirAll(cfg.StagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	// Generate code for each API
	for _, api := range generateReq.APIs {
		if err := generateAPI(ctx, cfg, generateReq, &api); err != nil {
			return fmt.Errorf("failed to generate API %s: %w", api.Path, err)
		}
	}

	// Run post-processor (synthtool) if enabled
	if !cfg.DisablePostProcessor {
		if err := runPostProcessor(ctx, cfg, generateReq); err != nil {
			return fmt.Errorf("post-processing failed: %w", err)
		}
	}

	slog.Debug("python generate: finished")
	return nil
}

// generateAPI generates code for a single API.
func generateAPI(ctx context.Context, cfg *Config, lib *request.Library, api *request.API) error {
	apiServiceDir := filepath.Join(cfg.SourceDir, api.Path)
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
			GrpcServiceConfig: api.ServiceConfig,
			Transport:         "grpc+rest",
			RestNumericEnums:  true,
		}
		cmd, err = BuildGapicCommand(api, cfg.SourceDir, cfg.StagingDir, opts)
	} else {
		// Build proto-only command
		cmd, err = BuildProtoCommand(api, cfg.SourceDir, cfg.StagingDir)
	}

	if err != nil {
		return fmt.Errorf("failed to build protoc command: %w", err)
	}

	// Run protoc
	args := append([]string{cmd.Command}, cmd.Args...)
	if err := execvRun(ctx, args, cfg.OutputDir); err != nil {
		return fmt.Errorf("protoc failed: %w", err)
	}

	return nil
}

// runPostProcessor runs synthtool to post-process generated code.
func runPostProcessor(ctx context.Context, cfg *Config, lib *request.Library) error {
	slog.Debug("python generate: running post-processor")

	// Check if custom owlbot.py exists
	owlbotPath := filepath.Join(cfg.OutputDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err == nil {
		// Run custom owlbot.py
		args := []string{"python3", owlbotPath}
		if err := execvRun(ctx, args, cfg.OutputDir); err != nil {
			return fmt.Errorf("owlbot.py failed: %w", err)
		}
		return nil
	}

	// Run default synthtool
	// synthtool.languages.python_mono_repo.owlbot_main expects the relative library path
	libraryPath := filepath.Join("packages", lib.ID)
	args := []string{
		"python3", "-c",
		fmt.Sprintf("from synthtool.languages import python_mono_repo; python_mono_repo.owlbot_main('%s')", libraryPath),
	}
	if err := execvRun(ctx, args, cfg.OutputDir); err != nil {
		return fmt.Errorf("synthtool failed: %w", err)
	}

	return nil
}

// readGenerateReq reads generate-request.json from the librarian-tool input directory.
func readGenerateReq(librarianDir string) (*request.Library, error) {
	reqPath := filepath.Join(librarianDir, "generate-request.json")
	slog.Debug("python generate: reading generate request", "path", reqPath)

	generateReq, err := requestParse(reqPath)
	if err != nil {
		return nil, err
	}
	slog.Debug("python generate: successfully read request", "library_id", generateReq.ID)
	return generateReq, nil
}
