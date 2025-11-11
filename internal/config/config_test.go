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
				"generate.output":    "packages/",
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
				"generate.output": "generated/",
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
	if err := cfg.Set("generate.output", "packages/"); err != nil {
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
