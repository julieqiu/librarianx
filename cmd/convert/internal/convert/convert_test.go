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

package convert

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

// Use the existing testdata from generate/golang.
const inputDir = "/Users/julieqiu/code/julieqiu/librarianx/internal/language/golang/testdata/generate"

func TestConvert(t *testing.T) {

	// Create a temporary output directory
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "librarian.yaml")

	// Run conversion
	if err := Convert(inputDir, outputFile); err != nil {
		t.Fatal(err)
	}

	// Read the output file
	got, err := config.Read(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	// Expected output
	want := &config.Config{
		Version:  "v1",
		Language: "go",
		Container: &config.Container{
			Image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go",
			Tag:   "latest",
		},
		Global: &config.Global{
			FilesAllowlist: []config.FileAllowlist{
				{Path: "go.work", Permissions: "read-write"},
				{Path: "internal/README.md.template", Permissions: "read-only"},
				{Path: "README.md", Permissions: "write-only"},
			},
		},
		Defaults: &config.Defaults{
			Output:           "{name}/",
			OneLibraryPer:    "service",
			Transport:        "grpc+rest",
			RestNumericEnums: true,
		},
		Release: &config.Release{
			TagFormat: "{name}/v{version}",
		},
		Libraries: []config.LibraryEntry{
			{Name: "*"},
			{
				Name: "secretmanager",
				Config: &config.LibraryConfig{
					APIs: []string{
						"google/cloud/secretmanager/v1",
						"google/cloud/secretmanager/v1beta2",
					},
					Keep: []string{
						"CHANGES.md",
						"aliasshim/aliasshim.go",
						"apiv1/iam.go",
						"apiv1/iam_example_test.go",
						"internal/version.go",
						"internal/generated/snippets/secretmanager/snippet_metadata.google.cloud.secretmanager.v1.json",
					},
					BazelMetadata: &config.BazelMetadata{
						ReleaseLevel:      "ga",
						GRPCServiceConfig: "secretmanager_grpc_service_config.json",
						Go: &config.BazelGoMetadata{
							ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
							Metadata:   true,
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParseImage(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   string
		wantImg string
		wantTag string
	}{
		{
			name:    "with tag",
			input:   "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go:latest",
			wantImg: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go",
			wantTag: "latest",
		},
		{
			name:    "without tag",
			input:   "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go",
			wantImg: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go",
			wantTag: "latest",
		},
		{
			name:    "with version tag",
			input:   "gcr.io/my-project/my-image:v1.2.3",
			wantImg: "gcr.io/my-project/my-image",
			wantTag: "v1.2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotImg, gotTag := parseImage(test.input)
			if gotImg != test.wantImg {
				t.Errorf("image: got %q, want %q", gotImg, test.wantImg)
			}
			if gotTag != test.wantTag {
				t.Errorf("tag: got %q, want %q", gotTag, test.wantTag)
			}
		})
	}
}
