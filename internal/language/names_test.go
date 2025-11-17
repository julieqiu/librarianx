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

import "testing"

func TestDeriveLibraryName_Version(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{"google/cloud/secretmanager/v1", "google-cloud-secretmanager-v1"},
		{"google/api/apikeys/v2", "google-api-apikeys-v2"},
		{"grafeas/v1", "grafeas-v1"},
		{"google/cloud/storage/v1", "google-cloud-storage-v1"},
	} {
		got, err := DeriveLibraryName("version", test.apiPath)
		if err != nil {
			t.Errorf("DeriveLibraryName(%q, %q) error: %v", "version", test.apiPath, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveLibraryName(%q, %q) = %q, want %q", "version", test.apiPath, got, test.want)
		}
	}
}

func TestDeriveLibraryName_Service(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{"google/cloud/secretmanager/v1", "google-cloud-secretmanager"},
		{"google/cloud/secretmanager/v2", "google-cloud-secretmanager"},
		{"google/api/apikeys/v2", "google-api-apikeys"},
		{"google/cloud/storage/v1", "google-cloud-storage"},
	} {
		got, err := DeriveLibraryName("service", test.apiPath)
		if err != nil {
			t.Errorf("DeriveLibraryName(%q, %q) error: %v", "service", test.apiPath, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveLibraryName(%q, %q) = %q, want %q", "service", test.apiPath, got, test.want)
		}
	}
}

func TestDeriveLibraryName_Invalid(t *testing.T) {
	_, err := DeriveLibraryName("invalid", "google/cloud/secretmanager/v1")
	if err == nil {
		t.Error("DeriveLibraryName with invalid mode should return error")
	}
}

func TestDeriveAPIPath_Version(t *testing.T) {
	for _, test := range []struct {
		libraryName string
		want        string
	}{
		{"google-cloud-secretmanager-v1", "google/cloud/secretmanager/v1"},
		{"google-api-apikeys-v2", "google/api/apikeys/v2"},
		{"grafeas-v1", "grafeas/v1"},
		{"google-cloud-storage-v1", "google/cloud/storage/v1"},
	} {
		got, err := DeriveAPIPath("version", test.libraryName)
		if err != nil {
			t.Errorf("DeriveAPIPath(%q, %q) error: %v", "version", test.libraryName, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveAPIPath(%q, %q) = %q, want %q", "version", test.libraryName, got, test.want)
		}
	}
}

func TestDeriveAPIPath_Service(t *testing.T) {
	for _, test := range []struct {
		libraryName string
		want        string
	}{
		{"google-cloud-secretmanager", "google/cloud/secretmanager"},
		{"google-api-apikeys", "google/api/apikeys"},
		{"google-cloud-storage", "google/cloud/storage"},
	} {
		got, err := DeriveAPIPath("service", test.libraryName)
		if err != nil {
			t.Errorf("DeriveAPIPath(%q, %q) error: %v", "service", test.libraryName, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveAPIPath(%q, %q) = %q, want %q", "service", test.libraryName, got, test.want)
		}
	}
}

func TestDeriveAPIPath_Invalid(t *testing.T) {
	_, err := DeriveAPIPath("invalid", "google-cloud-secretmanager-v1")
	if err == nil {
		t.Error("DeriveAPIPath with invalid mode should return error")
	}
}
