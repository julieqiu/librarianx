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
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/rust"
)

func TestRun_Version(t *testing.T) {
	err := Run(t.Context(), []string{"librarian", "--version"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun_VersionCommand(t *testing.T) {
	err := Run(t.Context(), []string{"librarian", "version"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun_Help(t *testing.T) {
	err := Run(t.Context(), []string{"librarian", "--help"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun_CommandsExist(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "init command exists",
			args:    []string{"librarian", "init", "--help"},
			wantErr: "",
		},
		{
			name:    "version command exists",
			args:    []string{"librarian", "version"},
			wantErr: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Run(t.Context(), test.args)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInitCommand_NoLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	err := Run(t.Context(), []string{"librarian", "init"})
	if err == nil {
		t.Error("init should fail when no language is provided")
	}
}

func TestRunInit_CreatesConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		language string
	}{
		{
			name:     "go",
			language: "go",
		},
		{
			name:     "python",
			language: "python",
		},
		{
			name:     "rust",
			language: "rust",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			if err := runInit(test.language, nil, false); err != nil {
				t.Fatal(err)
			}

			if _, err := os.Stat("librarian.yaml"); os.IsNotExist(err) {
				t.Fatal("librarian.yaml was not created")
			}

			cfg, err := os.ReadFile("librarian.yaml")
			if err != nil {
				t.Fatal(err)
			}

			if len(cfg) == 0 {
				t.Error("created config is empty")
			}
		})
	}
}

func TestRunInit_PreventsOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := runInit("go", nil, false); err == nil {
		t.Error("runInit() should fail when librarian.yaml exists")
	} else if !errors.Is(err, errConfigAlreadyExists) {
		t.Errorf("want %v; got %v", errConfigAlreadyExists, err)
	}
}

func TestVersion_IncludesOSArch(t *testing.T) {
	version := Version()
	expectedSuffix := runtime.GOOS + "/" + runtime.GOARCH
	if !strings.Contains(version, expectedSuffix) {
		t.Errorf("Version() = %q, want it to contain %q", version, expectedSuffix)
	}
}

func TestRunInit_ConfigContent(t *testing.T) {
	for _, test := range []struct {
		name           string
		language       string
		wantHasSources bool
	}{
		{
			name:           "go",
			language:       "go",
			wantHasSources: false, // nil source passed in test
		},
		{
			name:           "python",
			language:       "python",
			wantHasSources: false,
		},
		{
			name:           "rust",
			language:       "rust",
			wantHasSources: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			if err := runInit(test.language, nil, false); err != nil {
				t.Fatal(err)
			}

			cfg, err := config.Read("librarian.yaml")
			if err != nil {
				t.Fatal(err)
			}

			if cfg.Language != test.language {
				t.Errorf("Language = %q, want %q", cfg.Language, test.language)
			}

			if cfg.Release == nil {
				t.Fatal("Release is nil")
			}
			if cfg.Release.TagFormat != "{name}/v{version}" {
				t.Errorf("TagFormat = %q, want %q", cfg.Release.TagFormat, "{name}/v{version}")
			}

			hasSources := cfg.Sources.Googleapis != nil
			if hasSources != test.wantHasSources {
				t.Errorf("has sources = %v, want %v", hasSources, test.wantHasSources)
			}
		})
	}
}

func TestRunSet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := runSet("release.tag_format", "{id}/v{version}"); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Release == nil {
		t.Fatal("Release is nil")
	}
	if cfg.Release.TagFormat != "{id}/v{version}" {
		t.Errorf("got %q, want %q", cfg.Release.TagFormat, "{id}/v{version}")
	}
}

func TestRunSet_ConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	err := runSet("language", "go")
	if err == nil {
		t.Error("runSet() should fail when librarian.yaml does not exist")
	} else if !errors.Is(err, errConfigNotFound) {
		t.Errorf("want %v; got %v", errConfigNotFound, err)
	}
}

func TestRunSet_InvalidField(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
	}{
		{
			name:  "invalid field",
			field: "invalid.field",
		},
		{
			name:  "language field",
			field: "language",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			if err := runInit("go", nil, false); err != nil {
				t.Fatal(err)
			}

			err := runSet(test.field, "value")
			if err == nil {
				t.Error("runSet() should fail with invalid key")
			} else if !errors.Is(err, errInvalidKey) {
				t.Errorf("want %v; got %v", errInvalidKey, err)
			}
		})
	}
}

