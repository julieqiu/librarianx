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

package rust_test

import (
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language/rust"
)

func TestInit_All(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := rust.Init("0.0.1", true); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Read("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Language != "rust" {
		t.Errorf("Language = %q, want %q", cfg.Language, "rust")
	}

	if len(cfg.Libraries) != 1 {
		t.Fatalf("Libraries length = %d, want 1", len(cfg.Libraries))
	}

	if cfg.Libraries[0].Name != "*" {
		t.Errorf("Libraries[0].Name = %q, want %q", cfg.Libraries[0].Name, "*")
	}

	if cfg.Defaults == nil {
		t.Fatal("Defaults is nil")
	}

	if cfg.Defaults.Output != "src/generated/" {
		t.Errorf("Defaults.Output = %q, want %q", cfg.Defaults.Output, "src/generated/")
	}

	if cfg.Defaults.OneLibraryPer != "version" {
		t.Errorf("Defaults.OneLibraryPer = %q, want %q", cfg.Defaults.OneLibraryPer, "version")
	}

	if cfg.Defaults.ReleaseLevel != "stable" {
		t.Errorf("Defaults.ReleaseLevel = %q, want %q", cfg.Defaults.ReleaseLevel, "stable")
	}

	if cfg.Defaults.Rust == nil {
		t.Fatal("Defaults.Rust is nil")
	}

	if len(cfg.Defaults.Rust.DisabledRustdocWarnings) != 2 {
		t.Errorf("DisabledRustdocWarnings length = %d, want 2", len(cfg.Defaults.Rust.DisabledRustdocWarnings))
	}
}
