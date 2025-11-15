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

package golang_test

import (
	"context"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language/golang"
	"github.com/googleapis/librarian/internal/language/golang/release"
)

// TestAPIExists verifies that the top-level API functions exist and can be called.
// This is a smoke test to ensure the thin wrapper functions are properly exported.
func TestAPIExists(t *testing.T) {
	ctx := context.Background()

	// Test that Release function exists (it will fail in this test, but that's ok)
	lib := &config.Library{Name: "test"}
	_ = golang.Release(ctx, "", lib, "1.0.0", []*release.Change{})

	// Test that Publish function exists
	_ = golang.Publish(ctx, "", lib, "1.0.0")

	// We're just testing that the API compiles and is callable
	// The actual functionality is tested in the subpackages
}
