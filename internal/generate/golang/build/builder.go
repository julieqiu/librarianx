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

// Package build provides functionality for building and testing generated Go code.
package build

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/generate/golang/execv"
)

// Test substitution vars.
var (
	execvRun = execv.Run
)

// Build is the main entrypoint for the `build` command. It runs `go build`
// and then `go test`.
func Build(ctx context.Context, repoDir string, library *config.Library) error {
	slog.Debug("librariangen: build command started", "library", library.Name)

	moduleDir := filepath.Join(repoDir, library.Name)
	if err := goBuild(ctx, moduleDir, library.Name); err != nil {
		return fmt.Errorf("librariangen: failed to run 'go build': %w", err)
	}
	// TODO(https://github.com/googleapis/google-cloud-go/issues/13335): run unit tests
	return nil
}

// goBuild builds all the code under the specified directory.
func goBuild(ctx context.Context, dir, module string) error {
	slog.Info("librariangen: building", "module", module)
	args := []string{"go", "build", "./..."}
	return execvRun(ctx, args, dir)
}
