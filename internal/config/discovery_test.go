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

func TestDiscoverAPIs(t *testing.T) {
	// Create a temporary googleapis structure
	tmpDir := t.TempDir()

	// Create some test API directories
	testAPIs := []struct {
		path     string
		hasBuild bool
	}{
		{"google/cloud/secretmanager/v1", true},
		{"google/cloud/secretmanager/v1beta1", true},
		{"google/cloud/vision/v1", true},
		{"google/cloud/vision/v2", false},
		{"google/type", false}, // No version directory
	}

	for _, api := range testAPIs {
		apiDir := filepath.Join(tmpDir, api.path)
		if err := os.MkdirAll(apiDir, 0755); err != nil {
			t.Fatal(err)
		}

		if api.hasBuild {
			buildFile := filepath.Join(apiDir, "BUILD.bazel")
			if err := os.WriteFile(buildFile, []byte("# BUILD file"), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Discover APIs
	discovered, err := DiscoverAPIs(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should find 4 APIs (excluding google/type since it has no version)
	if len(discovered) != 4 {
		t.Errorf("got %d APIs, want 4", len(discovered))
	}

	// Check first API (alphabetically)
	if len(discovered) > 0 {
		want := &DiscoveredAPI{
			Path:         "google/cloud/secretmanager/v1",
			Service:      "secretmanager",
			Namespace:    "cloud",
			Version:      "v1",
			HasBuildFile: true,
		}
		if diff := cmp.Diff(want, discovered[0]); diff != "" {
			t.Errorf("first API mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestGroupByService(t *testing.T) {
	apis := []*DiscoveredAPI{
		{
			Path:      "google/cloud/secretmanager/v1",
			Service:   "secretmanager",
			Namespace: "cloud",
			Version:   "v1",
		},
		{
			Path:      "google/cloud/secretmanager/v1beta1",
			Service:   "secretmanager",
			Namespace: "cloud",
			Version:   "v1beta1",
		},
		{
			Path:      "google/cloud/vision/v1",
			Service:   "vision",
			Namespace: "cloud",
			Version:   "v1",
		},
	}

	groups := GroupByService(apis)

	// Should have 2 service groups
	if len(groups) != 2 {
		t.Errorf("got %d groups, want 2", len(groups))
	}

	// Check cloud/secretmanager group
	secretmanagerAPIs := groups["cloud/secretmanager"]
	if len(secretmanagerAPIs) != 2 {
		t.Errorf("got %d secretmanager APIs, want 2", len(secretmanagerAPIs))
	}

	// Check cloud/vision group
	visionAPIs := groups["cloud/vision"]
	if len(visionAPIs) != 1 {
		t.Errorf("got %d vision APIs, want 1", len(visionAPIs))
	}
}

func TestFilterDiscoveredAPIs(t *testing.T) {
	cfg := &Config{
		Libraries: []LibraryEntry{
			{Name: "*"},
			{
				Name: "secretmanager",
				Config: &LibraryConfig{
					API: "google/cloud/secretmanager/v1",
				},
			},
		},
	}

	discovered := []*DiscoveredAPI{
		{Path: "google/cloud/secretmanager/v1"},
		{Path: "google/cloud/secretmanager/v1beta1"},
		{Path: "google/cloud/vision/v1"},
	}

	filtered := cfg.FilterDiscoveredAPIs(discovered)

	// Should filter out google/cloud/secretmanager/v1 (explicitly configured)
	if len(filtered) != 2 {
		t.Errorf("got %d filtered APIs, want 2", len(filtered))
	}

	// Verify the filtered APIs
	want := []string{
		"google/cloud/secretmanager/v1beta1",
		"google/cloud/vision/v1",
	}

	for i, api := range filtered {
		if i >= len(want) {
			break
		}
		if api.Path != want[i] {
			t.Errorf("filtered[%d].Path = %q, want %q", i, api.Path, want[i])
		}
	}
}

func TestFilterDiscoveredAPIs_AutoDiscoverDisabled(t *testing.T) {
	cfg := &Config{
		Libraries: []LibraryEntry{
			{
				Name: "secretmanager",
				Config: &LibraryConfig{
					API: "google/cloud/secretmanager/v1",
				},
			},
		},
	}

	discovered := []*DiscoveredAPI{
		{Path: "google/cloud/secretmanager/v1beta1"},
	}

	filtered := cfg.FilterDiscoveredAPIs(discovered)

	// Should return nil when auto-discover is disabled
	if filtered != nil {
		t.Errorf("got %v, want nil when auto-discover is disabled", filtered)
	}
}

func TestFilterDiscoveredAPIs_WithExcludePatterns(t *testing.T) {
	cfg := &Config{
		Defaults: &Defaults{
			ExcludeAPIs: []string{
				"google/ads/*",
				"google/actions/*",
			},
		},
		Libraries: []LibraryEntry{
			{Name: "*"},
		},
	}

	discovered := []*DiscoveredAPI{
		{Path: "google/cloud/secretmanager/v1"},
		{Path: "google/ads/admanager/v1"},
		{Path: "google/ads/googleads/v19"},
		{Path: "google/actions/sdk/v2"},
		{Path: "google/api/apikeys/v2"},
	}

	filtered := cfg.FilterDiscoveredAPIs(discovered)

	// Should filter out google/ads/* and google/actions/*
	want := []string{
		"google/cloud/secretmanager/v1",
		"google/api/apikeys/v2",
	}

	if len(filtered) != len(want) {
		t.Errorf("got %d filtered APIs, want %d", len(filtered), len(want))
	}

	for i, api := range filtered {
		if i >= len(want) {
			break
		}
		if api.Path != want[i] {
			t.Errorf("filtered[%d].Path = %q, want %q", i, api.Path, want[i])
		}
	}
}

func TestMatchesPattern(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			path:    "google/cloud/secretmanager/v1",
			pattern: "google/cloud/secretmanager/v1",
			want:    true,
		},
		{
			name:    "exact no match",
			path:    "google/cloud/secretmanager/v1",
			pattern: "google/cloud/secretmanager/v2",
			want:    false,
		},
		{
			name:    "suffix wildcard match",
			path:    "google/ads/admanager/v1",
			pattern: "google/ads/*",
			want:    true,
		},
		{
			name:    "suffix wildcard no match",
			path:    "google/cloud/secretmanager/v1",
			pattern: "google/ads/*",
			want:    false,
		},
		{
			name:    "prefix wildcard match",
			path:    "google/cloud/secretmanager/v1",
			pattern: "*/v1",
			want:    true,
		},
		{
			name:    "prefix wildcard no match",
			path:    "google/cloud/secretmanager/v1",
			pattern: "*/v2",
			want:    false,
		},
		{
			name:    "middle wildcard match",
			path:    "google/cloud/secretmanager/v1",
			pattern: "google/*/v1",
			want:    true,
		},
		{
			name:    "middle wildcard no match",
			path:    "google/cloud/secretmanager/v1",
			pattern: "google/*/v2",
			want:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := matchesPattern(test.path, test.pattern)
			if got != test.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v",
					test.path, test.pattern, got, test.want)
			}
		})
	}
}

func TestGetLibrariesForGeneration_ServiceLevel(t *testing.T) {
	// Create a temporary googleapis structure
	tmpDir := t.TempDir()

	// Create test API directories
	testAPIs := []string{
		"google/cloud/secretmanager/v1",
		"google/cloud/secretmanager/v1beta1",
		"google/cloud/vision/v1",
	}

	for _, apiPath := range testAPIs {
		apiDir := filepath.Join(tmpDir, apiPath)
		if err := os.MkdirAll(apiDir, 0755); err != nil {
			t.Fatal(err)
		}
		buildFile := filepath.Join(apiDir, "BUILD.bazel")
		if err := os.WriteFile(buildFile, []byte("# BUILD file"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &Config{
		Language: "python",
		Defaults: &Defaults{
			OneLibraryPer: "service",
		},
		Libraries: []LibraryEntry{
			{Name: "*"},
		},
	}

	libraries, err := cfg.GetLibrariesForGeneration(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 2 service groups: cloud/secretmanager and cloud/vision
	if len(libraries) != 2 {
		t.Errorf("got %d libraries, want 2", len(libraries))
	}

	// Find secretmanager library
	var secretmanagerLib *Library
	for _, lib := range libraries {
		if lib.Name == "google-cloud-secretmanager" {
			secretmanagerLib = lib
			break
		}
	}

	if secretmanagerLib == nil {
		t.Fatal("google-cloud-secretmanager library not found")
	}

	// Should contain both v1 and v1beta1
	if len(secretmanagerLib.Apis) != 2 {
		t.Errorf("got %d APIs in secretmanager, want 2", len(secretmanagerLib.Apis))
	}
}

func TestGetLibrariesForGeneration_VersionLevel(t *testing.T) {
	// Create a temporary googleapis structure
	tmpDir := t.TempDir()

	// Create test API directories
	testAPIs := []string{
		"google/cloud/secretmanager/v1",
		"google/cloud/secretmanager/v1beta1",
	}

	for _, apiPath := range testAPIs {
		apiDir := filepath.Join(tmpDir, apiPath)
		if err := os.MkdirAll(apiDir, 0755); err != nil {
			t.Fatal(err)
		}
		buildFile := filepath.Join(apiDir, "BUILD.bazel")
		if err := os.WriteFile(buildFile, []byte("# BUILD file"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &Config{
		Language: "rust",
		Defaults: &Defaults{
			OneLibraryPer: "version",
			Output:        "src/generated/",
		},
		Libraries: []LibraryEntry{
			{Name: "*"},
		},
	}

	libraries, err := cfg.GetLibrariesForGeneration(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 2 separate libraries (one per version)
	if len(libraries) != 2 {
		t.Errorf("got %d libraries, want 2", len(libraries))
	}

	// Each library should have exactly 1 API
	for _, lib := range libraries {
		if len(lib.Apis) != 1 {
			t.Errorf("library %s has %d APIs, want 1", lib.Name, len(lib.Apis))
		}
	}

	// Check library names include version
	hasV1 := false
	hasV1beta1 := false
	for _, lib := range libraries {
		if lib.Name == "google-cloud-secretmanager-v1" {
			hasV1 = true
			// Check that location is set correctly for Rust
			if lib.Location != "src/generated/cloud/secretmanager/v1/" {
				t.Errorf("library location = %q, want %q", lib.Location, "src/generated/cloud/secretmanager/v1/")
			}
		}
		if lib.Name == "google-cloud-secretmanager-v1beta1" {
			hasV1beta1 = true
			if lib.Location != "src/generated/cloud/secretmanager/v1beta1/" {
				t.Errorf("library location = %q, want %q", lib.Location, "src/generated/cloud/secretmanager/v1beta1/")
			}
		}
	}

	if !hasV1 || !hasV1beta1 {
		t.Errorf("missing expected versioned library names; hasV1=%v, hasV1beta1=%v", hasV1, hasV1beta1)
	}
}

func TestDeriveLibraryPath(t *testing.T) {
	for _, test := range []struct {
		name     string
		apiPath  string
		language string
		output   string
		want     string
	}{
		{
			name:     "rust with output",
			apiPath:  "google/cloud/bigquery/v2",
			language: "rust",
			output:   "src/generated/",
			want:     "src/generated/cloud/bigquery/v2/",
		},
		{
			name:     "rust without trailing slash",
			apiPath:  "google/cloud/secretmanager/v1",
			language: "rust",
			output:   "src/generated",
			want:     "src/generated/cloud/secretmanager/v1/",
		},
		{
			name:     "rust api namespace",
			apiPath:  "google/api/apikeys/v2",
			language: "rust",
			output:   "src/generated/",
			want:     "src/generated/api/apikeys/v2/",
		},
		{
			name:     "rust no namespace",
			apiPath:  "google/type",
			language: "rust",
			output:   "src/generated/",
			want:     "src/generated/type/",
		},
		{
			name:     "python returns empty",
			apiPath:  "google/cloud/bigquery/v2",
			language: "python",
			output:   "packages/",
			want:     "",
		},
		{
			name:     "go returns empty",
			apiPath:  "google/cloud/bigquery/v2",
			language: "go",
			output:   "./",
			want:     "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveLibraryPath(test.apiPath, test.language, test.output)
			if got != test.want {
				t.Errorf("deriveLibraryPath(%q, %q, %q) = %q, want %q",
					test.apiPath, test.language, test.output, got, test.want)
			}
		})
	}
}
