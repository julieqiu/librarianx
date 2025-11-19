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

func TestDeriveLibraryName_Channel(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{"google/cloud/secretmanager/v1", "google-cloud-secretmanager-v1"},
		{"google/api/apikeys/v2", "google-api-apikeys-v2"},
		{"grafeas/v1", "grafeas-v1"},
		{"google/cloud/storage/v1", "google-cloud-storage-v1"},
	} {
		got, err := DeriveLibraryName("channel", test.apiPath)
		if err != nil {
			t.Errorf("DeriveLibraryName(%q, %q) error: %v", "channel", test.apiPath, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveLibraryName(%q, %q) = %q, want %q", "channel", test.apiPath, got, test.want)
		}
	}
}

func TestDeriveLibraryName_API(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{"google/cloud/secretmanager/v1", "google-cloud-secretmanager"},
		{"google/cloud/secretmanager/v2", "google-cloud-secretmanager"},
		{"google/api/apikeys/v2", "google-api-apikeys"},
		{"google/cloud/storage/v1", "google-cloud-storage"},
	} {
		got, err := DeriveLibraryName("api", test.apiPath)
		if err != nil {
			t.Errorf("DeriveLibraryName(%q, %q) error: %v", "api", test.apiPath, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveLibraryName(%q, %q) = %q, want %q", "api", test.apiPath, got, test.want)
		}
	}
}

func TestDeriveLibraryName_Invalid(t *testing.T) {
	_, err := DeriveLibraryName("invalid", "google/cloud/secretmanager/v1")
	if err == nil {
		t.Error("DeriveLibraryName with invalid mode should return error")
	}
}

func TestDeriveAPIPath_Channel(t *testing.T) {
	for _, test := range []struct {
		libraryName string
		want        string
	}{
		{"google-cloud-secretmanager-v1", "google/cloud/secretmanager/v1"},
		{"google-api-apikeys-v2", "google/api/apikeys/v2"},
		{"grafeas-v1", "grafeas/v1"},
		{"google-cloud-storage-v1", "google/cloud/storage/v1"},
	} {
		got, err := DeriveAPIPath("channel", test.libraryName)
		if err != nil {
			t.Errorf("DeriveAPIPath(%q, %q) error: %v", "channel", test.libraryName, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveAPIPath(%q, %q) = %q, want %q", "channel", test.libraryName, got, test.want)
		}
	}
}

func TestDeriveAPIPath_API(t *testing.T) {
	for _, test := range []struct {
		libraryName string
		want        string
	}{
		{"google-cloud-secretmanager", "google/cloud/secretmanager"},
		{"google-api-apikeys", "google/api/apikeys"},
		{"google-cloud-storage", "google/cloud/storage"},
	} {
		got, err := DeriveAPIPath("api", test.libraryName)
		if err != nil {
			t.Errorf("DeriveAPIPath(%q, %q) error: %v", "api", test.libraryName, err)
			continue
		}
		if got != test.want {
			t.Errorf("DeriveAPIPath(%q, %q) = %q, want %q", "api", test.libraryName, got, test.want)
		}
	}
}

func TestDeriveAPIPath_Invalid(t *testing.T) {
	_, err := DeriveAPIPath("invalid", "google-cloud-secretmanager-v1")
	if err == nil {
		t.Error("DeriveAPIPath with invalid mode should return error")
	}
}
