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

package bazel_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/bazel"
)

// Example_parseGo demonstrates parsing Go API configuration from BUILD.bazel
func Example_parseGo() {
	// Create a temporary directory with a BUILD.bazel file
	tmpDir := setupTestDir()
	defer os.RemoveAll(tmpDir)

	// Parse the BUILD.bazel file for Go
	cfg, err := bazel.ParseAPI(tmpDir, "google/cloud/asset/v1", "go")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check if GAPIC library is present
	if !cfg.HasGAPIC {
		fmt.Println("No GAPIC library found")
		return
	}

	// Access common fields
	fmt.Printf("Transport: %s\n", cfg.Transport)
	fmt.Printf("Service YAML: %s\n", cfg.ServiceYAML)
	fmt.Printf("gRPC Service Config: %s\n", cfg.GRPCServiceConfig)
	fmt.Printf("REST Numeric Enums: %t\n", cfg.RestNumericEnums)
	fmt.Printf("Release Level: %s\n", cfg.ReleaseLevel)

	// Access Go-specific fields
	if cfg.Go != nil {
		fmt.Printf("Import Path: %s\n", cfg.Go.ImportPath)
		fmt.Printf("Has Metadata: %t\n", cfg.Go.Metadata)
	}

	// Output:
	// Transport: grpc+rest
	// Service YAML: cloudasset_v1.yaml
	// gRPC Service Config: cloudasset_grpc_service_config.json
	// REST Numeric Enums: true
	// Release Level: ga
	// Import Path: cloud.google.com/go/asset/apiv1;asset
	// Has Metadata: true
}

// Example_parsePython demonstrates parsing Python API configuration from BUILD.bazel
func Example_parsePython() {
	// Create a temporary directory with a BUILD.bazel file
	tmpDir := setupPythonTestDir()
	defer os.RemoveAll(tmpDir)

	// Parse the BUILD.bazel file for Python
	cfg, err := bazel.ParseAPI(tmpDir, "google/cloud/secretmanager/v1", "python")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !cfg.HasGAPIC {
		fmt.Println("No GAPIC library found")
		return
	}

	// Access common fields
	fmt.Printf("Transport: %s\n", cfg.Transport)
	fmt.Printf("Service YAML: %s\n", cfg.ServiceYAML)

	// Access Python-specific fields
	if cfg.Python != nil && len(cfg.Python.OptArgs) > 0 {
		fmt.Printf("Python Opt Args:\n")
		for _, arg := range cfg.Python.OptArgs {
			fmt.Printf("  - %s\n", arg)
		}
	}

	// Output:
	// Transport: grpc+rest
	// Service YAML: secretmanager_v1.yaml
	// Python Opt Args:
	//   - warehouse-package-name=google-cloud-secret-manager
	//   - python-gapic-namespace=google.cloud
}

// Example_multipleAPIs demonstrates parsing multiple APIs for batch processing
func Example_multipleAPIs() {
	tmpDir := setupMultiAPITestDir()
	defer os.RemoveAll(tmpDir)

	apis := []string{
		"google/cloud/secretmanager/v1",
		"google/cloud/secretmanager/v1beta1",
	}

	for _, apiPath := range apis {
		cfg, err := bazel.ParseAPI(tmpDir, apiPath, "python")
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", apiPath, err)
			continue
		}

		if cfg.HasGAPIC {
			fmt.Printf("%s: %s\n", apiPath, cfg.Transport)
		}
	}

	// Output:
	// google/cloud/secretmanager/v1: grpc+rest
	// google/cloud/secretmanager/v1beta1: grpc+rest
}

// setupTestDir creates a temporary directory with a Go BUILD.bazel file
func setupTestDir() string {
	tmpDir, _ := os.MkdirTemp("", "bazel-example-")
	apiDir := filepath.Join(tmpDir, "google/cloud/asset/v1")
	os.MkdirAll(apiDir, 0755)

	content := `
go_gapic_library(
    name = "asset_go_gapic",
    srcs = [":asset_proto_with_info"],
    grpc_service_config = "cloudasset_grpc_service_config.json",
    importpath = "cloud.google.com/go/asset/apiv1;asset",
    metadata = True,
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = "cloudasset_v1.yaml",
    transport = "grpc+rest",
)
`
	os.WriteFile(filepath.Join(apiDir, "BUILD.bazel"), []byte(content), 0644)
	return tmpDir
}

// setupPythonTestDir creates a temporary directory with a Python BUILD.bazel file
func setupPythonTestDir() string {
	tmpDir, _ := os.MkdirTemp("", "bazel-python-")
	apiDir := filepath.Join(tmpDir, "google/cloud/secretmanager/v1")
	os.MkdirAll(apiDir, 0755)

	content := `
py_gapic_library(
    name = "secretmanager_py_gapic",
    srcs = [":secretmanager_proto"],
    grpc_service_config = "secretmanager_grpc_service_config.json",
    opt_args = [
        "warehouse-package-name=google-cloud-secret-manager",
        "python-gapic-namespace=google.cloud",
    ],
    rest_numeric_enums = True,
    service_yaml = "secretmanager_v1.yaml",
    transport = "grpc+rest",
)
`
	os.WriteFile(filepath.Join(apiDir, "BUILD.bazel"), []byte(content), 0644)
	return tmpDir
}

// setupMultiAPITestDir creates a temporary directory with multiple API versions
func setupMultiAPITestDir() string {
	tmpDir, _ := os.MkdirTemp("", "bazel-multi-")

	// Create v1
	v1Dir := filepath.Join(tmpDir, "google/cloud/secretmanager/v1")
	os.MkdirAll(v1Dir, 0755)
	v1Content := `
py_gapic_library(
    name = "secretmanager_py_gapic",
    transport = "grpc+rest",
)
`
	os.WriteFile(filepath.Join(v1Dir, "BUILD.bazel"), []byte(v1Content), 0644)

	// Create v1beta1
	v1beta1Dir := filepath.Join(tmpDir, "google/cloud/secretmanager/v1beta1")
	os.MkdirAll(v1beta1Dir, 0755)
	v1beta1Content := `
py_gapic_library(
    name = "secretmanager_py_gapic",
    transport = "grpc+rest",
)
`
	os.WriteFile(filepath.Join(v1beta1Dir, "BUILD.bazel"), []byte(v1beta1Content), 0644)

	return tmpDir
}
