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

package rust

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cleanOutputDirectory deletes everything in the output directory except files listed in keepPaths.
// For Rust, if keepPaths is empty, all Cargo.toml files are automatically preserved.
func cleanOutputDirectory(outdir string, keepPaths []string) error {
	// Check if directory exists
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		return nil
	}

	// For Rust, find all Cargo.toml files recursively and add them to keep paths
	cargoFiles, err := findCargoTomlFiles(outdir)
	if err != nil {
		return fmt.Errorf("failed to find Cargo.toml files: %w", err)
	}
	// Convert absolute paths to relative paths
	for _, cargoFile := range cargoFiles {
		relPath, err := filepath.Rel(outdir, cargoFile)
		if err == nil {
			keepPaths = append(keepPaths, relPath)
		}
	}

	// Build map of paths to keep (normalized to absolute paths)
	keepMap := make(map[string]bool)
	for _, keepPath := range keepPaths {
		absKeepPath := filepath.Join(outdir, keepPath)
		keepMap[absKeepPath] = true
	}

	// Walk directory and delete everything not in keep list
	entries, err := os.ReadDir(outdir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", outdir, err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(outdir, entry.Name())

		// Check if this path or any parent should be kept
		if shouldKeep(entryPath, keepMap) {
			continue
		}

		// Delete this entry
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", entryPath, err)
		}
	}

	return nil
}

// findCargoTomlFiles recursively finds all Cargo.toml files in a directory.
func findCargoTomlFiles(root string) ([]string, error) {
	var cargoFiles []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && d.Name() == "Cargo.toml" {
			cargoFiles = append(cargoFiles, path)
		}

		return nil
	})

	return cargoFiles, err
}

// shouldKeep checks if a path should be kept based on the keep map.
// A path is kept if it exactly matches a keep path, or if it's a parent of a keep path.
func shouldKeep(path string, keepMap map[string]bool) bool {
	// Exact match
	if keepMap[path] {
		return true
	}

	// Check if path is a parent of any keep path
	for keepPath := range keepMap {
		if strings.HasPrefix(keepPath, path+string(filepath.Separator)) {
			return true
		}
	}

	return false
}
