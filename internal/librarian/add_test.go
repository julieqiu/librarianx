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

package librarian

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/julieqiu/librarianx/internal/config"
)

func TestAdd(t *testing.T) {
	// Create temporary googleapis directory structure
	tmpDir := t.TempDir()
	googleapisDir := filepath.Join(tmpDir, "googleapis")
	apiDir := filepath.Join(googleapisDir, "google/cloud/secretmanager/v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Language: "rust",
		Default: &config.Default{
			Generate: &config.DefaultGenerate{
				OneLibraryPer: "channel",
			},
		},
	}

	library := &config.Library{
		Name:    "google-cloud-secretmanager-v1",
		Version: "1.0.0",
	}

	err := Add(context.Background(), cfg, googleapisDir, library)
	if err != nil {
		t.Fatal(err)
	}

	// Verify versions was updated
	if cfg.Versions["google-cloud-secretmanager-v1"] != "1.0.0" {
		t.Errorf("versions not updated correctly: got %q", cfg.Versions["google-cloud-secretmanager-v1"])
	}
}

func TestAdd_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	googleapisDir := filepath.Join(tmpDir, "googleapis")
	apiDir := filepath.Join(googleapisDir, "google/cloud/secretmanager/v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Language: "rust",
		Default: &config.Default{
			Generate: &config.DefaultGenerate{
				OneLibraryPer: "channel",
			},
		},
		Versions: map[string]string{
			"google-cloud-secretmanager-v1": "1.0.0",
		},
	}

	library := &config.Library{
		Name:    "google-cloud-secretmanager-v1",
		Version: "1.1.0",
	}

	err := Add(context.Background(), cfg, googleapisDir, library)
	if err == nil {
		t.Error("expected error when library already exists")
	}
}
