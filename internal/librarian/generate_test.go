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
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateLibraryForAPI_Disabled(t *testing.T) {
	cfg := &config.Config{
		Default: &config.Default{
			Output: "output/",
			Generate: &config.DefaultGenerate{
				All: true,
			},
		},
		Libraries: []*config.Library{
			{
				Name: "test-library",
				API:  "google/test/v1",
				Generate: &config.LibraryGenerate{
					Disabled: true,
				},
			},
		},
	}

	// This should return nil without error, skipping generation
	err := generateLibraryForAPI(context.Background(), cfg, "/fake/googleapis", "google/test/v1", "/fake/service.yaml")
	if err != nil {
		t.Errorf("generateLibraryForAPI with disabled library should return nil, got error: %v", err)
	}
}
