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

	"github.com/google/go-cmp/cmp"
)

func TestParseBuildBazel(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    *BazelConfig
	}{
		{
			name: "full py_gapic_library",
			content: `
py_gapic_library(
    name = "secretmanager_py_gapic",
    srcs = [":secretmanager_proto"],
    grpc_service_config = "secretmanager_grpc_service_config.json",
    opt_args = [
        "warehouse-package-name=google-cloud-secret-manager",
    ],
    rest_numeric_enums = True,
    service_yaml = "secretmanager_v1.yaml",
    transport = "grpc+rest",
    deps = [
        "//google/iam/v1:iam_policy_py_proto",
    ],
)
`,
			want: &BazelConfig{
				GRPCServiceConfig: "secretmanager_grpc_service_config.json",
				ServiceYAML:       "secretmanager_v1.yaml",
				Transport:         "grpc+rest",
				RestNumericEnums:  true,
				OptArgs:           []string{"warehouse-package-name=google-cloud-secret-manager"},
				IsProtoOnly:       false,
			},
		},
		{
			name: "proto-only library",
			content: `
proto_library(
    name = "common_proto",
    srcs = ["resource.proto"],
)
`,
			want: &BazelConfig{
				IsProtoOnly: true,
			},
		},
		{
			name: "multiple opt_args",
			content: `
py_gapic_library(
    name = "test_py_gapic",
    grpc_service_config = "test.json",
    opt_args = [
        "python-gapic-name=TestService",
        "python-gapic-namespace=google.cloud.test",
        "warehouse-package-name=google-cloud-test",
    ],
    transport = "grpc",
)
`,
			want: &BazelConfig{
				GRPCServiceConfig: "test.json",
				Transport:         "grpc",
				OptArgs: []string{
					"python-gapic-name=TestService",
					"python-gapic-namespace=google.cloud.test",
					"warehouse-package-name=google-cloud-test",
				},
			},
		},
		{
			name: "rest_numeric_enums false",
			content: `
py_gapic_library(
    name = "test_py_gapic",
    rest_numeric_enums = False,
    transport = "rest",
)
`,
			want: &BazelConfig{
				Transport:        "rest",
				RestNumericEnums: false,
			},
		},
		{
			name: "minimal py_gapic_library",
			content: `
py_gapic_library(
    name = "minimal_py_gapic",
    srcs = [":proto"],
)
`,
			want: &BazelConfig{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseBuildBazel(test.content)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBazelConfig_GetPaths(t *testing.T) {
	config := &BazelConfig{
		GRPCServiceConfig: "test_grpc_service_config.json",
		ServiceYAML:       "test_v1.yaml",
	}

	googleapisDir := "/googleapis"
	apiPath := "google/cloud/test/v1"

	gotGRPC := config.GetGRPCServiceConfigPath(googleapisDir, apiPath)
	wantGRPC := "/googleapis/google/cloud/test/v1/test_grpc_service_config.json"
	if gotGRPC != wantGRPC {
		t.Errorf("GetGRPCServiceConfigPath() = %q, want %q", gotGRPC, wantGRPC)
	}

	gotYAML := config.GetServiceYAMLPath(googleapisDir, apiPath)
	wantYAML := "/googleapis/google/cloud/test/v1/test_v1.yaml"
	if gotYAML != wantYAML {
		t.Errorf("GetServiceYAMLPath() = %q, want %q", gotYAML, wantYAML)
	}
}

func TestBazelConfig_MergeWithLibrary(t *testing.T) {
	for _, test := range []struct {
		name   string
		bazel  *BazelConfig
		lib    *Library
		want   *Library
	}{
		{
			name: "merge transport",
			bazel: &BazelConfig{
				Transport: "grpc+rest",
			},
			lib: &Library{
				Name: "test-lib",
			},
			want: &Library{
				Name:      "test-lib",
				Transport: "grpc+rest",
			},
		},
		{
			name: "library transport takes precedence",
			bazel: &BazelConfig{
				Transport: "grpc+rest",
			},
			lib: &Library{
				Name:      "test-lib",
				Transport: "grpc",
			},
			want: &Library{
				Name:      "test-lib",
				Transport: "grpc",
			},
		},
		{
			name: "merge opt_args",
			bazel: &BazelConfig{
				OptArgs: []string{"warehouse-package-name=test-package"},
			},
			lib: &Library{
				Name: "test-lib",
			},
			want: &Library{
				Name: "test-lib",
				Python: &PythonPackage{
					OptArgs: []string{"warehouse-package-name=test-package"},
				},
			},
		},
		{
			name: "merge opt_args with existing",
			bazel: &BazelConfig{
				OptArgs: []string{"warehouse-package-name=test-package"},
			},
			lib: &Library{
				Name: "test-lib",
				Python: &PythonPackage{
					OptArgs: []string{"custom-arg=value"},
				},
			},
			want: &Library{
				Name: "test-lib",
				Python: &PythonPackage{
					OptArgs: []string{"custom-arg=value", "warehouse-package-name=test-package"},
				},
			},
		},
		{
			name: "no duplicate opt_args",
			bazel: &BazelConfig{
				OptArgs: []string{"arg=value"},
			},
			lib: &Library{
				Name: "test-lib",
				Python: &PythonPackage{
					OptArgs: []string{"arg=value"},
				},
			},
			want: &Library{
				Name: "test-lib",
				Python: &PythonPackage{
					OptArgs: []string{"arg=value"},
				},
			},
		},
		{
			name: "merge rest_numeric_enums",
			bazel: &BazelConfig{
				RestNumericEnums: true,
			},
			lib: &Library{
				Name: "test-lib",
			},
			want: &Library{
				Name:             "test-lib",
				RestNumericEnums: boolPtr(true),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.bazel.MergeWithLibrary(test.lib, nil)

			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestReadBuildBazel_RealFile(t *testing.T) {
	// Skip if googleapis directory doesn't exist
	googleapisDir := "/Users/julieqiu/code/googleapis/googleapis"
	if _, err := os.Stat(googleapisDir); os.IsNotExist(err) {
		t.Skip("googleapis directory not found")
	}

	config, err := ReadBuildBazel(googleapisDir, "google/cloud/secretmanager/v1")
	if err != nil {
		t.Fatal(err)
	}

	// Verify expected values
	if config.GRPCServiceConfig != "secretmanager_grpc_service_config.json" {
		t.Errorf("GRPCServiceConfig = %q, want %q", config.GRPCServiceConfig, "secretmanager_grpc_service_config.json")
	}

	if config.ServiceYAML != "secretmanager_v1.yaml" {
		t.Errorf("ServiceYAML = %q, want %q", config.ServiceYAML, "secretmanager_v1.yaml")
	}

	if config.Transport != "grpc+rest" {
		t.Errorf("Transport = %q, want %q", config.Transport, "grpc+rest")
	}

	if !config.RestNumericEnums {
		t.Errorf("RestNumericEnums = %v, want true", config.RestNumericEnums)
	}

	if len(config.OptArgs) == 0 {
		t.Errorf("OptArgs is empty, expected warehouse-package-name")
	}

	if config.IsProtoOnly {
		t.Error("IsProtoOnly = true, want false")
	}

	t.Logf("BUILD.bazel config: %s", config.String())
}