func TestRunUnset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := runSet("release.tag_format", "test-value"); err != nil {
		t.Fatal(err)
	}

	if err := runUnset("release.tag_format"); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Release != nil && cfg.Release.TagFormat != "" {
		t.Errorf("got %q, want empty string", cfg.Release.TagFormat)
	}
}

func TestRunUnset_ConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	err := runUnset("language")
	if err == nil {
		t.Error("runUnset() should fail when librarian.yaml does not exist")
	} else if !errors.Is(err, errConfigNotFound) {
		t.Errorf("want %v; got %v", errConfigNotFound, err)
	}
}

func TestRunUnset_InvalidField(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
	}{
		{
			name:  "invalid field",
			field: "invalid.field",
		},
		{
			name:  "language field",
			field: "language",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			if err := runInit("go", nil, false); err != nil {
				t.Fatal(err)
			}

			err := runUnset(test.field)
			if err == nil {
				t.Error("runUnset() should fail with invalid key")
			} else if !errors.Is(err, errInvalidKey) {
				t.Errorf("want %v; got %v", errInvalidKey, err)
			}
		})
	}
}

func TestRunAdd(t *testing.T) {
	t.Skip("TODO: Update test for new config format - add command uses legacy format")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := runAdd("secretmanager", []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta2"}, ""); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("got %d libraries, want 1", len(cfg.Libraries))
	}

	// TODO: Update for new format where Name is the identifier and API is in config
	// if cfg.Libraries[0].Name != "secretmanager" {
	// 	t.Errorf("got name %q, want %q", cfg.Libraries[0].Name, "secretmanager")
	// }
}

func TestRunAdd_ConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	err := runAdd("secretmanager", []string{"google/cloud/secretmanager/v1"}, "")
	if err == nil {
		t.Error("runAdd() should fail when librarian.yaml does not exist")
	} else if !errors.Is(err, errConfigNotFound) {
		t.Errorf("want %v; got %v", errConfigNotFound, err)
	}
}

func TestRunAdd_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := runAdd("secretmanager", []string{"google/cloud/secretmanager/v1"}, ""); err != nil {
		t.Fatal(err)
	}

	err := runAdd("secretmanager", []string{"google/cloud/secretmanager/v1"}, "")
	if err == nil {
		t.Error("runAdd() should fail when library already exists")
	}
}

func TestRunAdd_WithLocation(t *testing.T) {
	t.Skip("TODO: Update test for new config format - add command uses legacy format")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := runAdd("storage", nil, "storage/"); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("got %d libraries, want 1", len(cfg.Libraries))
	}

	// TODO: Update for new format where Name is the identifier
	// if cfg.Libraries[0].Name != "storage" {
	// 	t.Errorf("got name %q, want %q", cfg.Libraries[0].Name, "storage")
	// }
}

func TestRunGenerate_LibraryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	err := runGenerate(t.Context(), "nonexistent")
	if err == nil {
		t.Error("runGenerate() should fail when library does not exist")
	}
}

func TestRunGenerate_HandwrittenLibrary(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	if err := runAdd("storage", nil, "storage/"); err != nil {
		t.Fatal(err)
	}

	err := runGenerate(t.Context(), "storage")
	if err == nil {
		t.Error("runGenerate() should fail for handwritten librarys")
	}
}

