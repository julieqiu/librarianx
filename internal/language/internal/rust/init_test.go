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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/julieqiu/librarianx/internal/config"
)

func TestConfigDefault(t *testing.T) {
	got := ConfigDefault()

	if got.Output != "src/generated/" {
		t.Errorf("Output = %q, want %q", got.Output, "src/generated/")
	}
	if got.Generate.OneLibraryPer != "version" {
		t.Errorf("Generate.OneLibraryPer = %q, want %q", got.Generate.OneLibraryPer, "version")
	}
	if got.Release.Remote != "upstream" {
		t.Errorf("Release.Remote = %q, want %q", got.Release.Remote, "upstream")
	}
	if got.Rust == nil {
		t.Fatal("Rust config should not be nil")
	}
	if len(got.Rust.DisabledRustdocWarnings) != 2 {
		t.Errorf("len(Rust.DisabledRustdocWarnings) = %d, want 2", len(got.Rust.DisabledRustdocWarnings))
	}
	if len(got.Rust.PackageDependencies) == 0 {
		t.Error("Rust.PackageDependencies should not be empty")
	}
}

func TestConfigDefaultMatchesConfigInit(t *testing.T) {
	got := ConfigDefault()
	want, err := config.Init("rust")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
