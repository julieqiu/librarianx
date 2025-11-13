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
		{"google/ai/generativelanguage/v1", true},
		{"google/ai/generativelanguage/v1beta", false},
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

	// Check first API
	if len(discovered) > 0 {
		want := &DiscoveredAPI{
			Path:         "google/ai/generativelanguage/v1",
			Service:      "generativelanguage",
			Namespace:    "ai",
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
			Path:      "google/ai/generativelanguage/v1",
			Service:   "generativelanguage",
			Namespace: "ai",
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

	// Check ai/generativelanguage group
	genaiAPIs := groups["ai/generativelanguage"]
	if len(genaiAPIs) != 1 {
		t.Errorf("got %d generativelanguage APIs, want 1", len(genaiAPIs))
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
		{Path: "google/ai/generativelanguage/v1"},
	}

	filtered := cfg.FilterDiscoveredAPIs(discovered)

	// Should filter out google/cloud/secretmanager/v1 (explicitly configured)
	if len(filtered) != 2 {
		t.Errorf("got %d filtered APIs, want 2", len(filtered))
	}

	// Verify the filtered APIs
	want := []string{
		"google/cloud/secretmanager/v1beta1",
		"google/ai/generativelanguage/v1",
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

func TestGetLibrariesForGeneration_ServiceLevel(t *testing.T) {
	// Create a temporary googleapis structure
	tmpDir := t.TempDir()

	// Create test API directories
	testAPIs := []string{
		"google/cloud/secretmanager/v1",
		"google/cloud/secretmanager/v1beta1",
		"google/ai/generativelanguage/v1",
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

	// Should have 2 service groups: cloud/secretmanager and ai/generativelanguage
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
		}
		if lib.Name == "google-cloud-secretmanager-v1beta1" {
			hasV1beta1 = true
		}
	}

	if !hasV1 || !hasV1beta1 {
		t.Errorf("missing expected versioned library names; hasV1=%v, hasV1beta1=%v", hasV1, hasV1beta1)
	}
}
