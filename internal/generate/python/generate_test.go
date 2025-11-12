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
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	sourceDir := filepath.Join(tmpDir, "source")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name: "google-cloud-language",
		Apis: []string{
			"google/cloud/language/v1",
		},
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

	if err := Generate(t.Context(), lib, outputDir, sourceDir, true); err != nil {
		t.Fatal(err)
	}
}
