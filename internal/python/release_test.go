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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNormalizePythonVersion(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "stable", input: "1.16.0", want: "1.16.0"},
		{name: "rc", input: "1.16.0-rc.1", want: "1.16.0rc1"},
		{name: "alpha", input: "1.16.0-alpha.1", want: "1.16.0a1"},
		{name: "beta", input: "1.16.0-beta.2", want: "1.16.0b2"},
		{name: "multiple-rc", input: "2.0.0-rc.10", want: "2.0.0rc10"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := normalizePythonVersion(test.input)
			if got != test.want {
				t.Errorf("normalizePythonVersion(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestUpdatePyprojectToml(t *testing.T) {
	content := `[project]
name = "google-cloud-secret-manager"
version = "2.19.0"
description = "Secret Manager API"
`

	want := `[project]
name = "google-cloud-secret-manager"
version = "2.20.0"
description = "Secret Manager API"
`

	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile)

	if err := updatePyprojectToml(tmpfile, "2.20.0"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateSetupPy(t *testing.T) {
	content := `from setuptools import setup

setup(
    name="google-cloud-secret-manager",
    version = "2.19.0",
    description="Secret Manager API",
)
`

	want := `from setuptools import setup

setup(
    name="google-cloud-secret-manager",
    version = "2.20.0",
    description="Secret Manager API",
)
`

	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile)

	if err := updateSetupPy(tmpfile, "2.20.0"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateSetupPy_NotExists(t *testing.T) {
	// Should not error if file doesn't exist
	if err := updateSetupPy("/nonexistent/setup.py", "2.20.0"); err != nil {
		t.Errorf("updateSetupPy should not error for non-existent file: %v", err)
	}
}

func TestUpdateGapicVersion(t *testing.T) {
	content := `# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0

__version__ = "2.19.0"
`

	want := `# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0

__version__ = "2.20.0"
`

	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile)

	if err := updateGapicVersion(tmpfile, "2.20.0"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatChangelogEntry(t *testing.T) {
	changes := []*Change{
		{Type: "feat", Subject: "add rotation support", CommitHash: "abc1234567890"},
		{Type: "fix", Subject: "handle nil pointers", CommitHash: "def4567890123"},
		{Type: "docs", Subject: "update README", CommitHash: "ghi7890123456"},
	}

	got := formatChangelogEntry("google-cloud-secret-manager", "1.16.0", changes)

	// Check that all sections are present
	if !containsSubstring(got, "### Features") {
		t.Error("missing Features section")
	}
	if !containsSubstring(got, "### Bug Fixes") {
		t.Error("missing Bug Fixes section")
	}
	if !containsSubstring(got, "### Documentation") {
		t.Error("missing Documentation section")
	}

	// Check that changes are present
	if !containsSubstring(got, "add rotation support") {
		t.Error("missing feature change")
	}
	if !containsSubstring(got, "handle nil pointers") {
		t.Error("missing fix change")
	}
	if !containsSubstring(got, "update README") {
		t.Error("missing docs change")
	}

	// Check that commit links are present
	if !containsSubstring(got, "abc1234") {
		t.Error("missing commit hash for feature")
	}
	if !containsSubstring(got, "def4567") {
		t.Error("missing commit hash for fix")
	}
}

func TestFilterByType(t *testing.T) {
	changes := []*Change{
		{Type: "feat", Subject: "feature 1"},
		{Type: "fix", Subject: "fix 1"},
		{Type: "feat", Subject: "feature 2"},
		{Type: "docs", Subject: "doc 1"},
		{Type: "fix", Subject: "fix 2"},
	}

	for _, test := range []struct {
		name       string
		changeType string
		want       int
	}{
		{name: "feat", changeType: "feat", want: 2},
		{name: "fix", changeType: "fix", want: 2},
		{name: "docs", changeType: "docs", want: 1},
		{name: "chore", changeType: "chore", want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := filterByType(changes, test.changeType)
			if len(got) != test.want {
				t.Errorf("filterByType(%q) returned %d changes, want %d", test.changeType, len(got), test.want)
			}
		})
	}
}

func TestUpdateChangelog_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	changes := []*Change{
		{Type: "feat", Subject: "initial release", CommitHash: "abc1234567890"},
	}

	if err := updateChangelog(changelogPath, "google-cloud-secret-manager", "1.0.0", changes); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(got)
	if !containsSubstring(content, "# Changelog") {
		t.Error("missing changelog header")
	}
	if !containsSubstring(content, "## [1.0.0]") {
		t.Error("missing version header")
	}
	if !containsSubstring(content, "initial release") {
		t.Error("missing change")
	}
}

func TestUpdateChangelog_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create existing changelog
	existing := `# Changelog

## [1.0.0](https://github.com/googleapis/google-cloud-python/releases/tag/google-cloud-secret-manager-v1.0.0) (2025-01-01)

### Features

* initial release ([abc1234](https://github.com/googleapis/google-cloud-python/commit/abc1234))
`
	if err := os.WriteFile(changelogPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	// Add new entry
	changes := []*Change{
		{Type: "feat", Subject: "add new feature", CommitHash: "def4567890123"},
	}

	if err := updateChangelog(changelogPath, "google-cloud-secret-manager", "1.1.0", changes); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(got)
	// Check that new version comes before old version
	newIdx := indexOfSubstring(content, "## [1.1.0]")
	oldIdx := indexOfSubstring(content, "## [1.0.0]")

	if newIdx == -1 {
		t.Error("missing new version")
	}
	if oldIdx == -1 {
		t.Error("old version disappeared")
	}
	if newIdx >= oldIdx {
		t.Error("new version should come before old version")
	}
}

func TestFindLibrarySection(t *testing.T) {
	lines := []string{
		"# Changelog",
		"",
		"## google-cloud-pubsub",
		"",
		"### [1.0.0] (2025-01-01)",
		"",
		"## google-cloud-storage",
		"",
		"### [2.0.0] (2025-01-02)",
	}

	for _, test := range []struct {
		name    string
		libName string
		want    int
	}{
		{name: "pubsub", libName: "google-cloud-pubsub", want: 3},
		{name: "storage", libName: "google-cloud-storage", want: 7},
		{name: "not-found", libName: "google-cloud-secret-manager", want: 3}, // After "# Changelog"
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findLibrarySection(lines, test.libName)
			if got != test.want {
				t.Errorf("findLibrarySection(%q) = %d, want %d", test.libName, got, test.want)
			}
		})
	}
}

func TestFindFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file structure
	files := []string{
		"google/cloud/secretmanager/gapic_version.py",
		"google/cloud/secretmanager_v1/gapic_version.py",
		"google/cloud/pubsub/gapic_version.py",
		"other/file.txt",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test ** pattern
	pattern := filepath.Join(tmpDir, "**", "gapic_version.py")
	matches, err := findFiles(pattern)
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 3 {
		t.Errorf("findFiles found %d files, want 3", len(matches))
	}

	// Test simple pattern
	pattern2 := filepath.Join(tmpDir, "google/cloud/secretmanager/gapic_version.py")
	matches2, err := findFiles(pattern2)
	if err != nil {
		t.Fatal(err)
	}

	if len(matches2) != 1 {
		t.Errorf("findFiles found %d files, want 1", len(matches2))
	}
}

func TestUpdateSnippetMetadataFile(t *testing.T) {
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "snippet_metadata.json")

	// Create test metadata
	metadata := `{
  "clientVersion": "1.0.0",
  "snippets": []
}`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil {
		t.Fatal(err)
	}

	// Update version
	if err := updateSnippetMetadataFile(metadataPath, "1.1.0"); err != nil {
		t.Fatal(err)
	}

	// Read updated content
	got, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}

	if !containsSubstring(string(got), `"clientVersion": "1.1.0"`) {
		t.Errorf("clientVersion not updated correctly: %s", string(got))
	}
}

func TestFileExists(t *testing.T) {
	tmpfile := createTempFile(t, "test")
	defer os.Remove(tmpfile)

	if !fileExists(tmpfile) {
		t.Error("fileExists should return true for existing file")
	}

	if fileExists("/nonexistent/file.txt") {
		t.Error("fileExists should return false for non-existent file")
	}
}

// Helper functions

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	return tmpfile.Name()
}

func containsSubstring(s, substr string) bool {
	return indexOfSubstring(s, substr) != -1
}

func indexOfSubstring(s, substr string) int {
	idx := 0
	for idx < len(s) {
		if idx+len(substr) > len(s) {
			return -1
		}
		if s[idx:idx+len(substr)] == substr {
			return idx
		}
		idx++
	}
	return -1
}
