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

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGapicCommand(t *testing.T) {
	// Create a temporary directory structure with proto files for testing
	tmpDir := t.TempDir()

	// Setup test directories and files
	testAPIs := []string{
		"google/cloud/language/v1",
		"google/cloud/secretmanager/v1",
		"google/cloud/vision/v1",
	}

	for _, apiPath := range testAPIs {
		apiDir := filepath.Join(tmpDir, apiPath)
		if err := os.MkdirAll(apiDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create test proto files
		testProtos := []string{"service.proto", "resources.proto"}
		for _, proto := range testProtos {
			protoPath := filepath.Join(apiDir, proto)
			if err := os.WriteFile(protoPath, []byte("syntax = \"proto3\";"), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	for _, test := range []struct {
		name         string
		apiPath      string
		outputDir    string
		opts         *GapicOptions
		wantProtoLen int
		wantErr      bool
	}{
		{
			name:      "basic GAPIC",
			apiPath:   "google/cloud/language/v1",
			outputDir: "/output",
			opts: &GapicOptions{
				GrpcServiceConfig: "language_grpc_service_config.json",
				ServiceYAML:       "language_v1.yaml",
				Transport:         "grpc+rest",
				RestNumericEnums:  true,
			},
			wantProtoLen: 2,
		},
		{
			name:      "with opt args",
			apiPath:   "google/cloud/secretmanager/v1",
			outputDir: "/output",
			opts: &GapicOptions{
				GrpcServiceConfig: "secretmanager_grpc_service_config.json",
				Transport:         "grpc+rest",
				OptArgs:           []string{"python-gapic-namespace=google.cloud", "python-gapic-name=secretmanager"},
			},
			wantProtoLen: 2,
		},
		{
			name:         "minimal options",
			apiPath:      "google/cloud/vision/v1",
			outputDir:    "/output",
			opts:         &GapicOptions{},
			wantProtoLen: 2,
		},
		{
			name:      "empty path",
			apiPath:   "",
			outputDir: "/output",
			opts:      &GapicOptions{},
			wantErr:   true,
		},
		{
			name:      "nonexistent path",
			apiPath:   "google/cloud/nonexistent/v1",
			outputDir: "/output",
			opts:      &GapicOptions{},
			wantErr:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildGapicCommand(test.apiPath, tmpDir, test.outputDir, test.opts)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if got.Command != "protoc" {
				t.Errorf("got command %q, want %q", got.Command, "protoc")
			}

			// Verify proto_path
			if got.Args[0] != "--proto_path="+tmpDir {
				t.Errorf("got proto_path %q, want %q", got.Args[0], "--proto_path="+tmpDir)
			}

			// Verify python_gapic_out
			if got.Args[1] != "--python_gapic_out="+test.outputDir {
				t.Errorf("got python_gapic_out %q, want %q", got.Args[1], "--python_gapic_out="+test.outputDir)
			}

			// Verify gapic opt is present
			if !strings.HasPrefix(got.Args[2], "--python_gapic_opt=") {
				t.Errorf("expected python_gapic_opt, got %q", got.Args[2])
			}

			// Verify proto files are expanded
			protoFiles := got.Args[3:]
			if len(protoFiles) != test.wantProtoLen {
				t.Errorf("got %d proto files, want %d", len(protoFiles), test.wantProtoLen)
			}

			// Verify all proto file paths are absolute and end with .proto
			for _, protoFile := range protoFiles {
				if !strings.HasSuffix(protoFile, ".proto") {
					t.Errorf("proto file %q doesn't end with .proto", protoFile)
				}
				if !filepath.IsAbs(protoFile) {
					t.Errorf("proto file %q is not absolute", protoFile)
				}
			}
		})
	}
}

func TestBuildProtoCommand(t *testing.T) {
	// Create a temporary directory structure with proto files for testing
	tmpDir := t.TempDir()

	// Setup test directory and files
	apiDir := filepath.Join(tmpDir, "google/type")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create test proto files
	testProtos := []string{"date.proto", "latlng.proto"}
	for _, proto := range testProtos {
		protoPath := filepath.Join(apiDir, proto)
		if err := os.WriteFile(protoPath, []byte("syntax = \"proto3\";"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range []struct {
		name         string
		apiPath      string
		outputDir    string
		wantProtoLen int
		wantErr      bool
	}{
		{
			name:         "proto only",
			apiPath:      "google/type",
			outputDir:    "/output",
			wantProtoLen: 2,
		},
		{
			name:      "empty path",
			apiPath:   "",
			outputDir: "/output",
			wantErr:   true,
		},
		{
			name:      "nonexistent path",
			apiPath:   "google/nonexistent",
			outputDir: "/output",
			wantErr:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildProtoCommand(test.apiPath, tmpDir, test.outputDir)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if got.Command != "protoc" {
				t.Errorf("got command %q, want %q", got.Command, "protoc")
			}

			// Verify proto_path
			if got.Args[0] != "--proto_path="+tmpDir {
				t.Errorf("got proto_path %q, want %q", got.Args[0], "--proto_path="+tmpDir)
			}

			// Verify python_out
			if got.Args[1] != "--python_out="+test.outputDir {
				t.Errorf("got python_out %q, want %q", got.Args[1], "--python_out="+test.outputDir)
			}

			// Verify pyi_out
			if got.Args[2] != "--pyi_out="+test.outputDir {
				t.Errorf("got pyi_out %q, want %q", got.Args[2], "--pyi_out="+test.outputDir)
			}

			// Verify proto files are expanded
			protoFiles := got.Args[3:]
			if len(protoFiles) != test.wantProtoLen {
				t.Errorf("got %d proto files, want %d", len(protoFiles), test.wantProtoLen)
			}

			// Verify all proto file paths are absolute and end with .proto
			for _, protoFile := range protoFiles {
				if !strings.HasSuffix(protoFile, ".proto") {
					t.Errorf("proto file %q doesn't end with .proto", protoFile)
				}
				if !filepath.IsAbs(protoFile) {
					t.Errorf("proto file %q is not absolute", protoFile)
				}
			}
		})
	}
}

func TestGapicOptionsFormatting(t *testing.T) {
	// Create a temporary directory with proto files
	tmpDir := t.TempDir()
	apiPath := "google/cloud/test/v1"
	apiDir := filepath.Join(tmpDir, apiPath)
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a test proto file
	protoPath := filepath.Join(apiDir, "test.proto")
	if err := os.WriteFile(protoPath, []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &GapicOptions{
		GrpcServiceConfig: "test_grpc_service_config.json",
		ServiceYAML:       "test_v1.yaml",
		Transport:         "grpc+rest",
		RestNumericEnums:  true,
		OptArgs:           []string{"opt1=val1", "opt2=val2"},
	}

	got, err := BuildGapicCommand(apiPath, tmpDir, "/out", opts)
	if err != nil {
		t.Fatal(err)
	}

	var gapicOptArg string
	for _, arg := range got.Args {
		if strings.HasPrefix(arg, "--python_gapic_opt=") {
			gapicOptArg = arg
			break
		}
	}

	if gapicOptArg == "" {
		t.Fatal("no --python_gapic_opt argument found")
	}

	wantSubstrings := []string{
		"metadata",
		"retry-config=",
		"service-yaml=",
		"transport=grpc+rest",
		"rest-numeric-enums",
		"opt1=val1",
		"opt2=val2",
	}

	for _, substr := range wantSubstrings {
		if !strings.Contains(gapicOptArg, substr) {
			t.Errorf("gapic opt arg missing %q: %s", substr, gapicOptArg)
		}
	}
}
