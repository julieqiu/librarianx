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

// Package python provides Python-specific release functionality for client libraries.
package python

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
)

// Change represents a single commit change for a library.
type Change struct {
	Type       string
	Scope      string
	Subject    string
	Body       string
	CommitHash string
}

// runPythonTests runs nox unit tests for the library.
func runPythonTests(ctx context.Context, libPath string) error {
	cmd := exec.CommandContext(ctx, "nox", "-s", "unit")
	cmd.Dir = libPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nox unit tests failed: %w", err)
	}

	return nil
}

// normalizePythonVersion normalizes semver pre-release to PEP 440 format.
// Examples:
//   - 1.16.0-rc.1 -> 1.16.0rc1
//   - 1.16.0-alpha.1 -> 1.16.0a1
//   - 1.16.0-beta.1 -> 1.16.0b1
func normalizePythonVersion(version string) string {
	version = strings.ReplaceAll(version, "-rc.", "rc")
	version = strings.ReplaceAll(version, "-alpha.", "a")
	version = strings.ReplaceAll(version, "-beta.", "b")
	return version
}

// updateVersionFiles updates version strings in all version-related files.
func updateVersionFiles(libPath, version string) error {
	files := []struct {
		path    string
		updater func(string, string) error
	}{
		{
			path:    filepath.Join(libPath, "pyproject.toml"),
			updater: updatePyprojectToml,
		},
		{
			path:    filepath.Join(libPath, "setup.py"),
			updater: updateSetupPy,
		},
		{
			path:    findGapicVersionFile(libPath),
			updater: updateGapicVersion,
		},
		{
			path:    findVersionFile(libPath),
			updater: updateVersionPy,
		},
	}

	for _, f := range files {
		if f.path == "" {
			continue // File doesn't exist
		}

		if err := f.updater(f.path, version); err != nil {
			return fmt.Errorf("failed to update %s: %w", f.path, err)
		}
	}

	return nil
}

// updatePyprojectToml updates version in pyproject.toml.
func updatePyprojectToml(path, version string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Replace version in [project] section
	// version = "1.15.0" -> version = "1.16.0"
	re := regexp.MustCompile(`(?m)^version\s*=\s*"[^"]*"`)
	updated := re.ReplaceAllString(string(content), fmt.Sprintf(`version = "%s"`, version))

	return os.WriteFile(path, []byte(updated), 0644)
}

// updateSetupPy updates version in setup.py (legacy).
func updateSetupPy(path, version string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // setup.py is optional (legacy)
		}
		return err
	}

	// Replace version = "1.15.0" -> version = "1.16.0"
	re := regexp.MustCompile(`version\s*=\s*"[^"]*"`)
	updated := re.ReplaceAllString(string(content), fmt.Sprintf(`version = "%s"`, version))

	return os.WriteFile(path, []byte(updated), 0644)
}

// updateGapicVersion updates version in gapic_version.py.
func updateGapicVersion(path, version string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Replace __version__ = "1.15.0" -> __version__ = "1.16.0"
	re := regexp.MustCompile(`__version__\s*=\s*"[^"]*"`)
	updated := re.ReplaceAllString(string(content), fmt.Sprintf(`__version__ = "%s"`, version))

	return os.WriteFile(path, []byte(updated), 0644)
}

// updateVersionPy updates version in version.py.
func updateVersionPy(path, version string) error {
	return updateGapicVersion(path, version) // Same format as gapic_version.py
}

// findGapicVersionFile searches for google/cloud/*/gapic_version.py.
func findGapicVersionFile(libPath string) string {
	pattern := filepath.Join(libPath, "google", "cloud", "*", "gapic_version.py")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// findVersionFile searches for google/cloud/*/version.py.
func findVersionFile(libPath string) string {
	pattern := filepath.Join(libPath, "google", "cloud", "*", "version.py")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// updateChangelogs updates package, docs, and global changelogs.
func updateChangelogs(repoDir string, lib *config.Library, version string, changes []*Change) error {
	// Ensure lib.Location is absolute by joining with repoDir if needed
	libPath := lib.Location
	if !filepath.IsAbs(libPath) {
		libPath = filepath.Join(repoDir, libPath)
	}

	// 1. Update package CHANGELOG.md
	pkgChangelog := filepath.Join(libPath, "CHANGELOG.md")
	if err := updateChangelog(pkgChangelog, lib.Name, version, changes); err != nil {
		return fmt.Errorf("failed to update package changelog: %w", err)
	}

	// 2. Update docs/CHANGELOG.md (duplicate)
	docsChangelog := filepath.Join(libPath, "docs", "CHANGELOG.md")
	if err := updateChangelog(docsChangelog, lib.Name, version, changes); err != nil {
		return fmt.Errorf("failed to update docs changelog: %w", err)
	}

	// 3. Update global CHANGELOG.md (if exists)
	globalChangelog := filepath.Join(repoDir, "CHANGELOG.md")
	if fileExists(globalChangelog) {
		if err := updateGlobalChangelog(globalChangelog, lib.Name, version, changes); err != nil {
			return fmt.Errorf("failed to update global changelog: %w", err)
		}
	}

	return nil
}

// updateChangelog updates a package-level CHANGELOG.md file.
func updateChangelog(path, libName, version string, changes []*Change) error {
	// Generate new changelog entry
	entry := formatChangelogEntry(libName, version, changes)

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Read existing changelog
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new changelog
			newContent := "# Changelog\n\n" + entry
			return os.WriteFile(path, []byte(newContent), 0644)
		}
		return err
	}

	// Insert new entry at top (after "# Changelog" header)
	lines := strings.Split(string(content), "\n")

	// Find insertion point (after first header)
	insertIdx := 1
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			insertIdx = i + 1
			break
		}
	}

	// Insert new entry
	newLines := append(lines[:insertIdx], append([]string{"", entry, ""}, lines[insertIdx:]...)...)
	updated := strings.Join(newLines, "\n")

	return os.WriteFile(path, []byte(updated), 0644)
}

