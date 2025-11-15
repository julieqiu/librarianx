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

	"github.com/googleapis/librarian/internal/config"
)

func TestValidateGeneratedGoLibrarianYAML(t *testing.T) {
	// Read the generated data/go/librarian.yaml
	yamlPath := filepath.Join("..", "..", "data", "go", "librarian.yaml")

	cfg, err := config.Read(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify basic structure
	if cfg.Version != "v1" {
		t.Errorf("version: got %q, want %q", cfg.Version, "v1")
	}

	if cfg.Language != "go" {
		t.Errorf("language: got %q, want %q", cfg.Language, "go")
	}

	if cfg.Container == nil {
		t.Fatal("container is nil")
	}

	if cfg.Defaults == nil {
		t.Fatal("defaults is nil")
	}

	if cfg.Release == nil {
		t.Fatal("release is nil")
	}

	if len(cfg.Libraries) == 0 {
		t.Fatal("no libraries found")
	}

	t.Logf("Successfully validated librarian.yaml with %d libraries", len(cfg.Libraries))
}
