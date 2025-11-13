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
	"fmt"
	"os"
	"path/filepath"
)

// postProcessorConfig holds configuration for post-processing.
type postProcessorConfig struct {
	// OutputDir is the directory containing the generated code.
	OutputDir string
	// LibraryPath is the relative path to the library (e.g., "packages/google-cloud-language").
	LibraryPath string
}

// NOTE: The following functions are deprecated and not currently used.
// They are kept here for reference but should be removed once the new
// post-processor implementation in generate.go is fully tested.
//
// TODO(https://github.com/julieqiu/librarianx-rust/issues/XXX): Remove these functions
//
// func runSynthtool(ctx context.Context, cfg *postProcessorConfig) error { ... }
// func runCustomOwlbot(ctx context.Context, outputDir, owlbotPath string) error { ... }
// func runDefaultSynthtool(ctx context.Context, outputDir, libraryPath string) error { ... }
// func runFormatters(ctx context.Context, outputDir, libraryPath string) error { ... }

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
