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
	"strings"
)

// defaultKeepPaths returns the default list of files/directories to preserve during regeneration.
// The library name will be substituted for {name} in the paths.
func defaultKeepPaths(libraryName string) []string {
	paths := []string{
		"packages/{name}/CHANGELOG.md",
		"docs/CHANGELOG.md",
		"docs/README.rst",
		"docs/index.rst",
		"samples/README.txt",
		"scripts/client-post-processing/",
		"samples/snippets/README.rst",
		"tests/system/",
		"tests/unit/gapic/type/test_type.py",
	}

	result := make([]string, len(paths))
	for i, p := range paths {
		result[i] = strings.ReplaceAll(p, "{name}", libraryName)
	}
	return result
}

// cleanOutputDirectory deletes everything in the output directory except files listed in keepPaths.
// If keepPaths is empty, uses defaultKeepPaths for the library.
func cleanOutputDirectory(outdir string, keepPaths []string, libraryName string) error {
	// Check if directory exists
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		return nil
	}

	// Use default keep paths if none specified
	if len(keepPaths) == 0 {
		keepPaths = defaultKeepPaths(libraryName)
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
