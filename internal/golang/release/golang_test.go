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

package release

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGo_Release(t *testing.T) {
	for _, test := range []struct {
		name                string
		library             *config.Library
		version             string
		changes             []*Change
		initialFiles        map[string]string
		wantChangelogSubstr string
		wantVersion         string
		wantSnippetVersion  string
	}{
		{
			name: "standard library",
			library: &config.Library{
				Name: "secretmanager",
			},
			version: "1.16.0",
			changes: []*Change{
				{Type: "feat", Subject: "add new GetSecret API", CommitHash: "abcdef123456"},
				{Type: "feat", Subject: "another feature", CommitHash: "zxcvbn098765"},
				{Type: "fix", Subject: "correct typo in documentation", CommitHash: "123456abcdef"},
			},
			initialFiles: map[string]string{
				"secretmanager/CHANGES.md":          "# Changes\n\n## [1.15.0]\n- Old stuff.",
				"secretmanager/internal/version.go": `package internal; const Version = "1.15.0"`,
				"internal/generated/snippets/secretmanager/apiv1/snippet_metadata.google.cloud.secretmanager.v1.json": `{"version": "1.15.0"}`,
			},
			wantChangelogSubstr: "## [1.16.0](https://github.com/googleapis/google-cloud-go/releases/tag/secretmanager%2Fv1.16.0)",
			wantVersion:         "1.16.0",
			wantSnippetVersion:  `"version": "1.16.0"`,
		},
		{
			name: "root module",
			library: &config.Library{
				Name:        "root-module",
				SourceRoots: []string{"."},
			},
			version: "1.16.0",
			changes: []*Change{
				{Type: "feat", Subject: "add new feature", CommitHash: "abcdef123456"},
			},
			initialFiles: map[string]string{
				"CHANGES.md":          "# Changes\n\n## [1.15.0]\n- Old stuff.",
				"internal/version.go": `package internal; const Version = "1.15.0"`,
			},
			wantChangelogSubstr: "## [1.16.0](https://github.com/googleapis/google-cloud-go/releases/tag/root-module%2Fv1.16.0)",
			wantVersion:         "1.16.0",
		},
		{
			name: "library with explicit location",
			library: &config.Library{
				Name:     "storage",
				Location: "packages/storage",
			},
			version: "2.0.0",
			changes: []*Change{
				{Type: "feat", Subject: "add streaming support", CommitHash: "abc123"},
			},
			initialFiles: map[string]string{
				"packages/storage/CHANGES.md":          "# Changes\n",
				"packages/storage/internal/version.go": `package internal; const Version = "1.0.0"`,
			},
			wantChangelogSubstr: "## [2.0.0](https://github.com/googleapis/google-cloud-go/releases/tag/storage%2Fv2.0.0)",
			wantVersion:         "2.0.0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()

			// Create initial files
			for path, content := range test.initialFiles {
				fullPath := filepath.Join(repoRoot, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			r := &releaser{repoRoot: repoRoot}
			err := r.releaseWithOptions(context.Background(), test.library, test.version, test.changes, false)
			if err != nil {
				t.Fatal(err)
			}

			// Verify changelog
			changelogPath, err := r.changelogPath(test.library)
			if err != nil {
				t.Fatal(err)
			}
			changelog, err := os.ReadFile(changelogPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(changelog), test.wantChangelogSubstr) {
				t.Errorf("changelog does not contain expected substring\ngot:\n%s\nwant substring: %s", string(changelog), test.wantChangelogSubstr)
			}

			// Verify version file
			libPath, err := r.libraryPath(test.library)
			if err != nil {
				t.Fatal(err)
			}
			versionPath := filepath.Join(libPath, "internal", "version.go")
			assertVersion(t, versionPath, test.wantVersion)

			// Verify snippet metadata if expected
			if test.wantSnippetVersion != "" {
				snippetPath := filepath.Join(repoRoot, "internal/generated/snippets", test.library.Name, "apiv1/snippet_metadata.google.cloud.secretmanager.v1.json")
				snippet, err := os.ReadFile(snippetPath)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(snippet), test.wantSnippetVersion) {
					t.Errorf("snippet does not contain expected version\ngot: %s\nwant substring: %s", string(snippet), test.wantSnippetVersion)
				}
			}
		})
	}
}

