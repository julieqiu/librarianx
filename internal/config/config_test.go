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
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.yaml.in/yaml/v4"
)

func TestReadWrite(t *testing.T) {
	for _, test := range []struct {
		name     string
		filePath string
		language string
	}{
		{
			name:     "dart",
			filePath: "../../testdata/dart/librarian.yaml",
			language: "dart",
		},
		{
			name:     "go",
			filePath: "../../testdata/go/librarian.yaml",
			language: "go",
		},
		{
			name:     "python",
			filePath: "../../testdata/python/librarian.yaml",
			language: "python",
		},
		{
			name:     "rust",
			filePath: "../../testdata/rust/librarian.yaml",
			language: "rust",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Read(test.filePath)
			if err != nil {
				t.Fatal(err)
			}
			var got bytes.Buffer
			enc := yaml.NewEncoder(&got)
			enc.SetIndent(2)
			if err := enc.Encode(cfg); err != nil {
				t.Fatalf("failed to marshal struct to YAML: %v", err)
			}

			var index int
			data, err := os.ReadFile(test.filePath)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, "#") {
					// Skip the header, and the new lines after the header
					index = i + 2
					continue
				}
			}

			want := strings.Join(lines[index:], "\n")
			if diff := cmp.Diff(want, got.String()); diff != "" {
				t.Errorf("mismatch(-want, +got): %s", diff)
			}
		})
	}
}

func TestConfig_GetNameOverride(t *testing.T) {
	cfg := &Config{
		NameOverrides: map[string]string{
			"google/api/apikeys/v2":            "google-api-keys",
			"google/cloud/bigquery/storage/v1": "google-cloud-bigquery-storage",
		},
	}

	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{"found first override", "google/api/apikeys/v2", "google-api-keys"},
		{"found second override", "google/cloud/bigquery/storage/v1", "google-cloud-bigquery-storage"},
		{"not found", "google/cloud/secretmanager/v1", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cfg.GetNameOverride(test.apiPath)
			if got != test.want {
				t.Errorf("GetNameOverride(%q) = %q, want %q", test.apiPath, got, test.want)
			}
		})
	}
}
