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

func TestExtractServiceConfigSettings(t *testing.T) {
	// Create a temporary googleapis directory structure
	tmpDir := t.TempDir()

	// Create a test service config YAML
	serviceConfigYAML := `
name: secretmanager.googleapis.com
title: Secret Manager API
publishing:
  library_settings:
  - version: google.cloud.secretmanager.v1
    launch_stage: GA
    java_settings:
      library_package: com.google.cloud.secretmanager.v1
      service_class_names:
        google.cloud.secretmanager.v1.SecretManagerService: SecretManager
    python_settings:
      experimental_features:
        rest_async_io_enabled: true
    node_settings:
      common:
        selective_gapic_generation:
          methods:
          - google.cloud.secretmanager.v1.SecretManagerService.GetSecret
          - google.cloud.secretmanager.v1.SecretManagerService.CreateSecret
    dotnet_settings:
      renamed_services:
        SecretManagerService: SecretManagerClient
      renamed_resources:
        secretmanager.googleapis.com/Secret: SecretManagerSecret
`

	// Create directory structure
	apiDir := filepath.Join(tmpDir, "google/cloud/secretmanager/v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write service config file
	configPath := filepath.Join(apiDir, "secretmanager_v1.yaml")
	if err := os.WriteFile(configPath, []byte(serviceConfigYAML), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("extract java settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(tmpDir, "google/cloud/secretmanager/v1", "java")
		if err != nil {
			t.Fatal(err)
		}

		if config == nil {
			t.Fatal("expected config, got nil")
		}

		if config.Java == nil {
			t.Fatal("expected Java settings, got nil")
		}

		wantPackage := "com.google.cloud.secretmanager.v1"
		if config.Java.Package != wantPackage {
			t.Errorf("Java.Package = %q, want %q", config.Java.Package, wantPackage)
		}

		wantClassNames := map[string]string{
			"google.cloud.secretmanager.v1.SecretManagerService": "SecretManager",
		}
		if diff := cmp.Diff(wantClassNames, config.Java.ServiceClassNames); diff != "" {
			t.Errorf("Java.ServiceClassNames mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("extract python settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(tmpDir, "google/cloud/secretmanager/v1", "python")
		if err != nil {
			t.Fatal(err)
		}

		if config == nil {
			t.Fatal("expected config, got nil")
		}

		if config.Python == nil {
			t.Fatal("expected Python settings, got nil")
		}

		if !config.Python.RestAsyncIOEnabled {
			t.Error("expected RestAsyncIOEnabled = true, got false")
		}
	})

	t.Run("extract node settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(tmpDir, "google/cloud/secretmanager/v1", "node")
		if err != nil {
			t.Fatal(err)
		}

		if config == nil {
			t.Fatal("expected config, got nil")
		}

		if config.Node == nil {
			t.Fatal("expected Node settings, got nil")
		}

		wantMethods := []string{
			"google.cloud.secretmanager.v1.SecretManagerService.GetSecret",
			"google.cloud.secretmanager.v1.SecretManagerService.CreateSecret",
		}
		if diff := cmp.Diff(wantMethods, config.Node.SelectiveMethods); diff != "" {
			t.Errorf("Node.SelectiveMethods mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("extract dotnet settings", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(tmpDir, "google/cloud/secretmanager/v1", "dotnet")
		if err != nil {
			t.Fatal(err)
		}

		if config == nil {
			t.Fatal("expected config, got nil")
		}

		if config.Dotnet == nil {
			t.Fatal("expected Dotnet settings, got nil")
		}

		wantServices := map[string]string{
			"SecretManagerService": "SecretManagerClient",
		}
		if diff := cmp.Diff(wantServices, config.Dotnet.RenamedServices); diff != "" {
			t.Errorf("Dotnet.RenamedServices mismatch (-want +got):\n%s", diff)
		}

		wantResources := map[string]string{
			"secretmanager.googleapis.com/Secret": "SecretManagerSecret",
		}
		if diff := cmp.Diff(wantResources, config.Dotnet.RenamedResources); diff != "" {
			t.Errorf("Dotnet.RenamedResources mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no service config found", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings(tmpDir, "google/cloud/nonexistent/v1", "java")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if config != nil {
			t.Errorf("expected nil config, got %v", config)
		}
	})

	t.Run("empty googleapis root", func(t *testing.T) {
		config, err := ExtractServiceConfigSettings("", "google/cloud/secretmanager/v1", "java")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if config != nil {
			t.Errorf("expected nil config, got %v", config)
		}
	})
}

func TestFindServiceConfigForAPI(t *testing.T) {
	tmpDir := t.TempDir()

	for _, test := range []struct {
		name      string
		apiPath   string
		createAs  string // filename to create
		wantFound bool
	}{
		{
			name:      "standard naming",
			apiPath:   "google/cloud/secretmanager/v1",
			createAs:  "secretmanager_v1.yaml",
			wantFound: true,
		},
		{
			name:      "service name only",
			apiPath:   "google/cloud/logging/v2",
			createAs:  "logging.yaml",
			wantFound: true,
		},
		{
			name:      "not found",
			apiPath:   "google/cloud/nonexistent/v1",
			createAs:  "",
			wantFound: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Create directory
			apiDir := filepath.Join(tmpDir, test.apiPath)
			if err := os.MkdirAll(apiDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create file if specified
			if test.createAs != "" {
				configPath := filepath.Join(apiDir, test.createAs)
				if err := os.WriteFile(configPath, []byte("name: test\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := findServiceConfigForAPI(tmpDir, test.apiPath)
			if test.wantFound && got == "" {
				t.Error("expected to find service config, but got empty string")
			}
			if !test.wantFound && got != "" {
				t.Errorf("expected not to find service config, but got %q", got)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	for _, test := range []struct {
		name   string
		config *LibraryConfig
		want   bool
	}{
		{
			name:   "nil config",
			config: nil,
			want:   true,
		},
		{
			name:   "empty config",
			config: &LibraryConfig{},
			want:   true,
		},
		{
			name: "with java",
			config: &LibraryConfig{
				Java: &JavaLibrary{Package: "com.example"},
			},
			want: false,
		},
		{
			name: "with python",
			config: &LibraryConfig{
				Python: &PythonLibrary{RestAsyncIOEnabled: true},
			},
			want: false,
		},
		{
			name: "with node",
			config: &LibraryConfig{
				Node: &NodeLibrary{SelectiveMethods: []string{"method1"}},
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isEmpty(test.config)
			if got != test.want {
				t.Errorf("isEmpty() = %v, want %v", got, test.want)
			}
		})
	}
}
