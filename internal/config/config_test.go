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
			original := &Config{}
			if err := original.Read(test.testdata); err != nil {
				t.Fatal(err)
			}

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "librarian.yaml")
			if err := original.Write(configPath); err != nil {
				t.Fatal(err)
			}

			got := &Config{}
			if err := got.Read(configPath); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(original, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
