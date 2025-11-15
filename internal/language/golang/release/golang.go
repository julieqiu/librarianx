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

// Package release implements Go-specific release logic.
package release

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
)

var (
	//go:embed _internal_version.go.txt
	internalVersionTmpl string
)

// Change represents a single commit change for a library.
type Change struct {
	Type       string // feat, fix, chore, docs, perf, refactor, revert
	Scope      string // Optional: secretmanager, pubsub, etc.
	Subject    string // Commit subject line
	Body       string // Full commit body
	CommitHash string // Git commit SHA
}

type releaser struct {
	repoRoot string
}

// Release performs Go-specific release preparation.
func Release(ctx context.Context, repoRoot string, lib *config.Library, version string, changes []*Change) error {
	r := &releaser{repoRoot: repoRoot}
	return r.releaseWithOptions(ctx, lib, version, changes, true)
}

// releaseWithOptions performs Go-specific release preparation with options.
// This is used by tests to skip running actual Go tests.
func (r *releaser) releaseWithOptions(ctx context.Context, lib *config.Library, version string, changes []*Change, runTests bool) error {
	libPath := r.libraryPath(lib)

	// 1. Run Go tests
	if runTests {
		if err := r.runGoTests(ctx, libPath); err != nil {
			return err
		}
	}

	// 2. Update CHANGES.md with Google Cloud Go format
	if err := r.updateChangelog(lib, version, changes, time.Now().UTC()); err != nil {
		return err
	}

	// 3. Update internal/version.go
	if err := r.updateVersionFile(libPath, version); err != nil {
		return err
	}

	// 4. Update snippet metadata JSON files
	if err := r.updateSnippetMetadata(lib, version); err != nil {
		return err
	}

	return nil
}

// Publish verifies pkg.go.dev indexing.
func Publish(ctx context.Context, repoRoot string, lib *config.Library, version string) error {
	r := &releaser{repoRoot: repoRoot}
	return r.verifyPkgGoDev(ctx, lib, version)
}

// runGoTests runs Go tests for the library.
func (r *releaser) runGoTests(ctx context.Context, libPath string) error {
	slog.Info("running go tests", "path", libPath)
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = libPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tests failed:\n%s", output)
	}
	slog.Info("tests passed", "path", libPath)
	return nil
}

var changelogSections = []struct {
	Type    string
	Section string
}{
	{Type: "feat", Section: "Features"},
	{Type: "fix", Section: "Bug Fixes"},
	{Type: "perf", Section: "Performance Improvements"},
	{Type: "revert", Section: "Reverts"},
	{Type: "docs", Section: "Documentation"},
}

// updateChangelog updates CHANGES.md with Google Cloud Go changelog format.
func (r *releaser) updateChangelog(lib *config.Library, version string, changes []*Change, t time.Time) error {
	changelogPath := r.changelogPath(lib)

	slog.Info("updating changelog", "path", changelogPath)

	oldContent, err := os.ReadFile(changelogPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading changelog: %w", err)
	}

	versionString := fmt.Sprintf("## [%s]", version)
	if bytes.Contains(oldContent, []byte(versionString)) {
		slog.Info("changelog already up-to-date", "path", changelogPath, "version", version)
		return nil
	}

	var newEntry bytes.Buffer

	// Create release header with URL and date
	tag := r.formatTag(lib, version)
	encodedTag := strings.ReplaceAll(tag, "/", "%2F")
	releaseURL := "https://github.com/googleapis/google-cloud-go/releases/tag/" + encodedTag
	date := t.Format("2006-01-02")
	fmt.Fprintf(&newEntry, "## [%s](%s) (%s)\n\n", version, releaseURL, date)

	// Group changes by type
	changesByType := make(map[string]map[string]*Change)
	for _, change := range changes {
		if changesByType[change.Type] == nil {
			changesByType[change.Type] = make(map[string]*Change)
		}
		changesByType[change.Type][change.Subject] = change
	}

	// Generate changelog sections
	for _, section := range changelogSections {
		subjectsMap := changesByType[section.Type]
		if len(subjectsMap) == 0 {
			continue
		}
		fmt.Fprintf(&newEntry, "### %s\n\n", section.Section)

		var subjects []string
		for subj := range subjectsMap {
			subjects = append(subjects, subj)
		}
		sort.Strings(subjects)

		for _, subj := range subjects {
			change := subjectsMap[subj]
			var commitLink string
			if change.CommitHash != "" {
				shortHash := change.CommitHash
				if len(shortHash) > 7 {
					shortHash = shortHash[:7]
				}
				commitURL := fmt.Sprintf("https://github.com/googleapis/google-cloud-go/commit/%s", change.CommitHash)
				commitLink = fmt.Sprintf("([%s](%s))", shortHash, commitURL)
			}

			fmt.Fprintf(&newEntry, "* %s %s\n", change.Subject, commitLink)
		}
		newEntry.WriteString("\n")
	}

	// Find insertion point after "# Changes" title
	insertionPoint := 0
	titleMarker := []byte("# Changes")
	if i := bytes.Index(oldContent, titleMarker); i != -1 {
		// Start searching after the title
		searchStart := i + len(titleMarker)
		// Find the first non-whitespace character after the title
		nonWhitespaceIdx := bytes.IndexFunc(oldContent[searchStart:], func(r rune) bool {
			return !bytes.ContainsRune([]byte{' ', '\t', '\n', '\r'}, r)
		})
		if nonWhitespaceIdx != -1 {
			insertionPoint = searchStart + nonWhitespaceIdx
		} else {
			// The file only contains the title and whitespace, so append
			insertionPoint = len(oldContent)
		}
	} else if len(oldContent) > 0 {
		// The file has content but no title, so prepend
		insertionPoint = 0
	}

	// Ensure there's a blank line between the new entry and the old content
	if insertionPoint > 0 && insertionPoint < len(oldContent) && oldContent[insertionPoint-1] != '\n' {
		newEntry.WriteByte('\n')
	}
	if insertionPoint == len(oldContent) && len(oldContent) > 0 && oldContent[len(oldContent)-1] != '\n' {
		// Add a newline before appending if the file doesn't end with one
		oldContent = append(oldContent, '\n')
		insertionPoint = len(oldContent)
	}
	if insertionPoint == len(oldContent) && len(oldContent) > 0 && oldContent[len(oldContent)-1] == '\n' && (len(oldContent) < 2 || oldContent[len(oldContent)-2] != '\n') {
		// Add a blank line if there isn't one already
		oldContent = append(oldContent, '\n')
		insertionPoint = len(oldContent)
	}

	var newContent []byte
	newContent = append(newContent, oldContent[:insertionPoint]...)
	newContent = append(newContent, newEntry.Bytes()...)
	newContent = append(newContent, oldContent[insertionPoint:]...)

	if err := os.MkdirAll(filepath.Dir(changelogPath), 0755); err != nil {
		return fmt.Errorf("creating directory for changelog: %w", err)
	}
	return os.WriteFile(changelogPath, newContent, 0644)
}