func TestGo_updateChangelog(t *testing.T) {
	fixedTime := time.Date(2025, 9, 11, 0, 0, 0, 0, time.UTC)

	for _, test := range []struct {
		name            string
		library         *config.Library
		version         string
		changes         []*Change
		existingContent string
		wantContains    string
	}{
		{
			name: "new changelog",
			library: &config.Library{
				Name: "test",
			},
			version: "1.0.0",
			changes: []*Change{
				{Type: "feat", Subject: "initial release", CommitHash: "abc123"},
			},
			existingContent: "# Changes\n",
			wantContains:    "## [1.0.0](https://github.com/googleapis/google-cloud-go/releases/tag/test%2Fv1.0.0) (2025-09-11)\n\n### Features\n\n* initial release ([abc123]",
		},
		{
			name: "grouped by type",
			library: &config.Library{
				Name: "test",
			},
			version: "2.0.0",
			changes: []*Change{
				{Type: "feat", Subject: "feature b", CommitHash: "bbb"},
				{Type: "fix", Subject: "fix a", CommitHash: "aaa"},
				{Type: "feat", Subject: "feature a", CommitHash: "ccc"},
				{Type: "perf", Subject: "performance improvement", CommitHash: "ddd"},
			},
			existingContent: "# Changes\n",
			wantContains:    "### Features\n\n* feature a ([ccc",
		},
		{
			name: "alphabetically sorted",
			library: &config.Library{
				Name: "test",
			},
			version: "3.0.0",
			changes: []*Change{
				{Type: "feat", Subject: "zebra feature", CommitHash: "zzz"},
				{Type: "feat", Subject: "apple feature", CommitHash: "aaa"},
				{Type: "feat", Subject: "middle feature", CommitHash: "mmm"},
			},
			existingContent: "# Changes\n",
			wantContains:    "* apple feature ([aaa",
		},
		{
			name: "skip when already up-to-date",
			library: &config.Library{
				Name: "test",
			},
			version:         "1.5.0",
			changes:         []*Change{{Type: "feat", Subject: "new feature"}},
			existingContent: "# Changes\n\n## [1.5.0]\n- Already there.",
			wantContains:    "## [1.5.0]\n- Already there.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			changelogPath := filepath.Join(repoRoot, test.library.Name, "CHANGES.md")
			if err := os.MkdirAll(filepath.Dir(changelogPath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(changelogPath, []byte(test.existingContent), 0644); err != nil {
				t.Fatal(err)
			}

			r := &releaser{repoRoot: repoRoot}
			err := r.updateChangelog(test.library, test.version, test.changes, fixedTime)
			if err != nil {
				t.Fatal(err)
			}

			content, err := os.ReadFile(changelogPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), test.wantContains) {
				t.Errorf("changelog does not contain expected content\ngot:\n%s\nwant substring: %s", string(content), test.wantContains)
			}
		})
	}
}

func TestGo_updateVersionFile(t *testing.T) {
	for _, test := range []struct {
		name        string
		version     string
		wantVersion string
	}{
		{
			name:        "basic version",
			version:     "1.0.0",
			wantVersion: "1.0.0",
		},
		{
			name:        "prerelease version",
			version:     "2.0.0-rc.1",
			wantVersion: "2.0.0-rc.1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			libPath := filepath.Join(repoRoot, "testlib")

			r := &releaser{repoRoot: repoRoot}
			err := r.updateVersionFile(libPath, test.version)
			if err != nil {
				t.Fatal(err)
			}

			versionPath := filepath.Join(libPath, "internal", "version.go")
			assertVersion(t, versionPath, test.wantVersion)
		})
	}
}

