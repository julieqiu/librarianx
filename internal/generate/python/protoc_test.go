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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildGapicCommand(t *testing.T) {
	for _, test := range []struct {
		name      string
		apiPath   string
		sourceDir string
		outputDir string
		opts      *GapicOptions
		wantArgs  []string
		wantErr   bool
	}{
		{
			name:      "basic GAPIC",
			apiPath:   "google/cloud/language/v1",
			sourceDir: "/source",
			outputDir: "/output",
			opts: &GapicOptions{
				GrpcServiceConfig: "language_grpc_service_config.json",
				ServiceYAML:       "language_v1.yaml",
				Transport:         "grpc+rest",
				RestNumericEnums:  true,
			},
			wantArgs: []string{
				"--proto_path=/source",
				"--python_gapic_out=/output",
				"--python_gapic_opt=metadata,retry-config=/source/google/cloud/language/v1/language_grpc_service_config.json,service-yaml=/source/google/cloud/language/v1/language_v1.yaml,transport=grpc+rest,rest-numeric-enums",
				"/source/google/cloud/language/v1/*.proto",
			},
		},
		{
			name:      "with opt args",
			apiPath:   "google/cloud/secretmanager/v1",
			sourceDir: "/source",
			outputDir: "/output",
			opts: &GapicOptions{
				GrpcServiceConfig: "secretmanager_grpc_service_config.json",
				Transport:         "grpc+rest",
				OptArgs:           []string{"python-gapic-namespace=google.cloud", "python-gapic-name=secretmanager"},
			},
			wantArgs: []string{
				"--proto_path=/source",
				"--python_gapic_out=/output",
				"--python_gapic_opt=metadata,retry-config=/source/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,transport=grpc+rest,python-gapic-namespace=google.cloud,python-gapic-name=secretmanager",
				"/source/google/cloud/secretmanager/v1/*.proto",
			},
		},
		{
			name:      "minimal options",
			apiPath:   "google/cloud/vision/v1",
			sourceDir: "/source",
			outputDir: "/output",
			opts:      &GapicOptions{},
			wantArgs: []string{
				"--proto_path=/source",
				"--python_gapic_out=/output",
				"--python_gapic_opt=metadata",
				"/source/google/cloud/vision/v1/*.proto",
			},
		},
		{
			name:      "empty path",
			apiPath:   "",
			sourceDir: "/source",
			outputDir: "/output",
			opts:      &GapicOptions{},
			wantErr:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildGapicCommand(test.apiPath, test.sourceDir, test.outputDir, test.opts)
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

			if diff := cmp.Diff(test.wantArgs, got.Args); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildProtoCommand(t *testing.T) {
	for _, test := range []struct {
		name      string
		apiPath   string
		sourceDir string
		outputDir string
		wantArgs  []string
		wantErr   bool
	}{
		{
			name:      "proto only",
			apiPath:   "google/type",
			sourceDir: "/source",
			outputDir: "/output",
			wantArgs: []string{
				"--proto_path=/source",
				"--python_out=/output",
				"--pyi_out=/output",
				"/source/google/type/*.proto",
			},
		},
		{
			name:      "empty path",
			apiPath:   "",
			sourceDir: "/source",
			outputDir: "/output",
			wantErr:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildProtoCommand(test.apiPath, test.sourceDir, test.outputDir)
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

			if diff := cmp.Diff(test.wantArgs, got.Args); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGapicOptionsFormatting(t *testing.T) {
	apiPath := "google/cloud/test/v1"
	opts := &GapicOptions{
		GrpcServiceConfig: "test_grpc_service_config.json",
		ServiceYAML:       "test_v1.yaml",
		Transport:         "grpc+rest",
		RestNumericEnums:  true,
		OptArgs:           []string{"opt1=val1", "opt2=val2"},
	}

	got, err := BuildGapicCommand(apiPath, "/src", "/out", opts)
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
