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

package config

import (
	"testing"
)

func TestReadGoLibrarianData(t *testing.T) {
	cfg, err := Read("../../data/go/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Version != "v0.5.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "v0.5.0")
	}

	if cfg.Language != "go" {
		t.Errorf("Language = %q, want %q", cfg.Language, "go")
	}

	if cfg.Container == nil {
		t.Fatal("Container is nil")
	}

	if cfg.Container.Image != "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go" {
		t.Errorf("Container.Image = %q, want %q", cfg.Container.Image, "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go")
	}

	if cfg.Defaults == nil {
		t.Fatal("Defaults is nil")
	}

	if cfg.Defaults.GeneratedDir != "./" {
		t.Errorf("Defaults.GeneratedDir = %q, want %q", cfg.Defaults.GeneratedDir, "./")
	}

	if cfg.Defaults.Transport != "grpc+rest" {
		t.Errorf("Defaults.Transport = %q, want %q", cfg.Defaults.Transport, "grpc+rest")
	}

	if cfg.Defaults.ReleaseLevel != "stable" {
		t.Errorf("Defaults.ReleaseLevel = %q, want %q", cfg.Defaults.ReleaseLevel, "stable")
	}

	if cfg.Release == nil {
		t.Fatal("Release is nil")
	}

	if cfg.Release.TagFormat != "{id}/v{version}" {
		t.Errorf("Release.TagFormat = %q, want %q", cfg.Release.TagFormat, "{id}/v{version}")
	}

	if len(cfg.Libraries) != 183 {
		t.Errorf("Libraries count = %d, want %d", len(cfg.Libraries), 183)
	}

	// Check first library with generate section
	var foundGenerate bool
	for _, lib := range cfg.Libraries {
		if lib.Generate != nil {
			foundGenerate = true
			if len(lib.Generate.APIs) == 0 {
				t.Errorf("Library %q has generate section but no APIs", lib.Name)
			}
			if lib.Version == "" {
				t.Errorf("Library %q has no version", lib.Name)
			}
			break
		}
	}

	if !foundGenerate {
		t.Error("No library with generate section found")
	}
}