func TestGo_updateSnippetMetadata(t *testing.T) {
	for _, test := range []struct {
		name         string
		library      *config.Library
		version      string
		initialFiles map[string]string
		wantVersion  string
	}{
		{
			name: "update $VERSION placeholder",
			library: &config.Library{
				Name: "secretmanager",
			},
			version: "1.16.0",
			initialFiles: map[string]string{
				"internal/generated/snippets/secretmanager/apiv1/snippet_metadata.google.cloud.secretmanager.v1.json": `{"version": "$VERSION"}`,
			},
			wantVersion: `"version": "1.16.0"`,
		},
		{
			name: "update existing version",
			library: &config.Library{
				Name: "pubsub",
			},
			version: "2.0.0",
			initialFiles: map[string]string{
				"internal/generated/snippets/pubsub/apiv1/snippet_metadata.google.cloud.pubsub.v1.json": `{"version": "1.0.0"}`,
			},
			wantVersion: `"version": "2.0.0"`,
		},
		{
			name: "no snippets directory",
			library: &config.Library{
				Name: "nosnippets",
			},
			version:      "1.0.0",
			initialFiles: map[string]string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()

			for path, content := range test.initialFiles {
				fullPath := filepath.Join(repoRoot, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			r := &releaser{repoRoot: repoRoot}
			err := r.updateSnippetMetadata(test.library, test.version)
			if err != nil {
				t.Fatal(err)
			}

			if test.wantVersion != "" {
				for path := range test.initialFiles {
					fullPath := filepath.Join(repoRoot, path)
					content, err := os.ReadFile(fullPath)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(content), test.wantVersion) {
						t.Errorf("snippet does not contain expected version\ngot: %s\nwant substring: %s", string(content), test.wantVersion)
					}
				}
			}
		})
	}
}

func TestGo_libraryPath(t *testing.T) {
	for _, test := range []struct {
		name     string
		library  *config.Library
		wantPath string
	}{
		{
			name: "standard library",
			library: &config.Library{
				Name: "secretmanager",
			},
			wantPath: "secretmanager",
		},
		{
			name: "root module with SourceRoots",
			library: &config.Library{
				Name:        "mylib",
				SourceRoots: []string{"."},
			},
			wantPath: ".",
		},
		{
			name: "root-module special name",
			library: &config.Library{
				Name: "root-module",
			},
			wantPath: ".",
		},
		{
			name: "explicit location",
			library: &config.Library{
				Name:     "storage",
				Location: "packages/storage",
			},
			wantPath: "packages/storage",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			r := &releaser{repoRoot: repoRoot}
			got, err := r.libraryPath(test.library)
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(repoRoot, test.wantPath)
			if got != want {
				t.Errorf("libraryPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestGo_formatTag(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		version string
		want    string
	}{
		{
			name:    "standard library",
			library: &config.Library{Name: "secretmanager"},
			version: "1.16.0",
			want:    "secretmanager/v1.16.0",
		},
		{
			name:    "prerelease",
			library: &config.Library{Name: "pubsub"},
			version: "2.0.0-rc.1",
			want:    "pubsub/v2.0.0-rc.1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &releaser{repoRoot: ""}
			got := r.formatTag(test.library, test.version)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsRootRepoModule(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    bool
	}{
		{
			name: "standard library",
			library: &config.Library{
				Name: "secretmanager",
			},
			want: false,
		},
		{
			name: "root module with dot in SourceRoots",
			library: &config.Library{
				Name:        "mylib",
				SourceRoots: []string{"."},
			},
			want: true,
		},
		{
			name: "root-module special name",
			library: &config.Library{
				Name: "root-module",
			},
			want: true,
		},
		{
			name: "multiple source roots with dot",
			library: &config.Library{
				Name:        "mylib",
				SourceRoots: []string{".", "other"},
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isRootRepoModule(test.library)
			if got != test.want {
				t.Errorf("isRootRepoModule() = %v, want %v", got, test.want)
			}
		})
	}
}

// assertVersion parses a version.go file and checks the Version constant.
func assertVersion(t *testing.T, versionGoPath, wantVersion string) {
	t.Helper()
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, versionGoPath, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if valueSpec.Names[0].Name == "Version" {
				gotVersion := valueSpec.Values[0].(*ast.BasicLit).Value
				// trim quotes
				gotVersion = gotVersion[1 : len(gotVersion)-1]
				if gotVersion != wantVersion {
					t.Errorf("version.go Version = %q, want %q", gotVersion, wantVersion)
				}
				return
			}
		}
	}
	t.Errorf("could not find Version constant in version.go")
}