// formatChangelogEntry formats a changelog entry with GitHub links.
func formatChangelogEntry(libName, version string, changes []*Change) string {
	today := time.Now().Format("2006-01-02")

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("## [%s](https://github.com/googleapis/google-cloud-python/releases/tag/%s-v%s) (%s)\n\n",
		version, libName, version, today))

	// Group changes by type
	features := filterByType(changes, "feat")
	fixes := filterByType(changes, "fix")
	docs := filterByType(changes, "docs")

	if len(features) > 0 {
		buf.WriteString("### Features\n\n")
		for _, c := range features {
			buf.WriteString(fmt.Sprintf("* %s ([%s](https://github.com/googleapis/google-cloud-python/commit/%s))\n",
				c.Subject, c.CommitHash[:7], c.CommitHash))
		}
		buf.WriteString("\n")
	}

	if len(fixes) > 0 {
		buf.WriteString("### Bug Fixes\n\n")
		for _, c := range fixes {
			buf.WriteString(fmt.Sprintf("* %s ([%s](https://github.com/googleapis/google-cloud-python/commit/%s))\n",
				c.Subject, c.CommitHash[:7], c.CommitHash))
		}
		buf.WriteString("\n")
	}

	if len(docs) > 0 {
		buf.WriteString("### Documentation\n\n")
		for _, c := range docs {
			buf.WriteString(fmt.Sprintf("* %s ([%s](https://github.com/googleapis/google-cloud-python/commit/%s))\n",
				c.Subject, c.CommitHash[:7], c.CommitHash))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// updateGlobalChangelog updates the global CHANGELOG.md file.
func updateGlobalChangelog(path, libName, version string, changes []*Change) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	entry := formatGlobalChangelogEntry(libName, version, changes)

	// Find section for this library, or create it
	lines := strings.Split(string(content), "\n")
	insertIdx := findLibrarySection(lines, libName)

	// Insert entry
	newLines := append(lines[:insertIdx], append([]string{entry}, lines[insertIdx:]...)...)
	updated := strings.Join(newLines, "\n")

	return os.WriteFile(path, []byte(updated), 0644)
}

// formatGlobalChangelogEntry formats a global changelog entry.
func formatGlobalChangelogEntry(libName, version string, changes []*Change) string {
	today := time.Now().Format("2006-01-02")

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("### [%s] (%s)\n\n", version, today))

	// Group changes by type
	features := filterByType(changes, "feat")
	fixes := filterByType(changes, "fix")
	docs := filterByType(changes, "docs")

	if len(features) > 0 {
		buf.WriteString("#### Features\n\n")
		for _, c := range features {
			buf.WriteString(fmt.Sprintf("* %s\n", c.Subject))
		}
		buf.WriteString("\n")
	}

	if len(fixes) > 0 {
		buf.WriteString("#### Bug Fixes\n\n")
		for _, c := range fixes {
			buf.WriteString(fmt.Sprintf("* %s\n", c.Subject))
		}
		buf.WriteString("\n")
	}

	if len(docs) > 0 {
		buf.WriteString("#### Documentation\n\n")
		for _, c := range docs {
			buf.WriteString(fmt.Sprintf("* %s\n", c.Subject))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// findLibrarySection finds the insertion point for a library's section in the global changelog.
func findLibrarySection(lines []string, libName string) int {
	// Look for ## <libName> section
	sectionHeader := "## " + libName

	for i, line := range lines {
		if strings.TrimSpace(line) == sectionHeader {
			// Found the section, insert after the header
			return i + 1
		}
	}

	// Section not found, insert after # Changelog header
	for i, line := range lines {
		if strings.HasPrefix(line, "# Changelog") {
			// Insert new section after Changelog header
			newSection := []string{"", sectionHeader, ""}
			return i + len(newSection)
		}
	}

	// No header found, insert at beginning
	return 0
}

// filterByType filters changes by commit type.
func filterByType(changes []*Change, changeType string) []*Change {
	var filtered []*Change
	for _, c := range changes {
		if c.Type == changeType {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// updateSnippetMetadata updates snippet metadata JSON files.
func updateSnippetMetadata(repoDir string, lib *config.Library, version string) error {
	// Find snippet metadata files
	pattern := filepath.Join(repoDir, "internal", "generated", "snippets", lib.Name, "**", "snippet_metadata.*.json")
	matches, err := findFiles(pattern)
	if err != nil {
		return err
	}

	for _, path := range matches {
		if err := updateSnippetMetadataFile(path, version); err != nil {
			return fmt.Errorf("failed to update %s: %w", path, err)
		}
	}

	return nil
}

// updateSnippetMetadataFile updates a single snippet metadata file.
func updateSnippetMetadataFile(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	// Update version field
	metadata["clientVersion"] = version

	updated, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, updated, 0644)
}

// findFiles finds all files matching the given pattern (supports ** for recursive).
func findFiles(pattern string) ([]string, error) {
	var matches []string

	// Handle ** pattern for recursive search
	if strings.Contains(pattern, "**") {
		parts := strings.SplitN(pattern, "**", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid pattern: %s", pattern)
		}

		baseDir := parts[0]
		suffix := strings.TrimPrefix(parts[1], "/")

		err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// Ignore errors for non-existent directories
				return nil
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
		found, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		matches = found
	}

	return matches, nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
