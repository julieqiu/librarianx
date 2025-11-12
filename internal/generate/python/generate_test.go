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
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/generate/golang/request"
)

func TestConfigValidate(t *testing.T) {
	for _, test := range []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				LibrarianDir: "/librarian",
				OutputDir:    "/output",
				SourceDir:    "/source",
				StagingDir:   "/staging",
			},
			wantErr: false,
		},
		{
			name: "missing librarian dir",
			cfg: &Config{
				OutputDir:  "/output",
				SourceDir:  "/source",
				StagingDir: "/staging",
			},
			wantErr: true,
		},
		{
			name: "missing output dir",
			cfg: &Config{
				LibrarianDir: "/librarian",
				SourceDir:    "/source",
				StagingDir:   "/staging",
			},
			wantErr: true,
		},
		{
			name: "missing source dir",
			cfg: &Config{
				LibrarianDir: "/librarian",
				OutputDir:    "/output",
				StagingDir:   "/staging",
			},
			wantErr: true,
		},
		{
			name: "missing staging dir",
			cfg: &Config{
				LibrarianDir: "/librarian",
				OutputDir:    "/output",
				SourceDir:    "/source",
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.Validate()
			if test.wantErr && err == nil {
				t.Fatal("expected error but got none")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReadGenerateReq(t *testing.T) {
	tmpDir := t.TempDir()
	reqPath := filepath.Join(tmpDir, "generate-request.json")

	lib := &request.Library{
		ID:      "google-cloud-language",
		Version: "1.0.0",
		APIs: []request.API{
			{
				Path:          "google/cloud/language/v1",
				ServiceConfig: "language_grpc_service_config.json",
			},
		},
	}

	data, err := json.Marshal(lib)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(reqPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Mock requestParse
	oldRequestParse := requestParse
	defer func() { requestParse = oldRequestParse }()
	requestParse = request.ParseLibrary

	got, err := readGenerateReq(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != lib.ID {
		t.Errorf("got ID %q, want %q", got.ID, lib.ID)
	}
	if len(got.APIs) != len(lib.APIs) {
		t.Errorf("got %d APIs, want %d", len(got.APIs), len(lib.APIs))
	}
}

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	librarianDir := filepath.Join(tmpDir, "librarian")
	outputDir := filepath.Join(tmpDir, "output")
	sourceDir := filepath.Join(tmpDir, "source")
	stagingDir := filepath.Join(tmpDir, "staging")

	if err := os.MkdirAll(librarianDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	lib := &request.Library{
		ID:      "google-cloud-language",
		Version: "1.0.0",
		APIs: []request.API{
			{
				Path:          "google/cloud/language/v1",
				ServiceConfig: "language_grpc_service_config.json",
			},
		},
	}

	reqPath := filepath.Join(librarianDir, "generate-request.json")
	data, err := json.Marshal(lib)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reqPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Create mock API directory
	apiDir := filepath.Join(sourceDir, "google/cloud/language/v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Mock execvRun to avoid actually running protoc
	oldExecvRun := execvRun
	defer func() { execvRun = oldExecvRun }()
	execvRun = func(ctx context.Context, args []string, workDir string) error {
		return nil
	}

	cfg := &Config{
		LibrarianDir:         librarianDir,
		OutputDir:            outputDir,
		SourceDir:            sourceDir,
		StagingDir:           stagingDir,
		DisablePostProcessor: true,
	}

	if err := Generate(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}

	// Verify staging directory was created
	if _, err := os.Stat(stagingDir); os.IsNotExist(err) {
		t.Errorf("staging directory was not created")
	}
}
