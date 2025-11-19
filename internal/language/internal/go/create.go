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
	"path/filepath"
	"strings"

	"github.com/julieqiu/librarianx/internal/config"
)

// Create creates a new Go client library.
// It creates initial scaffolding files and calls Generate to create the library code.
func Create(ctx context.Context, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
	// Determine output directory
	outdir := library.Path
	if outdir == "" {
		// Use default output pattern if no explicit path
		if defaults != nil {
			outdir = strings.ReplaceAll(defaults.Output, "{name}", library.Name)
		}
	}

	// Convert to absolute path
	var err error
	outdir, err = filepath.Abs(outdir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create initial scaffolding files
	if err := createInitialScaffoldingFiles(outdir, library); err != nil {
		return fmt.Errorf("failed to create initial scaffolding files: %w", err)
	}

	// Call Generate to create the library code
	if err := Generate(ctx, library, defaults, googleapisDir, serviceConfigPath, defaultOutput); err != nil {
		return fmt.Errorf("failed to generate library: %w", err)
	}

	return nil
}

// createInitialScaffoldingFiles creates the initial files for a new Go library.
func createInitialScaffoldingFiles(outdir string, library *config.Library) error {
	// Create README.md
	readmePath := filepath.Join(outdir, "README.md")
	readme := fmt.Sprintf(`# %s

## Installation

`+"```"+`bash
go get cloud.google.com/go/%s
`+"```"+`
`, library.Name, library.Name)
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Create CHANGES.md
	changesPath := filepath.Join(outdir, "CHANGES.md")
	changes := "# Changes\n"
	if err := os.WriteFile(changesPath, []byte(changes), 0644); err != nil {
		return fmt.Errorf("failed to write CHANGES.md: %w", err)
	}

	// Create internal/version.go
	internalDir := filepath.Join(outdir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return fmt.Errorf("failed to create internal directory: %w", err)
	}

	versionPath := filepath.Join(internalDir, "version.go")
	version := `package internal

// Version is the current version of this client library.
const Version = "0.0.0"
`
	if err := os.WriteFile(versionPath, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to write internal/version.go: %w", err)
	}

	// Create go.mod
	goModPath := filepath.Join(outdir, "go.mod")
	goMod := fmt.Sprintf(`module cloud.google.com/go/%s

go 1.23
`, library.Name)
	if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	return nil
}
