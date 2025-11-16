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
	overrides := &ServiceConfigOverrides{
		ExcludeAPIs: []string{
			"google/actions",
			"google/ads",
			"google/cloud/batch",
		},
	}

	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{"excluded nested subdirectory", "google/actions/sdk/v2", true},
		{"excluded direct subdirectory", "google/actions/v2", true},
		{"excluded deeply nested", "google/ads/googleads/v1/services", true},
		{"excluded batch subdirectory", "google/cloud/batch/v1", true},
		{"not excluded different path", "google/cloud/secretmanager/v1", false},
		{"not excluded similar prefix", "google/action/v1", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := overrides.IsExcluded(test.path)
			if got != test.want {
				t.Errorf("IsExcluded(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestServiceConfigOverrides_EmbeddedExclusions(t *testing.T) {
	overrides, err := ReadServiceConfigOverrides()
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{"actions excluded", "google/actions/sdk/v2", true},
		{"apigeeregistry excluded", "google/cloud/apigeeregistry/v1", true},
		{"videointelligence v1 excluded", "google/cloud/videointelligence/v1", true},
		{"videointelligence v1beta2 excluded", "google/cloud/videointelligence/v1beta2", true},
		{"videointelligence v1p1beta1 excluded", "google/cloud/videointelligence/v1p1beta1", true},
		{"videointelligence v1p2beta1 excluded", "google/cloud/videointelligence/v1p2beta1", true},
		{"videointelligence v1p3beta1 excluded", "google/cloud/videointelligence/v1p3beta1", true},
		{"vision v1 excluded", "google/cloud/vision/v1", true},
		{"vision v1p1beta1 excluded", "google/cloud/vision/v1p1beta1", true},
		{"vision v1p2beta1 excluded", "google/cloud/vision/v1p2beta1", true},
		{"vision v1p3beta1 excluded", "google/cloud/vision/v1p3beta1", true},
		{"vision v1p4beta1 excluded", "google/cloud/vision/v1p4beta1", true},
		{"visionai v1 excluded", "google/cloud/visionai/v1", true},
		{"visionai v1alpha1 excluded", "google/cloud/visionai/v1alpha1", true},
		{"firestore v1 excluded", "google/firestore/v1", true},
		{"genomics excluded", "google/genomics/v1", true},
		{"geo/type excluded", "google/geo/type", true},
		{"home excluded", "google/home/v1", true},
		{"networking/trafficdirector/type excluded", "google/networking/trafficdirector/type", true},
		{"maps excluded", "google/maps/v1", true},
		{"marketingplatform excluded", "google/marketingplatform/admin/v1alpha", true},
		{"partner excluded", "google/partner/aistreams/v1alpha1", true},
		{"search excluded", "google/search/partnerdataingestion/logging/v1", true},
		{"security excluded", "google/security/v1", true},
		{"shopping excluded", "google/shopping/v1", true},
		{"streetview excluded", "google/streetview/publish/v1", true},
		{"watcher excluded", "google/watcher/v1", true},
		{"remoteworkers excluded", "google/devtools/remoteworkers/v1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := overrides.IsExcluded(test.path)
			if got != test.want {
				t.Errorf("IsExcluded(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}
