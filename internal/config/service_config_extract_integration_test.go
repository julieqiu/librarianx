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
	"testing"
)

// TestExtractServiceConfigSettings_RealAPI tests extraction with a real googleapis service config.
// This test is skipped if GOOGLEAPIS_ROOT is not set.
func TestExtractServiceConfigSettings_RealAPI(t *testing.T) {
	googleapisRoot := os.Getenv("GOOGLEAPIS_ROOT")
	if googleapisRoot == "" {
		t.Skip("GOOGLEAPIS_ROOT not set, skipping integration test")
	}

	apiPath := "google/cloud/secretmanager/v1"

	t.Run("extract real java settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(googleapisRoot, apiPath, "java")
		if err != nil {
			t.Fatal(err)
		}

		if config != nil {
			t.Logf("Java package: %s", GetJavaPackage(config))
			t.Logf("Java service class names: %v", GetJavaServiceClassNames(config))
		} else {
			t.Log("No Java settings found")
		}
	})

	t.Run("extract real python settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(googleapisRoot, apiPath, "python")
		if err != nil {
			t.Fatal(err)
		}

		if config != nil {
			t.Logf("REST async I/O enabled: %v", GetPythonRestAsyncIOEnabled(config))
		} else {
			t.Log("No Python settings found")
		}
	})

	t.Run("extract real node settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(googleapisRoot, apiPath, "node")
		if err != nil {
			t.Fatal(err)
		}

		if config != nil {
			t.Logf("Selective methods: %v", GetNodeSelectiveMethods(config))
		} else {
			t.Log("No Node settings found")
		}
	})

	t.Run("extract real dotnet settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(googleapisRoot, apiPath, "dotnet")
		if err != nil {
			t.Fatal(err)
		}

		if config != nil {
			t.Logf("Renamed services: %v", GetDotnetRenamedServices(config))
			t.Logf("Renamed resources: %v", GetDotnetRenamedResources(config))
		} else {
			t.Log("No Dotnet settings found")
		}
	})
}
