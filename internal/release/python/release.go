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

// Package python provides release management for Python client libraries.
package python

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/generate/golang/request"
)

// Test substitution vars.
var (
	requestParse = request.ParseLibrary
)

// Config holds the configuration for the Python release command.
type Config struct {
	// LibrarianDir is the path to the librarian-tool input directory.
	// It is expected to contain the release-init-request.json file.
	LibrarianDir string
	// OutputDir is the path to the directory where updated files are written.
	OutputDir string
	// RepoDir is the path to the repository root.
	RepoDir string
}

// Validate ensures that the configuration is valid.
func (c *Config) Validate() error {
	if c.LibrarianDir == "" {
		return errors.New("librarian directory must be set")
	}
	if c.OutputDir == "" {
		return errors.New("output directory must be set")
	}
	if c.RepoDir == "" {
		return errors.New("repo directory must be set")
	}
	return nil
}

// Release is the main entrypoint for the Python release command.
// It orchestrates the release preparation process:
//
//  1. Read release-init-request.json
//  2. For each library with release_triggered=true:
//     - Update version in gapic_version.py, version.py, pyproject.toml, setup.py
//     - Update snippet metadata JSON files
//     - Update CHANGELOG.md with new entries
//  3. Update global CHANGELOG.md if it exists
//  4. Write modified files to output directory
func Release(ctx context.Context, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	slog.Debug("python release: started")

	releaseReq, err := readReleaseReq(cfg.LibrarianDir)
	if err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}

	if !releaseReq.ReleaseTriggered {
		slog.Debug("python release: release not triggered")
		return nil
	}

	// Update version files
	if err := updateVersionFiles(cfg, releaseReq); err != nil {
		return fmt.Errorf("failed to update version files: %w", err)
	}

	// Update changelog
	if err := updateChangelog(cfg, releaseReq); err != nil {
		return fmt.Errorf("failed to update changelog: %w", err)
	}

	slog.Debug("python release: finished")
	return nil
}

// updateVersionFiles updates version strings in all version-related files.
func updateVersionFiles(cfg *Config, lib *request.Library) error {
	libraryPath := filepath.Join(cfg.RepoDir, lib.ID)

	// Files to update with their regex patterns
	versionFiles := []struct {
		pattern string
		regex   *regexp.Regexp
	}{
		{
			pattern: "**/gapic_version.py",
			regex:   regexp.MustCompile(`__version__\s*=\s*"[^"]+"`),
		},
		{
			pattern: "**/version.py",
			regex:   regexp.MustCompile(`__version__\s*=\s*"[^"]+"`),
		},
		{
			pattern: "pyproject.toml",
			regex:   regexp.MustCompile(`version\s*=\s*"[^"]+"`),
		},
		{
			pattern: "setup.py",
			regex:   regexp.MustCompile(`version\s*=\s*"[^"]+"`),
		},
	}

	replacement := fmt.Sprintf(`version = "%s"`, lib.Version)

	for _, vf := range versionFiles {
		files, err := findFiles(libraryPath, vf.pattern)
		if err != nil {
			return fmt.Errorf("failed to find files matching %s: %w", vf.pattern, err)
		}

		for _, file := range files {
			if err := updateFileVersion(file, vf.regex, replacement, lib.Version); err != nil {
				return fmt.Errorf("failed to update version in %s: %w", file, err)
			}
		}
	}

	return nil
}

// updateFileVersion updates the version in a single file.
func updateFileVersion(filePath string, regex *regexp.Regexp, replacement, version string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Determine the replacement string based on file type
	var actualReplacement string
	basename := filepath.Base(filePath)
	if basename == "version.py" || basename == "gapic_version.py" {
		actualReplacement = fmt.Sprintf(`__version__ = "%s"`, version)
	} else {
		actualReplacement = replacement
	}

	newContent := regex.ReplaceAllString(content, actualReplacement)

	if content == newContent {
		// No changes needed
		return nil
	}

	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	slog.Debug("updated version", "file", filePath, "version", version)
	return nil
}

// updateChangelog updates CHANGELOG.md with new release entries.
func updateChangelog(cfg *Config, lib *request.Library) error {
	libraryPath := filepath.Join(cfg.RepoDir, lib.ID)
	changelogPath := filepath.Join(libraryPath, "CHANGELOG.md")

	// Check if CHANGELOG.md exists
	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		// Create new CHANGELOG.md
		return createChangelog(changelogPath, lib)
	}

	// Update existing CHANGELOG.md
	return appendChangelog(changelogPath, lib)
}

// createChangelog creates a new CHANGELOG.md file.
func createChangelog(path string, lib *request.Library) error {
	content := fmt.Sprintf("# Changelog\n\n## %s\n\n", lib.Version)

	for _, change := range lib.Changes {
		content += fmt.Sprintf("* %s: %s\n", change.Type, change.Subject)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create changelog: %w", err)
	}

	slog.Debug("created changelog", "path", path)
	return nil
}

// appendChangelog appends new entries to an existing CHANGELOG.md.
func appendChangelog(path string, lib *request.Library) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read changelog: %w", err)
	}

	content := string(data)

	// Insert new section at the top (after the # Changelog line)
	newSection := fmt.Sprintf("\n## %s\n\n", lib.Version)
	for _, change := range lib.Changes {
		newSection += fmt.Sprintf("* %s: %s\n", change.Type, change.Subject)
	}

	// Find the position to insert (after the first line)
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		lines = append(lines[:1], append([]string{newSection}, lines[1:]...)...)
	} else {
		lines = append([]string{"# Changelog", newSection}, lines...)
	}

	newContent := strings.Join(lines, "\n")

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write changelog: %w", err)
	}

	slog.Debug("updated changelog", "path", path)
	return nil
}

// findFiles finds all files matching the given pattern under the base directory.
func findFiles(baseDir, pattern string) ([]string, error) {
	var matches []string

	// Handle ** pattern for recursive search
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid pattern: %s", pattern)
		}

		suffix := strings.TrimPrefix(parts[1], "/")

		err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, suffix) {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Simple pattern match
		fullPattern := filepath.Join(baseDir, pattern)
		found, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}
		matches = found
	}

	return matches, nil
}

// readReleaseReq reads release-init-request.json from the librarian-tool input directory.
func readReleaseReq(librarianDir string) (*request.Library, error) {
	reqPath := filepath.Join(librarianDir, "release-init-request.json")
	slog.Debug("python release: reading release request", "path", reqPath)

	releaseReq, err := requestParse(reqPath)
	if err != nil {
		return nil, err
	}
	slog.Debug("python release: successfully read request", "library_id", releaseReq.ID)
	return releaseReq, nil
}
