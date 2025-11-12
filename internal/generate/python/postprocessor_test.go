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
	"testing"
)

func TestCopyREADME(t *testing.T) {
	tmpDir := t.TempDir()
	libraryPath := "packages/google-cloud-language"
	libraryDir := filepath.Join(tmpDir, libraryPath)

	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		t.Fatal(err)
	}

	readmeContent := "# Test README\n\nThis is a test."
	readmeSrc := filepath.Join(libraryDir, "README.rst")
	if err := os.WriteFile(readmeSrc, []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopyREADME(tmpDir, libraryPath); err != nil {
		t.Fatal(err)
	}

	readmeDst := filepath.Join(libraryDir, "docs", "README.rst")
	got, err := os.ReadFile(readmeDst)
	if err != nil {
		t.Fatalf("failed to read copied README: %v", err)
	}

	if string(got) != readmeContent {
		t.Errorf("got README content %q, want %q", string(got), readmeContent)
	}
}

func TestCopyREADME_NoSourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	libraryPath := "packages/google-cloud-language"
	libraryDir := filepath.Join(tmpDir, libraryPath)

	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CopyREADME(tmpDir, libraryPath); err != nil {
		t.Fatalf("expected no error when README.rst does not exist, got: %v", err)
	}
}

func TestPostProcessorConfig(t *testing.T) {
	cfg := &PostProcessorConfig{
		OutputDir:   "/output",
		LibraryPath: "packages/google-cloud-language",
		StagingDir:  "/staging",
	}

	if cfg.OutputDir != "/output" {
		t.Errorf("got OutputDir %q, want %q", cfg.OutputDir, "/output")
	}
	if cfg.LibraryPath != "packages/google-cloud-language" {
		t.Errorf("got LibraryPath %q, want %q", cfg.LibraryPath, "packages/google-cloud-language")
	}
	if cfg.StagingDir != "/staging" {
		t.Errorf("got StagingDir %q, want %q", cfg.StagingDir, "/staging")
	}
}