func TestRunGenerate_ConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	err := runGenerate(t.Context(), "secretmanager")
	if err == nil {
		t.Error("runGenerate() should fail when librarian.yaml does not exist")
	} else if !errors.Is(err, errConfigNotFound) {
		t.Errorf("want %v; got %v", errConfigNotFound, err)
	}
}

func TestGenerateRust(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name    string
		cfg     *config.Config
		library *config.Library
		wantErr bool
	}{
		{
			name: "single API",
			cfg: &config.Config{
				Language: "rust",
				Generate: &config.Generate{
					Output: filepath.Join(tmpDir, "packages", "/"),
				},
			},
			library: &config.Library{
				Name: "secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1"},
			},
			wantErr: true,
		},
		{
			name: "multiple APIs",
			cfg: &config.Config{
				Language: "rust",
			},
			library: &config.Library{
				Name: "secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta2"},
			},
			wantErr: true,
		},
		{
			name: "no APIs",
			cfg: &config.Config{
				Language: "rust",
			},
			library: &config.Library{
				Name: "storage",
				Apis: []string{},
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := rust.Generate(t.Context(), test.cfg, test.library, "")
			if test.wantErr && err == nil {
				t.Error("rust.Generate() should fail")
			}
			if !test.wantErr && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGenerateRust_RequiresSingleAPI(t *testing.T) {
	cfg := &config.Config{
		Language: "rust",
	}
	library := &config.Library{
		Name: "secretmanager",
		Apis: []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta2"},
	}

	err := rust.Generate(t.Context(), cfg, library, "")
	if err == nil {
		t.Error("rust.Generate() should fail with multiple APIs")
	}
	want := "rust generation requires exactly one API per library"
	if err != nil && !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v, want it to contain %q", err, want)
	}
}

func TestRunInitRust_All(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInitRust(true); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Language != "rust" {
		t.Errorf("Language = %q, want %q", cfg.Language, "rust")
	}

	if len(cfg.Libraries) != 1 {
		t.Fatalf("Libraries length = %d, want 1", len(cfg.Libraries))
	}

	if cfg.Libraries[0].Name != "*" {
		t.Errorf("Libraries[0].Name = %q, want %q", cfg.Libraries[0].Name, "*")
	}

	if cfg.Defaults == nil {
		t.Fatal("Defaults is nil")
	}

	if cfg.Defaults.Output != "src/generated/" {
		t.Errorf("Defaults.Output = %q, want %q", cfg.Defaults.Output, "src/generated/")
	}

	if cfg.Defaults.OneLibraryPer != "version" {
		t.Errorf("Defaults.OneLibraryPer = %q, want %q", cfg.Defaults.OneLibraryPer, "version")
	}

	if cfg.Defaults.ReleaseLevel != "stable" {
		t.Errorf("Defaults.ReleaseLevel = %q, want %q", cfg.Defaults.ReleaseLevel, "stable")
	}

	if cfg.Defaults.Rust == nil {
		t.Fatal("Defaults.Rust is nil")
	}

	if len(cfg.Defaults.Rust.DisabledRustdocWarnings) != 2 {
		t.Errorf("DisabledRustdocWarnings length = %d, want 2", len(cfg.Defaults.Rust.DisabledRustdocWarnings))
	}
}

func TestRunGenerateAll_NoLibraries(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, false); err != nil {
		t.Fatal(err)
	}

	err := runGenerateAll(t.Context())
	if err == nil {
		t.Error("runGenerateAll() should fail when no libraries are configured")
	}
}

func TestRunGenerateAll_Wildcard(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := runInit("go", nil, true); err != nil {
		t.Fatal(err)
	}

	err := runGenerateAll(t.Context())
	if err == nil {
		t.Error("runGenerateAll() should fail with wildcard (not yet implemented)")
	}
	if !strings.Contains(err.Error(), "wildcard") {
		t.Errorf("error should mention wildcard, got: %v", err)
	}
}
