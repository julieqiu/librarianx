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

package librarian

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/julieqiu/librarianx/internal/config"
	"github.com/urfave/cli/v3"
)

// fmtCommand formats the librarian.yaml file.
func fmtCommand() *cli.Command {
	return &cli.Command{
		Name:      "fmt",
		Usage:     "format librarian.yaml",
		UsageText: "librarian fmt",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := FormatLibrarianYAML("."); err != nil {
				return fmt.Errorf("format failed: %w", err)
			}
			fmt.Println("Formatted librarian.yaml")
			return nil
		},
	}
}

// FormatLibrarianYAML formats the librarian.yaml file in the given directory.
// It reads the file using config.Read and writes it back using config.Write,
// which applies formatting. Returns nil if the file doesn't exist or if
// formatting succeeds.
func FormatLibrarianYAML(dir string) error {
	path := filepath.Join(dir, "librarian.yaml")

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Not an error if file doesn't exist
	}

	// Read the config
	cfg, err := config.Read(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	// Write it back (this applies formatting)
	if err := cfg.Write(path); err != nil {
		return fmt.Errorf("failed to format %s: %w", path, err)
	}
	return nil
}
