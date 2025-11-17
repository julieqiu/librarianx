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
	"regexp"

	"github.com/googleapis/librarian/internal/config"
)

// Create creates a new Python client library.
// It creates changelog files for the new library.
func Create(ctx context.Context, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
	// Get current working directory as repoDir
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// 1. Update global CHANGELOG.md
	globalChangelogSrc := filepath.Join(repoDir, "CHANGELOG.md")
	globalChangelogDest := filepath.Join(defaultOutput, "CHANGELOG.md")

	// Create library config for changelog update
	newLibraryConfig := map[string]interface{}{
		"id":      library.Name,
		"version": library.Version,
	}

	// The Python function calls _update_global_changelog with a list containing only the new library config.
	allLibraries := []map[string]interface{}{newLibraryConfig}
	if err := updateGlobalChangelog(globalChangelogSrc, globalChangelogDest, allLibraries); err != nil {
		return fmt.Errorf("failed to update global changelog: %w", err)
	}

	// 2. Create a `CHANGELOG.md` for the new library
	// 3. Create a `docs/CHANGELOG.md` file for the new library
	if err := createNewChangelogForLibrary(library.Name, defaultOutput); err != nil {
		return fmt.Errorf("failed to create new library changelogs: %w", err)
	}

	return nil
}

// readTextFile is a helper function that reads a text file and returns its content.
func readTextFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// writeTextFile is a helper function that writes content to a text file,
// creating necessary directories if they don't exist.
func writeTextFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// updateGlobalChangelog updates the versions of libraries in the main CHANGELOG.md.
func updateGlobalChangelog(changelogSrc, changelogDest string, allLibraries []map[string]interface{}) error {
	content, err := readTextFile(changelogSrc)
	if err != nil {
		// If the source changelog doesn't exist, initialize with a basic header.
		if os.IsNotExist(err) {
			content = "# Changelog\n\n"
		} else {
			return fmt.Errorf("failed to read global changelog source: %w", err)
		}
	}

	newContent := content
	for _, library := range allLibraries {
		libraryID, ok := library["id"].(string)
		if !ok {
			return fmt.Errorf("library 'id' not found or not a string in allLibraries entry: %v", library)
		}
		version, ok := library["version"].(string)
		if !ok {
			return fmt.Errorf("library 'version' not found or not a string in allLibraries entry: %v", library)
		}

		// Regex pattern to find and replace the version: `[library_id]==version]`
		// Example: `[google-cloud-language]==1.2.3]`
		patternStr := fmt.Sprintf(`(\[%s)(==)([\d\.]+)(\])`, regexp.QuoteMeta(libraryID))
		re := regexp.MustCompile(patternStr)
		replacement := fmt.Sprintf("$1==%s$4", version)
		newContent = re.ReplaceAllString(newContent, replacement)
	}

	return writeTextFile(changelogDest, newContent)
}

// createNewChangelogForLibrary creates a new CHANGELOG.md and docs/CHANGELOG.md
// for a given library.
func createNewChangelogForLibrary(libraryID, outputDir string) error {
	packageChangelogPath := filepath.Join(outputDir, "packages", libraryID, "CHANGELOG.md")
	docsChangelogPath := filepath.Join(outputDir, "packages", libraryID, "docs", "CHANGELOG.md")

	changelogContent := fmt.Sprintf("# Changelog\n\n[PyPI History][1]\n\n[1]: https://pypi.org/project/%s/#history\n", libraryID)

	if err := writeTextFile(packageChangelogPath, changelogContent); err != nil {
		return fmt.Errorf("failed to write package changelog: %w", err)
	}

	if err := writeTextFile(docsChangelogPath, changelogContent); err != nil {
		return fmt.Errorf("failed to write docs changelog: %w", err)
	}

	return nil
}
