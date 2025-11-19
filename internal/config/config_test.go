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

			wantCfg, err := Read(test.filePath)
			if err != nil {
				t.Fatal(err)
			}
			var gotCfg Config
			if err := yaml.Unmarshal(got.Bytes(), &gotCfg); err != nil {
				t.Fatalf("failed to unmarshal generated YAML: %v", err)
			}

			if diff := cmp.Diff(wantCfg, &gotCfg); diff != "" {
				t.Errorf("mismatch(-want, +got): %s", diff)
			}
		})
	}
}