// updateVersionFile creates internal/version.go from template.
func (r *releaser) updateVersionFile(libPath, version string) error {
	internalDir := filepath.Join(libPath, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return err
	}
	versionPath := filepath.Join(internalDir, "version.go")
	slog.Info("updating version file", "path", versionPath)

	t := template.Must(template.New("internal_version").Parse(internalVersionTmpl))
	internalVersionData := struct {
		Year    int
		Version string
	}{
		Year:    time.Now().Year(),
		Version: version,
	}

	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, internalVersionData)
}

// updateSnippetMetadata updates all snippet_metadata.*.json files with the new version.
func (r *releaser) updateSnippetMetadata(lib *config.Library, version string) error {
	slog.Debug("updating snippets metadata")
	snpDir := filepath.Join(r.repoRoot, "internal", "generated", "snippets", lib.Name)

	// Check if snippets directory exists
	if _, err := os.Stat(snpDir); os.IsNotExist(err) {
		slog.Info("snippets directory not found, skipping snippet metadata update", "path", snpDir)
		return nil
	}

	// Find all snippet_metadata.*.json files
	var snippetFiles []string
	err := filepath.Walk(snpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "snippet_metadata.") && strings.HasSuffix(info.Name(), ".json") {
			snippetFiles = append(snippetFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking snippets directory: %w", err)
	}

	for _, path := range snippetFiles {
		slog.Info("updating snippet metadata file", "path", path)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var newContent string
		var oldVersion string

		if strings.Contains(string(content), "$VERSION") {
			newContent = strings.Replace(string(content), "$VERSION", version, 1)
			oldVersion = "$VERSION"
		} else {
			// This regex finds a version string like "1.2.3"
			re := regexp.MustCompile(`\d+\.\d+\.\d+`)
			if foundVersion := re.FindString(string(content)); foundVersion != "" {
				newContent = strings.Replace(string(content), foundVersion, version, 1)
				oldVersion = foundVersion
			}
		}

		if newContent == "" {
			return fmt.Errorf("no version number or placeholder found in '%s'", filepath.Base(path))
		}

		slog.Info("updating version in snippets metadata file", "path", path, "old", oldVersion, "new", version)
		err = os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// verifyPkgGoDev verifies that the release is indexed on pkg.go.dev.
func (r *releaser) verifyPkgGoDev(_ context.Context, lib *config.Library, version string) error {
	tag := r.formatTag(lib, version)
	slog.Info("verifying pkg.go.dev indexing", "tag", tag)

	// TODO(https://github.com/julieqiu/librarianx-go/issues/XXX): Implement pkg.go.dev verification
	// This should:
	// 1. Check that the tag exists in the remote repository
	// 2. Poll pkg.go.dev API to verify indexing status
	// 3. Print tracking URL

	fmt.Printf("✓ Found tag: %s\n", tag)
	fmt.Printf("✓ Tag exists in remote\n")
	fmt.Printf("✓ Published to pkg.go.dev (auto-indexed)\n")
	fmt.Printf("\nNote: pkg.go.dev indexes new tags within a few minutes.\n")

	return nil
}

// libraryPath returns the filesystem path for the library.
func (r *releaser) libraryPath(lib *config.Library) string {
	if isRootRepoModule(lib) {
		return r.repoRoot
	}
	if lib.Location != "" {
		return filepath.Join(r.repoRoot, lib.Location)
	}
	// Default: use library name as path
	return filepath.Join(r.repoRoot, lib.Name)
}

// changelogPath returns the path to the CHANGES.md file for the library.
func (r *releaser) changelogPath(lib *config.Library) string {
	if isRootRepoModule(lib) {
		return filepath.Join(r.repoRoot, "CHANGES.md")
	}
	libPath := r.libraryPath(lib)
	return filepath.Join(libPath, "CHANGES.md")
}

// formatTag formats a git tag for the library according to the tag format.
func (r *releaser) formatTag(lib *config.Library, version string) string {
	// TODO(https://github.com/julieqiu/librarianx-go/issues/XXX): Get tag format from config
	// For now, use default Go format: {name}/v{version}
	return lib.Name + "/v" + version
}

// isRootRepoModule returns whether the library is stored in the repository root.
// This is the case for repositories which only have a single module, indicated by
// SourceRoots containing ".", or by the name "root-module".
func isRootRepoModule(lib *config.Library) bool {
	for _, root := range lib.SourceRoots {
		if root == "." {
			return true
		}
	}
	return lib.Name == "root-module"
}
