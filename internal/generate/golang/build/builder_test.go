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

package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

// testEnv encapsulates a temporary test environment.
type testEnv struct {
	tmpDir  string
	repoDir string
}

func TestBuild(t *testing.T) {
	library := &config.Library{Name: "foo"}

	tests := []struct {
		name           string
		buildErr       error
		wantErr        bool
		wantExecvCount int
	}{
		{
			name:           "happy path",
			wantErr:        false,
			wantExecvCount: 1,
		},
		{
			name:           "go build fails",
			buildErr:       errors.New("build failed"),
			wantErr:        true,
			wantExecvCount: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := newTestEnv(t)
			defer e.cleanup(t)

			var execvCount int
			execvRun = func(ctx context.Context, args []string, dir string) error {
				execvCount++
				want := filepath.Join(e.repoDir, "foo")
				if dir != want {
					t.Errorf("execv called with wrong working directory %s; want %s", dir, want)
				}
				switch {
				case slices.Equal(args, []string{"go", "build", "./..."}):
					return test.buildErr
				default:
					t.Errorf("execv called with unexpected args %v", args)
					return nil
				}
			}

			if err := Build(t.Context(), e.repoDir, library); (err != nil) != test.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, test.wantErr)
			}

			if execvCount != test.wantExecvCount {
				t.Errorf("execv called = %v; want %v", execvCount, test.wantExecvCount)
			}
		})
	}
}

// cleanup removes the temporary directory.
func (e *testEnv) cleanup(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(e.tmpDir); err != nil {
		t.Fatalf("failed to remove temp dir: %v", err)
	}
}

// newTestEnv creates a new test environment.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	tmpDir := t.TempDir()
	e := &testEnv{
		tmpDir:  tmpDir,
		repoDir: filepath.Join(tmpDir, "repo"),
	}
	if err := os.Mkdir(e.repoDir, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", e.repoDir, err)
	}

	return e
}
