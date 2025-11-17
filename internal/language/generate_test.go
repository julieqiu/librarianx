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

package language

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/julieqiu/librarianx/internal/config"
)

func TestGetDefaultAPI(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want string
	}{
		{
			name: "single API",
			lib:  &config.Library{API: "google/cloud/secretmanager/v1"},
			want: "google/cloud/secretmanager/v1",
		},
		{
			name: "multiple APIs - highest stable",
			lib: &config.Library{
				APIs: []string{
					"google/cloud/secretmanager/v1",
					"google/cloud/secretmanager/v2",
					"google/cloud/secretmanager/v1beta1",
				},
			},
			want: "google/cloud/secretmanager/v2",
		},
		{
			name: "only beta versions",
			lib: &config.Library{
				APIs: []string{
					"google/cloud/secretmanager/v1beta1",
					"google/cloud/secretmanager/v2beta1",
				},
			},
			want: "google/cloud/secretmanager/v2beta1",
		},
		{
			name: "stable before beta",
			lib: &config.Library{
				APIs: []string{
					"google/cloud/secretmanager/v2beta1",
					"google/cloud/secretmanager/v1",
				},
			},
			want: "google/cloud/secretmanager/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getDefaultAPI(test.lib)
			if got != test.want {
				t.Errorf("getDefaultAPI() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSortAPIsByVersion(t *testing.T) {
	for _, test := range []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "stable versions descending",
			input: []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v2"},
			want:  []string{"google/cloud/secretmanager/v2", "google/cloud/secretmanager/v1"},
		},
		{
			name:  "stable before beta",
			input: []string{"google/cloud/secretmanager/v1beta1", "google/cloud/secretmanager/v1"},
			want:  []string{"google/cloud/secretmanager/v1", "google/cloud/secretmanager/v1beta1"},
		},
		{
			name:  "beta before alpha",
			input: []string{"google/cloud/secretmanager/v1alpha1", "google/cloud/secretmanager/v1beta1"},
			want:  []string{"google/cloud/secretmanager/v1beta1", "google/cloud/secretmanager/v1alpha1"},
		},
		{
			name: "complex ordering",
			input: []string{
				"google/cloud/secretmanager/v1beta1",
				"google/cloud/secretmanager/v2",
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v2beta1",
			},
			want: []string{
				"google/cloud/secretmanager/v2",
				"google/cloud/secretmanager/v1",
				"google/cloud/secretmanager/v2beta1",
				"google/cloud/secretmanager/v1beta1",
			},
		},
		{
			name: "all types",
			input: []string{
				"google/cloud/foo/v1alpha1",
				"google/cloud/foo/v2beta1",
				"google/cloud/foo/v1",
				"google/cloud/foo/v2",
				"google/cloud/foo/v1beta1",
				"google/cloud/foo/v2alpha1",
			},
			want: []string{
				"google/cloud/foo/v2",
				"google/cloud/foo/v1",
				"google/cloud/foo/v2beta1",
				"google/cloud/foo/v1beta1",
				"google/cloud/foo/v2alpha1",
				"google/cloud/foo/v1alpha1",
			},
		},
		{
			name: "beta versions with numbers",
			input: []string{
				"google/cloud/foo/v1beta1",
				"google/cloud/foo/v1beta2",
			},
			want: []string{
				"google/cloud/foo/v1beta2",
				"google/cloud/foo/v1beta1",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := sortAPIsByVersion(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
