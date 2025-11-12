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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadWrite(t *testing.T) {
	for _, test := range []struct {
		name     string
		testdata string
	}{
		{
			name:     "release_only",
			testdata: "testdata/release_only.yaml",
		},
		{
			name:     "go",
			testdata: "testdata/go.yaml",
		},
		{
			name:     "python",
			testdata: "testdata/python.yaml",
		},
		{
			name:     "rust",
			testdata: "testdata/rust.yaml",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			original, err := Read(test.testdata)
			if err != nil {
				t.Fatal(err)
			}

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "librarian.yaml")
			if err := original.Write(configPath); err != nil {
				t.Fatal(err)
			}

			got, err := Read(configPath)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(original, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfig_Set(t *testing.T) {
	for _, test := range []struct {
		name     string
		language string
		version  string
		source   *Source
		sets     map[string]string
		testdata string
	}{
		{
			name:     "python",
			language: "python",
			version:  "v0.5.0",
			source: &Source{
				URL:    "https://github.com/googleapis/googleapis/archive/9fcfbea0aa5b50fa22e190faceb073d74504172b.tar.gz",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
			sets: map[string]string{
				"release.tag_format": "{id}-v{version}",
				"generate.output":    "packages/{name}/",
			},
			testdata: "testdata/python.yaml",
		},
		{
			name:     "rust",
			language: "rust",
			version:  "v0.5.0",
			source: &Source{
				URL:    "https://github.com/googleapis/googleapis/archive/9fcfbea0aa5b50fa22e190faceb073d74504172b.tar.gz",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
			sets: map[string]string{
				"generate.output": "src/generated/{api.path}/",
			},
			testdata: "testdata/rust.yaml",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := New(test.version, test.language, test.source)

			for field, value := range test.sets {
				if err := cfg.Set(field, value); err != nil {
					t.Fatal(err)
				}
			}

			want, err := Read(test.testdata)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(want, cfg); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfig_Set_InvalidField(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	err := cfg.Set("invalid.field", "value")
	if err == nil {
		t.Error("Set() should fail with invalid field")
	}
}

func TestConfig_Unset(t *testing.T) {
	source := &Source{
		URL:    "https://github.com/googleapis/googleapis/archive/9fcfbea0aa5b50fa22e190faceb073d74504172b.tar.gz",
		SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
	}
	cfg := New("v0.5.0", "python", source)

	if err := cfg.Set("release.tag_format", "{id}-v{version}"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Set("generate.output", "packages/{name}/"); err != nil {
		t.Fatal(err)
	}

	if err := cfg.Unset("generate.output"); err != nil {
		t.Fatal(err)
	}

	want := New("v0.5.0", "python", source)
	if err := want.Set("release.tag_format", "{id}-v{version}"); err != nil {
		t.Fatal(err)
	}
	want.Generate = &Generate{
		Output: "",
	}

	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestConfig_Unset_InvalidField(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	err := cfg.Unset("invalid.field")
	if err == nil {
		t.Error("Unset() should fail with invalid field")
	}
}

func TestConfig_Add(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	if err := cfg.Add("secretmanager", []string{"google/cloud/secretmanager/v1"}, "", ""); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("got %d libraries, want 1", len(cfg.Libraries))
	}

	if cfg.Libraries[0].Name != "secretmanager" {
		t.Errorf("got name %q, want %q", cfg.Libraries[0].Name, "secretmanager")
	}

	wantApis := []string{"google/cloud/secretmanager/v1"}
	if diff := cmp.Diff(wantApis, cfg.Libraries[0].Apis); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestConfig_Add_Duplicate(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	if err := cfg.Add("secretmanager", []string{"google/cloud/secretmanager/v1"}, "", ""); err != nil {
		t.Fatal(err)
	}

	err := cfg.Add("secretmanager", []string{"google/cloud/secretmanager/v1"}, "", "")
	if err == nil {
		t.Error("Add() should fail when library with same name and apis already exists")
	}
}

func TestConfig_Add_SameNameDifferentApis(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	if err := cfg.Add("secretmanager", []string{"google/cloud/secretmanager/v1"}, "", ""); err != nil {
		t.Fatal(err)
	}

	if err := cfg.Add("secretmanager", []string{"google/cloud/secretmanager/v1beta2"}, "", ""); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 2 {
		t.Errorf("got %d libraries, want 2", len(cfg.Libraries))
	}
}

func TestConfig_Add_EmptyName(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	err := cfg.Add("", []string{"google/cloud/secretmanager/v1"}, "", "")
	if err == nil {
		t.Error("Add() should fail when name is empty")
	}
}

func TestConfig_Add_EmptyApis(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	err := cfg.Add("secretmanager", []string{}, "", "")
	if err == nil {
		t.Error("Add() should fail when apis is empty and no location provided")
	}
}

func TestConfig_Add_WithLocation(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	if err := cfg.Add("storage", nil, "storage/", ""); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("got %d libraries, want 1", len(cfg.Libraries))
	}

	if cfg.Libraries[0].Name != "storage" {
		t.Errorf("got name %q, want %q", cfg.Libraries[0].Name, "storage")
	}

	if cfg.Libraries[0].Location != "storage/" {
		t.Errorf("got location %q, want %q", cfg.Libraries[0].Location, "storage/")
	}

	if len(cfg.Libraries[0].Apis) != 0 {
		t.Errorf("got %d apis, want 0", len(cfg.Libraries[0].Apis))
	}
}

func TestLibrary_ExpandTemplate(t *testing.T) {
	for _, test := range []struct {
		name     string
		library  Library
		template string
		want     string
		wantErr  bool
	}{
		{
			name: "name_only",
			library: Library{
				Name: "secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1"},
			},
			template: "{name}/",
			want:     "secretmanager/",
		},
		{
			name: "api_path_single_api",
			library: Library{
				Name: "google-cloud-secretmanager-v1",
				Apis: []string{"google/cloud/secretmanager/v1"},
			},
			template: "src/generated/{api.path}/",
			want:     "src/generated/google/cloud/secretmanager/v1/",
		},
		{
			name: "api_path_multiple_apis_fails",
			library: Library{
				Name: "secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta2"},
			},
			template: "src/generated/{api.path}/",
			wantErr:  true,
		},
		{
			name: "packages_with_name",
			library: Library{
				Name: "google-cloud-secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta2"},
			},
			template: "packages/{name}/",
			want:     "packages/google-cloud-secretmanager/",
		},
		{
			name: "no_keywords",
			library: Library{
				Name: "secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1"},
			},
			template: "generated/",
			want:     "generated/",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.library.ExpandTemplate(test.template)
			if test.wantErr {
				if err == nil {
					t.Error("ExpandTemplate() should fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestLibrary_GeneratedLocation(t *testing.T) {
	for _, test := range []struct {
		name           string
		library        Library
		generateOutput string
		want           string
		wantErr        bool
	}{
		{
			name: "explicit_location",
			library: Library{
				Name:     "storage",
				Location: "storage/",
			},
			generateOutput: "{name}/",
			want:           "storage/",
		},
		{
			name: "computed_from_template_name",
			library: Library{
				Name: "secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1"},
			},
			generateOutput: "{name}/",
			want:           "secretmanager/",
		},
		{
			name: "computed_from_template_api",
			library: Library{
				Name: "google-cloud-secretmanager-v1",
				Apis: []string{"google/cloud/secretmanager/v1"},
			},
			generateOutput: "src/generated/{api.path}/",
			want:           "src/generated/google/cloud/secretmanager/v1/",
		},
		{
			name: "api_path_multiple_apis_fails",
			library: Library{
				Name: "secretmanager",
				Apis: []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta2"},
			},
			generateOutput: "src/generated/{api.path}/",
			wantErr:        true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.library.GeneratedLocation(test.generateOutput)
			if test.wantErr {
				if err == nil {
					t.Error("GeneratedLocation() should fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestConfig_Add_WithBazelParsing(t *testing.T) {
	tmpDir := t.TempDir()
	apiPath := "google/cloud/secretmanager/v1"
	buildDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	buildContent := `
py_gapic_library(
    name = "secretmanager_py_gapic",
    srcs = [":secretmanager_proto"],
    grpc_service_config = "secretmanager_grpc_service_config.json",
    opt_args = [
        "warehouse-package-name=google-cloud-secret-manager",
        "python-gapic-namespace=google.cloud",
    ],
    rest_numeric_enums = True,
    service_yaml = "secretmanager_v1.yaml",
    transport = "grpc+rest",
)
`
	buildPath := filepath.Join(buildDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(buildContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := New("v0.5.0", "python", nil)
	if err := cfg.Add("secretmanager", []string{apiPath}, "", tmpDir); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Fatalf("got %d libraries, want 1", len(cfg.Libraries))
	}

	library := cfg.Libraries[0]
	if library.Name != "secretmanager" {
		t.Errorf("got name %q, want %q", library.Name, "secretmanager")
	}

	if library.Generate == nil {
		t.Fatal("library.Generate is nil")
	}

	if len(library.Generate.APIs) != 1 {
		t.Fatalf("got %d API configs, want 1", len(library.Generate.APIs))
	}

	apiCfg := library.Generate.APIs[0]
	want := API{
		Path:              apiPath,
		HasGAPIC:          true,
		GRPCServiceConfig: "secretmanager_grpc_service_config.json",
		ServiceYAML:       "secretmanager_v1.yaml",
		Transport:         "grpc+rest",
		RestNumericEnums:  true,
		Python: &PythonAPI{
			OptArgs: []string{
				"warehouse-package-name=google-cloud-secret-manager",
				"python-gapic-namespace=google.cloud",
			},
		},
	}

	if diff := cmp.Diff(want, apiCfg); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
