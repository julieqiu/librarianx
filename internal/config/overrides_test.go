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
		// Trailing /* patterns (prefix matching)
		{"trailing slash star matches exact", "google/actions/*", "google/actions/v2", true},
		{"trailing slash star matches nested", "google/actions/*", "google/actions/sdk/v2", true},
		{"trailing slash star matches deeply nested", "google/actions/*", "google/actions/sdk/v2/foo", true},
		{"trailing slash star no match", "google/actions/*", "google/cloud/v1", false},
		{"trailing slash star matches prefix exactly", "google/actions/*", "google/actions/test", true},
		{"trailing slash star does not match parent", "google/actions/*", "google/actions", false},
		{"trailing slash star does not match partial", "google/actions/*", "google/action", false},

		// Exact matches
		{"exact match", "google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1", true},
		{"exact no match", "google/cloud/secretmanager/v1", "google/cloud/secretmanager/v2", false},

		// Other glob patterns
		{"glob pattern match", "google/*/v1", "google/cloud/v1", true},
		{"glob pattern no match", "google/*/v1", "google/cloud/secretmanager/v1", false},
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
			"google/actions/*",
			"google/ads/*",
			"google/cloud/batch/*",
		},
	}

	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{"excluded by trailing slash star", "google/actions/sdk/v2", true},
		{"excluded exact prefix", "google/actions/v2", true},
		{"excluded deeply nested", "google/ads/googleads/v1/services", true},
		{"excluded batch", "google/cloud/batch/v1", true},
		{"not excluded", "google/cloud/secretmanager/v1", false},
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
