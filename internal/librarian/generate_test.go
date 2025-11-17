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
	"context"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateLibraryForAPI_Disabled(t *testing.T) {
	cfg := &config.Config{
		Default: &config.Default{
			Output: "output/",
			Generate: &config.DefaultGenerate{
				All: true,
			},
		},
		Libraries: []*config.Library{
			{
				Name: "test-library",
				API:  "google/test/v1",
				Generate: &config.LibraryGenerate{
					Disabled: true,
				},
			},
		},
	}

	// This should return nil without error, skipping generation
	err := generateLibraryForAPI(context.Background(), cfg, "/fake/googleapis", "google/test/v1", "/fake/service.yaml", false)
	if err != nil {
		t.Errorf("generateLibraryForAPI with disabled library should return nil, got error: %v", err)
	}
}

func TestGenerateLibraryForAPI_NameOverride(t *testing.T) {
	cfg := &config.Config{
		Default: &config.Default{
			Output: "output/",
			Generate: &config.DefaultGenerate{
				All: true,
			},
		},
		NameOverrides: map[string]string{
			"google/api/apikeys/v2": "google-cloud-apikeys-v2",
		},
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-apikeys-v2",
				Version: "1.1.1",
				Generate: &config.LibraryGenerate{
					Disabled: true,
				},
			},
		},
	}

	// Should find library by name override and skip because it's disabled
	err := generateLibraryForAPI(context.Background(), cfg, "/fake/googleapis", "google/api/apikeys/v2", "/fake/service.yaml", false)
	if err != nil {
		t.Errorf("generateLibraryForAPI should handle name override, got error: %v", err)
	}
}

func TestGenerateLibraryForAPI_DerivedName(t *testing.T) {
	cfg := &config.Config{
		Default: &config.Default{
			Output: "output/",
			Generate: &config.DefaultGenerate{
				All: true,
			},
		},
		Libraries: []*config.Library{
			{
				Name:    "google-api-cloudquotas-v1",
				Version: "1.1.0",
				Generate: &config.LibraryGenerate{
					Disabled: true,
				},
			},
		},
	}

	// Should find library by derived name (google/api/cloudquotas/v1 -> google-api-cloudquotas-v1)
	err := generateLibraryForAPI(context.Background(), cfg, "/fake/googleapis", "google/api/cloudquotas/v1", "/fake/service.yaml", false)
	if err != nil {
		t.Errorf("generateLibraryForAPI should match by derived name, got error: %v", err)
	}
}

func TestDeriveLibraryName(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{"google/api/cloudquotas/v1", "google-api-cloudquotas-v1"},
		{"google/cloud/asset/v1", "google-cloud-asset-v1"},
		{"google/api/apikeys/v2", "google-api-apikeys-v2"},
	} {
		t.Run(test.apiPath, func(t *testing.T) {
			got := deriveLibraryName(test.apiPath)
			if got != test.want {
				t.Errorf("deriveLibraryName(%q) = %q, want %q", test.apiPath, got, test.want)
			}
		})
	}
}
