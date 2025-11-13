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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
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

	if err := cfg.Add("secretmanager", nil); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("got %d libraries, want 1", len(cfg.Libraries))
	}

	if cfg.Libraries[0].Name != "secretmanager" {
		t.Errorf("got name %q, want %q", cfg.Libraries[0].Name, "secretmanager")
	}

	if cfg.Libraries[0].Config != nil {
		t.Errorf("got config %v, want nil", cfg.Libraries[0].Config)
	}
}

func TestConfig_Add_Duplicate(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	if err := cfg.Add("secretmanager", nil); err != nil {
		t.Fatal(err)
	}

	err := cfg.Add("secretmanager", nil)
	if err == nil {
		t.Error("Add() should fail when library with same name already exists")
	}
}

func TestConfig_Add_DifferentNames(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	if err := cfg.Add("secretmanager", nil); err != nil {
		t.Fatal(err)
	}

	if err := cfg.Add("storage", nil); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 2 {
		t.Errorf("got %d libraries, want 2", len(cfg.Libraries))
	}
}

func TestConfig_Add_EmptyName(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	err := cfg.Add("", nil)
	if err == nil {
		t.Error("Add() should fail when library name is empty")
	}
}

func TestConfig_Add_WithAPIConfig(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	config := &LibraryConfig{
		API: "google/cloud/secretmanager/v1",
	}
	if err := cfg.Add("secretmanager", config); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("got %d libraries, want 1", len(cfg.Libraries))
	}

	if cfg.Libraries[0].Config == nil {
		t.Fatal("library config is nil")
	}

	if apiStr, ok := cfg.Libraries[0].Config.API.(string); !ok || apiStr != "google/cloud/secretmanager/v1" {
		t.Errorf("got API %v, want %q", cfg.Libraries[0].Config.API, "google/cloud/secretmanager/v1")
	}
}

func TestConfig_Add_WithPathOverride(t *testing.T) {
	cfg := New("v0.5.0", "go", nil)

	config := &LibraryConfig{
		Path: "storage/",
	}
	if err := cfg.Add("storage", config); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("got %d libraries, want 1", len(cfg.Libraries))
	}

	if cfg.Libraries[0].Config == nil {
		t.Fatal("library config is nil")
	}

	if cfg.Libraries[0].Config.Path != "storage/" {
		t.Errorf("got path %q, want %q", cfg.Libraries[0].Config.Path, "storage/")
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

func TestLibraryEntry_UnmarshalYAML(t *testing.T) {
	for _, test := range []struct {
		name    string
		yaml    string
		want    LibraryEntry
		wantErr bool
	}{
		{
			name: "wildcard",
			yaml: `"*"`,
			want: LibraryEntry{
				Name:   "*",
				Config: nil,
			},
		},
		{
			name: "simple_name",
			yaml: `secretmanager:
  api: google/cloud/secretmanager/v1`,
			want: LibraryEntry{
				Name: "secretmanager",
				Config: &LibraryConfig{
					API: "google/cloud/secretmanager/v1",
				},
			},
		},
		{
			name: "with_keep",
			yaml: `google-cloud-bigquerystorage:
  api: google/cloud/bigquery/storage/v1
  keep:
    - google/cloud/bigquery_storage_v1/client.py
    - google/cloud/bigquery_storage_v1/reader.py`,
			want: LibraryEntry{
				Name: "google-cloud-bigquerystorage",
				Config: &LibraryConfig{
					API: "google/cloud/bigquery/storage/v1",
					Keep: []string{
						"google/cloud/bigquery_storage_v1/client.py",
						"google/cloud/bigquery_storage_v1/reader.py",
					},
				},
			},
		},
		{
			name: "with_disabled",
			yaml: `google-cloud-broken:
  api: google/cloud/broken/v1
  disabled: true
  reason: "Missing BUILD.bazel configuration"`,
			want: LibraryEntry{
				Name: "google-cloud-broken",
				Config: &LibraryConfig{
					API:      "google/cloud/broken/v1",
					Disabled: true,
					Reason:   "Missing BUILD.bazel configuration",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got LibraryEntry
			err := got.UnmarshalYAML(func(v interface{}) error {
				return yaml.Unmarshal([]byte(test.yaml), v)
			})
			if test.wantErr {
				if err == nil {
					t.Error("UnmarshalYAML() should fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLibraryEntry_MarshalYAML(t *testing.T) {
	for _, test := range []struct {
		name  string
		entry LibraryEntry
		want  string
	}{
		{
			name: "wildcard",
			entry: LibraryEntry{
				Name:   "*",
				Config: nil,
			},
			want: "'*'\n",
		},
		{
			name: "with_api",
			entry: LibraryEntry{
				Name: "secretmanager",
				Config: &LibraryConfig{
					API: "google/cloud/secretmanager/v1",
				},
			},
			want: "secretmanager:\n    api: google/cloud/secretmanager/v1\n",
		},
		{
			name: "with_keep",
			entry: LibraryEntry{
				Name: "google-cloud-bigquerystorage",
				Config: &LibraryConfig{
					API: "google/cloud/bigquery/storage/v1",
					Keep: []string{
						"google/cloud/bigquery_storage_v1/client.py",
					},
				},
			},
			want: "google-cloud-bigquerystorage:\n    api: google/cloud/bigquery/storage/v1\n    keep:\n        - google/cloud/bigquery_storage_v1/client.py\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := yaml.Marshal(test.entry)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Errorf("got:\n%s\nwant:\n%s", string(got), test.want)
			}
		})
	}
}

// TODO(https://github.com/julieqiu/librarianx/issues/XXX): Re-enable after implementing BUILD.bazel parsing for new config format
// func TestConfig_Add_WithBazelParsing(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	apiPath := "google/cloud/secretmanager/v1"
// 	buildDir := filepath.Join(tmpDir, apiPath)
// 	if err := os.MkdirAll(buildDir, 0755); err != nil {
// 		t.Fatal(err)
// 	}
//
// 	buildContent := `
// py_gapic_library(
//     name = "secretmanager_py_gapic",
//     srcs = [":secretmanager_proto"],
//     grpc_service_config = "secretmanager_grpc_service_config.json",
//     opt_args = [
//         "warehouse-package-name=google-cloud-secret-manager",
//         "python-gapic-namespace=google.cloud",
//     ],
//     rest_numeric_enums = True,
//     service_yaml = "secretmanager_v1.yaml",
//     transport = "grpc+rest",
// )
// `
// 	buildPath := filepath.Join(buildDir, "BUILD.bazel")
// 	if err := os.WriteFile(buildPath, []byte(buildContent), 0644); err != nil {
// 		t.Fatal(err)
// 	}
//
// 	cfg := New("v0.5.0", "python", nil)
// 	if err := cfg.AddLegacy("secretmanager", []string{apiPath}, "", tmpDir); err != nil {
// 		t.Fatal(err)
// 	}
//
// 	if len(cfg.Libraries) != 1 {
// 		t.Fatalf("got %d libraries, want 1", len(cfg.Libraries))
// 	}
// }
