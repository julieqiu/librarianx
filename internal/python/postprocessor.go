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
	"path/filepath"

	"github.com/googleapis/librarian/internal/golang/execv"
)

// postProcessorConfig holds configuration for post-processing.
type postProcessorConfig struct {
	// OutputDir is the directory containing the generated code.
	OutputDir string
	// LibraryPath is the relative path to the library (e.g., "packages/google-cloud-language").
	LibraryPath string
}

// RunSynthtool runs synthtool to post-process generated code.
// It applies templates, runs formatters, and copies code from staging to final location.
func runSynthtool(ctx context.Context, cfg *postProcessorConfig) error {
	// Check if custom owlbot.py exists
	owlbotPath := filepath.Join(cfg.OutputDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err == nil {
		return runCustomOwlbot(ctx, cfg.OutputDir, owlbotPath)
	}

	// Check if noxfile.py exists (GAPIC library)
	noxfilePath := filepath.Join(cfg.OutputDir, cfg.LibraryPath, "noxfile.py")
	if _, err := os.Stat(noxfilePath); err == nil {
		return runDefaultSynthtool(ctx, cfg.OutputDir, cfg.LibraryPath)
	}

	// Proto-only library - run formatters directly
	return runFormatters(ctx, cfg.OutputDir, cfg.LibraryPath)
}

// runCustomOwlbot runs a custom owlbot.py script.
func runCustomOwlbot(ctx context.Context, outputDir, owlbotPath string) error {
	args := []string{"python3", owlbotPath}
	if err := execv.Run(ctx, args, outputDir); err != nil {
		return fmt.Errorf("owlbot.py failed: %w", err)
	}
	return nil
}

// runDefaultSynthtool runs the default synthtool post-processor.
func runDefaultSynthtool(ctx context.Context, outputDir, libraryPath string) error {
	args := []string{
		"python3", "-c",
		fmt.Sprintf("from synthtool.languages import python_mono_repo; python_mono_repo.owlbot_main('%s')", libraryPath),
	}
	if err := execv.Run(ctx, args, outputDir); err != nil {
		return fmt.Errorf("synthtool failed: %w", err)
	}
	return nil
}

// runFormatters runs isort and black on proto-only libraries.
func runFormatters(ctx context.Context, outputDir, libraryPath string) error {
	libraryFullPath := filepath.Join(outputDir, libraryPath)

	// Run isort
	isortArgs := []string{"isort", libraryFullPath}
	if err := execv.Run(ctx, isortArgs, outputDir); err != nil {
		return fmt.Errorf("isort failed: %w", err)
	}

	// Run black
	blackArgs := []string{"black", libraryFullPath}
	if err := execv.Run(ctx, blackArgs, outputDir); err != nil {
		return fmt.Errorf("black failed: %w", err)
	}

	return nil
}

// CopyREADME copies README.rst from the library root to the docs/ directory.
func copyREADME(outputDir, libraryPath string) error {
	readmeSrc := filepath.Join(outputDir, libraryPath, "README.rst")
	readmeDst := filepath.Join(outputDir, libraryPath, "docs", "README.rst")

	// Check if README.rst exists
	if _, err := os.Stat(readmeSrc); os.IsNotExist(err) {
		// No README.rst, nothing to copy
		return nil
	}

	// Create docs directory if it doesn't exist
	docsDir := filepath.Join(outputDir, libraryPath, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	// Copy README.rst
	data, err := os.ReadFile(readmeSrc)
	if err != nil {
		return fmt.Errorf("failed to read README.rst: %w", err)
	}

	if err := os.WriteFile(readmeDst, data, 0644); err != nil {
		return fmt.Errorf("failed to write README.rst to docs: %w", err)
	}

	return nil
}
