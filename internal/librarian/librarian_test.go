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
	"runtime"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
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

	if err := Run(t.Context(), []string{"librarian", "init"}); err != nil {
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
}

func TestRunInit_CreatesConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		language string
	}{
		{
			name:     "no language",
			language: "",
		},
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

			if err := runInit(test.language, nil); err != nil {
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

	if err := runInit("go", nil); err != nil {
		t.Fatal(err)
	}

	if err := runInit("go", nil); err == nil {
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
			name:           "no language",
			language:       "",
			wantHasSources: false,
		},
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

			if err := runInit(test.language, nil); err != nil {
				t.Fatal(err)
			}

			cfg := &config.Config{}
			if err := cfg.Read("librarian.yaml"); err != nil {
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
