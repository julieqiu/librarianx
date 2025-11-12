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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/generate/golang/request"
)

func TestConfigValidate(t *testing.T) {
	for _, test := range []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				LibrarianDir: "/librarian",
				OutputDir:    "/output",
				RepoDir:      "/repo",
			},
			wantErr: false,
		},
		{
			name: "missing librarian dir",
			cfg: &Config{
				OutputDir: "/output",
				RepoDir:   "/repo",
			},
			wantErr: true,
		},
		{
			name: "missing output dir",
			cfg: &Config{
				LibrarianDir: "/librarian",
				RepoDir:      "/repo",
			},
			wantErr: true,
		},
		{
			name: "missing repo dir",
			cfg: &Config{
				LibrarianDir: "/librarian",
				OutputDir:    "/output",
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.Validate()
			if test.wantErr && err == nil {
				t.Fatal("expected error but got none")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateFileVersion(t *testing.T) {
	for _, test := range []struct {
		name     string
		filename string
		content  string
		regex    *regexp.Regexp
		version  string
		want     string
	}{
		{
			name:     "python version file",
			filename: "version.py",
			content:  `__version__ = "1.0.0"`,
			regex:    regexp.MustCompile(`__version__\s*=\s*"[^"]+"`),
			version:  "1.1.0",
			want:     `__version__ = "1.1.0"`,
		},
		{
			name:     "pyproject.toml",
			filename: "pyproject.toml",
			content:  `version = "1.0.0"`,
			regex:    regexp.MustCompile(`version\s*=\s*"[^"]+"`),
			version:  "1.1.0",
			want:     `version = "1.1.0"`,
		},
		{
			name:     "setup.py",
			filename: "setup.py",
			content: `setup(
    name="google-cloud-language",
    version = "1.0.0",
)`,
			regex:   regexp.MustCompile(`version\s*=\s*"[^"]+"`),
			version: "1.1.0",
			want: `setup(
    name="google-cloud-language",
    version = "1.1.0",
)`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, test.filename)
			if err := os.WriteFile(testFile, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}

			replacement := fmt.Sprintf(`version = "%s"`, test.version)
			if err := updateFileVersion(testFile, test.regex, replacement, test.version); err != nil {
				t.Fatal(err)
			}

			got, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateChangelog(t *testing.T) {
	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	lib := &request.Library{
		ID:      "google-cloud-language",
		Version: "1.1.0",
		Changes: []*request.Change{
			{Type: "feat", Subject: "Add new feature"},
			{Type: "fix", Subject: "Fix bug"},
		},
	}

	if err := createChangelog(changelogPath, lib); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(got)
	if !strings.Contains(content, "# Changelog") {
		t.Error("changelog missing header")
	}
	if !strings.Contains(content, "## 1.1.0") {
		t.Error("changelog missing version section")
	}
	if !strings.Contains(content, "feat: Add new feature") {
		t.Error("changelog missing feature change")
	}
	if !strings.Contains(content, "fix: Fix bug") {
		t.Error("changelog missing fix change")
	}
}

func TestAppendChangelog(t *testing.T) {
	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	existingContent := `# Changelog

## 1.0.0

* Initial release
`
	if err := os.WriteFile(changelogPath, []byte(existingContent), 0644); err != nil {
		t.Fatal(err)
	}

	lib := &request.Library{
		ID:      "google-cloud-language",
		Version: "1.1.0",
		Changes: []*request.Change{
			{Type: "feat", Subject: "Add new feature"},
		},
	}

	if err := appendChangelog(changelogPath, lib); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(got)
	if !strings.Contains(content, "## 1.1.0") {
		t.Error("changelog missing new version section")
	}
	if !strings.Contains(content, "## 1.0.0") {
		t.Error("changelog missing old version section")
	}

	// New version should come before old version
	newPos := strings.Index(content, "## 1.1.0")
	oldPos := strings.Index(content, "## 1.0.0")
	if newPos >= oldPos {
		t.Error("new version should appear before old version")
	}
}

func TestFindFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "google", "cloud", "language"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "google", "cloud", "language", "v1"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := []string{
		"google/cloud/language/gapic_version.py",
		"google/cloud/language/v1/gapic_version.py",
		"setup.py",
		"pyproject.toml",
	}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range []struct {
		name    string
		pattern string
		want    int
	}{
		{
			name:    "recursive gapic_version.py",
			pattern: "**/gapic_version.py",
			want:    2,
		},
		{
			name:    "setup.py",
			pattern: "setup.py",
			want:    1,
		},
		{
			name:    "pyproject.toml",
			pattern: "pyproject.toml",
			want:    1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findFiles(tmpDir, test.pattern)
			if err != nil {
				t.Fatal(err)
			}

			if len(got) != test.want {
				t.Errorf("got %d files, want %d", len(got), test.want)
			}
		})
	}
}

func TestReadReleaseReq(t *testing.T) {
	tmpDir := t.TempDir()
	reqPath := filepath.Join(tmpDir, "release-init-request.json")

	lib := &request.Library{
		ID:               "google-cloud-language",
		Version:          "1.1.0",
		ReleaseTriggered: true,
		Changes: []*request.Change{
			{Type: "feat", Subject: "Add new feature"},
		},
	}

	data, err := json.Marshal(lib)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(reqPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := readReleaseReq(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != lib.ID {
		t.Errorf("got ID %q, want %q", got.ID, lib.ID)
	}
	if got.Version != lib.Version {
		t.Errorf("got Version %q, want %q", got.Version, lib.Version)
	}
	if got.ReleaseTriggered != lib.ReleaseTriggered {
		t.Errorf("got ReleaseTriggered %v, want %v", got.ReleaseTriggered, lib.ReleaseTriggered)
	}
}
