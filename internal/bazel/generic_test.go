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

package bazel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseAPI_Go(t *testing.T) {
	content := `
go_grpc_library(
    name = "secretmanager_go_proto",
    importpath = "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb",
    protos = [":secretmanager_proto"],
)

go_gapic_library(
    name = "secretmanager_go_gapic",
    srcs = [":secretmanager_proto_with_info"],
    grpc_service_config = "secretmanager_grpc_service_config.json",
    importpath = "cloud.google.com/go/secretmanager/apiv1;secretmanager",
    metadata = True,
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = "secretmanager_v1.yaml",
    transport = "grpc+rest",
)
`
	tmpDir := t.TempDir()
	apiPath := "google/cloud/secretmanager/v1"
	buildDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join(buildDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseAPI(tmpDir, apiPath, "go")
	if err != nil {
		t.Fatalf("ParseAPI() failed: %v", err)
	}

	want := &APIConfig{
		Language:          "go",
		HasGAPIC:          true,
		GRPCServiceConfig: "secretmanager_grpc_service_config.json",
		ServiceYAML:       "secretmanager_v1.yaml",
		Transport:         "grpc+rest",
		RestNumericEnums:  true,
		ReleaseLevel:      "ga",
		Go: &GoConfig{
			ImportPath:    "cloud.google.com/go/secretmanager/apiv1;secretmanager",
			Metadata:      true,
			Diregapic:     false,
			HasGoGRPC:     true,
			HasLegacyGRPC: false,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParseAPI_Python(t *testing.T) {
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
	tmpDir := t.TempDir()
	apiPath := "google/cloud/secretmanager/v1"
	buildDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join(buildDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseAPI(tmpDir, apiPath, "python")
	if err != nil {
		t.Fatalf("ParseAPI() failed: %v", err)
	}

	want := &APIConfig{
		Language:          "python",
		HasGAPIC:          true,
		GRPCServiceConfig: "secretmanager_grpc_service_config.json",
		ServiceYAML:       "secretmanager_v1.yaml",
		Transport:         "grpc+rest",
		RestNumericEnums:  true,
		ReleaseLevel:      "",
		Python: &PythonConfig{
			OptArgs: []string{
				"warehouse-package-name=google-cloud-secret-manager",
				"python-gapic-namespace=google.cloud",
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParseAPI_Python_NoGapic(t *testing.T) {
	content := `
proto_library(
    name = "common_proto",
    srcs = ["common.proto"],
)
`
	tmpDir := t.TempDir()
	apiPath := "google/cloud/common"
	buildDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join(buildDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseAPI(tmpDir, apiPath, "python")
	if err != nil {
		t.Fatalf("ParseAPI() failed: %v", err)
	}

	if got.HasGAPIC {
		t.Error("HasGAPIC = true; want false for proto-only library")
	}
}

func TestParseAPI_Go_LegacyGRPC(t *testing.T) {
	content := `
go_proto_library(
    name = "secretmanager_go_proto",
    compilers = ["@io_bazel_rules_go//proto:go_grpc"],
    importpath = "cloud.google.com/go/secretmanager/apiv1beta2/secretmanagerpb",
    protos = [":secretmanager_proto"],
)

go_gapic_library(
    name = "secretmanager_go_gapic",
    srcs = [":secretmanager_proto_with_info"],
    grpc_service_config = "secretmanager_grpc_service_config.json",
    importpath = "cloud.google.com/go/secretmanager/apiv1beta2;secretmanager",
    metadata = True,
    release_level = "beta",
    rest_numeric_enums = True,
    service_yaml = "secretmanager_v1beta2.yaml",
    transport = "grpc+rest",
)
`
	tmpDir := t.TempDir()
	apiPath := "google/cloud/secretmanager/v1beta2"
	buildDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join(buildDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseAPI(tmpDir, apiPath, "go")
	if err != nil {
		t.Fatalf("ParseAPI() failed: %v", err)
	}

	if !got.Go.HasLegacyGRPC {
		t.Error("HasLegacyGRPC = false; want true")
	}
	if got.Go.HasGoGRPC {
		t.Error("HasGoGRPC = true; want false")
	}
}

func TestParseAPI_Python_WithReleaseLevel(t *testing.T) {
	content := `
py_gapic_library(
    name = "asset_py_gapic",
    srcs = [":asset_proto"],
    grpc_service_config = "cloudasset_grpc_service_config.json",
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = "cloudasset_v1.yaml",
    transport = "grpc+rest",
)
`
	tmpDir := t.TempDir()
	apiPath := "google/cloud/asset/v1"
	buildDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join(buildDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseAPI(tmpDir, apiPath, "python")
	if err != nil {
		t.Fatalf("ParseAPI() failed: %v", err)
	}

	if got.ReleaseLevel != "ga" {
		t.Errorf("ReleaseLevel = %q; want %q", got.ReleaseLevel, "ga")
	}
}

func TestParseAPI_UnsupportedLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	apiPath := "google/cloud/secretmanager/v1"
	buildDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join(buildDir, "BUILD.bazel")
	if err := os.WriteFile(buildPath, []byte("# empty"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseAPI(tmpDir, apiPath, "rust")
	if err == nil {
		t.Error("ParseAPI() succeeded; want error for unsupported language")
	}
}

func TestFindStringList(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		field   string
		want    []string
	}{
		{
			name: "simple list",
			content: `opt_args = [
        "warehouse-package-name=google-cloud-secret-manager",
        "python-gapic-namespace=google.cloud",
    ]`,
			field: "opt_args",
			want:  []string{"warehouse-package-name=google-cloud-secret-manager", "python-gapic-namespace=google.cloud"},
		},
		{
			name:    "single item",
			content: `opt_args = ["single-item"]`,
			field:   "opt_args",
			want:    []string{"single-item"},
		},
		{
			name:    "empty list",
			content: `opt_args = []`,
			field:   "opt_args",
			want:    []string{},
		},
		{
			name:    "not found",
			content: `other = ["foo"]`,
			field:   "opt_args",
			want:    nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findStringList(test.content, test.field)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
