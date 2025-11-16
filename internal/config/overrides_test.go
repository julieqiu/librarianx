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

import "testing"

func TestMatchGlobPattern(t *testing.T) {
	for _, test := range []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		// Directory prefix matching
		{"prefix matches subdirectory", "google/actions", "google/actions/v2", true},
		{"prefix matches nested subdirectory", "google/actions", "google/actions/sdk/v2", true},
		{"prefix matches deeply nested", "google/actions", "google/actions/sdk/v2/foo", true},
		{"prefix no match different path", "google/actions", "google/cloud/v1", false},
		{"prefix matches any subdirectory", "google/actions", "google/actions/test", true},
		{"prefix matches exact path", "google/actions", "google/actions", true},
		{"prefix does not match partial", "google/actions", "google/action", false},
		{"prefix does not match similar", "google/actions", "google/actionssdk", false},

		// Exact matches
		{"exact match full path", "google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1", true},
		{"exact no match different version", "google/cloud/secretmanager/v1", "google/cloud/secretmanager/v2", false},

		// Trailing slash (backwards compatibility)
		{"trailing slash stripped and matches", "google/ads/", "google/ads/v1", true},
		{"trailing slash exact match", "google/ads/", "google/ads", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := matchGlobPattern(test.pattern, test.path)
			if got != test.want {
				t.Errorf("matchGlobPattern(%q, %q) = %v, want %v", test.pattern, test.path, got, test.want)
			}
		})
	}
}

func TestServiceConfigOverrides_IsExcluded(t *testing.T) {
	overrides := &ServiceConfigOverrides{}
	overrides.ExcludedAPIs.All = []string{"google/firebase"}
	overrides.ExcludedAPIs.Rust = []string{
		"google/actions",
		"google/ads",
		"google/cloud/batch",
	}

	for _, test := range []struct {
		name     string
		language string
		path     string
		want     bool
	}{
		{"rust excluded nested subdirectory", "rust", "google/actions/sdk/v2", true},
		{"rust excluded direct subdirectory", "rust", "google/actions/v2", true},
		{"rust excluded deeply nested", "rust", "google/ads/googleads/v1/services", true},
		{"rust excluded batch subdirectory", "rust", "google/cloud/batch/v1", true},
		{"rust not excluded different path", "rust", "google/cloud/secretmanager/v1", false},
		{"rust not excluded similar prefix", "rust", "google/action/v1", false},
		{"python not excluded even if in rust list", "python", "google/actions/sdk/v2", false},
		{"python not excluded batch", "python", "google/cloud/batch/v1", false},
		{"all languages excluded firebase", "rust", "google/firebase/v1", true},
		{"python also excluded firebase", "python", "google/firebase/v1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := overrides.IsExcluded(test.language, test.path)
			if got != test.want {
				t.Errorf("IsExcluded(%q, %q) = %v, want %v", test.language, test.path, got, test.want)
			}
		})
	}
}

func TestServiceConfigOverrides_EmbeddedExclusions(t *testing.T) {
	overrides, err := ReadServiceConfigOverrides()
	if err != nil {
		t.Fatal(err)
	}

	// Test Rust exclusions
	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{"actions excluded for rust", "google/actions/sdk/v2", true},
		{"apigeeregistry excluded for rust", "google/cloud/apigeeregistry/v1", true},
		{"videointelligence v1 excluded for rust", "google/cloud/videointelligence/v1", true},
		{"videointelligence v1beta2 excluded for rust", "google/cloud/videointelligence/v1beta2", true},
		{"videointelligence v1p1beta1 excluded for rust", "google/cloud/videointelligence/v1p1beta1", true},
		{"videointelligence v1p2beta1 excluded for rust", "google/cloud/videointelligence/v1p2beta1", true},
		{"videointelligence v1p3beta1 excluded for rust", "google/cloud/videointelligence/v1p3beta1", true},
		{"vision v1 excluded for rust", "google/cloud/vision/v1", true},
		{"vision v1p1beta1 excluded for rust", "google/cloud/vision/v1p1beta1", true},
		{"vision v1p2beta1 excluded for rust", "google/cloud/vision/v1p2beta1", true},
		{"vision v1p3beta1 excluded for rust", "google/cloud/vision/v1p3beta1", true},
		{"vision v1p4beta1 excluded for rust", "google/cloud/vision/v1p4beta1", true},
		{"visionai v1 excluded for rust", "google/cloud/visionai/v1", true},
		{"visionai v1alpha1 excluded for rust", "google/cloud/visionai/v1alpha1", true},
		{"firestore v1 excluded for rust", "google/firestore/v1", true},
		{"genomics excluded for rust", "google/genomics/v1", true},
		{"geo/type excluded for rust", "google/geo/type", true},
		{"home excluded for rust", "google/home/v1", true},
		{"networking/trafficdirector/type excluded for rust", "google/networking/trafficdirector/type", true},
		{"maps excluded for rust", "google/maps/v1", true},
		{"marketingplatform excluded for rust", "google/marketingplatform/admin/v1alpha", true},
		{"partner excluded for rust", "google/partner/aistreams/v1alpha1", true},
		{"search excluded for rust", "google/search/partnerdataingestion/logging/v1", true},
		{"security excluded for rust", "google/security/v1", true},
		{"shopping excluded for rust", "google/shopping/v1", true},
		{"streetview excluded for rust", "google/streetview/publish/v1", true},
		{"watcher excluded for rust", "google/watcher/v1", true},
		{"remoteworkers excluded for rust", "google/devtools/remoteworkers/v1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := overrides.IsExcluded("rust", test.path)
			if got != test.want {
				t.Errorf("IsExcluded(\"rust\", %q) = %v, want %v", test.path, got, test.want)
			}
		})
	}

	// Test that Python does not have these exclusions
	t.Run("python not excluded", func(t *testing.T) {
		pythonNotExcluded := []string{
			"google/actions/sdk/v2",
			"google/cloud/videointelligence/v1",
			"google/firestore/v1",
		}
		for _, path := range pythonNotExcluded {
			if overrides.IsExcluded("python", path) {
				t.Errorf("IsExcluded(\"python\", %q) = true, want false (should not be excluded for Python)", path)
			}
		}
	})
}
