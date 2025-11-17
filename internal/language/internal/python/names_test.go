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

package python

import "testing"

func TestDeriveLibraryName(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{"google/cloud/secretmanager/v1", "google-cloud-secretmanager"},
		{"google/cloud/secretmanager/v2", "google-cloud-secretmanager"},
		{"google/api/apikeys/v2", "google-api-apikeys"},
		{"google/cloud/storage/v1", "google-cloud-storage"},
	} {
		got := DeriveLibraryName(test.apiPath)
		if got != test.want {
			t.Errorf("DeriveLibraryName(%q) = %q, want %q", test.apiPath, got, test.want)
		}
	}
}

func TestDeriveAPIPath(t *testing.T) {
	for _, test := range []struct {
		libraryName string
		want        string
	}{
		{"google-cloud-secretmanager", "google/cloud/secretmanager"},
		{"google-api-apikeys", "google/api/apikeys"},
		{"google-cloud-storage", "google/cloud/storage"},
	} {
		got := DeriveAPIPath(test.libraryName)
		if got != test.want {
			t.Errorf("DeriveAPIPath(%q) = %q, want %q", test.libraryName, got, test.want)
		}
	}
}
