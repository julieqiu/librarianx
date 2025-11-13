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
	"fmt"

	"github.com/googleapis/librarian/internal/config"
)

// Generate generates Python client library code.
// This function is already implemented in generate.go.

// Release prepares a Python library for release by updating version files,
// changelogs, and snippet metadata.
func Release(ctx context.Context, lib *config.Library, version string, changes []*Change, repoDir string) error {
	// Normalize version for Python (PEP 440)
	pythonVersion := normalizePythonVersion(version)

	// 1. Run tests
	if err := runPythonTests(ctx, lib.Location); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	// 2. Update version files
	if err := updateVersionFiles(lib.Location, pythonVersion); err != nil {
		return fmt.Errorf("failed to update version files: %w", err)
	}

	// 3. Update changelogs
	if err := updateChangelogs(repoDir, lib, pythonVersion, changes); err != nil {
		return fmt.Errorf("failed to update changelogs: %w", err)
	}

	// 4. Update snippet metadata (if exists)
	if err := updateSnippetMetadata(repoDir, lib, pythonVersion); err != nil {
		return fmt.Errorf("failed to update snippet metadata: %w", err)
	}

	return nil
}

// Publish publishes a Python library to PyPI.
// Note: Python libraries typically do NOT use librarian publish.
// Publishing to PyPI is handled separately via CI/CD automation.
// This function is provided for interface compatibility.
func Publish(ctx context.Context, lib *config.Library, repoDir string) error {
	return fmt.Errorf("python libraries do not use 'librarian publish'. Publishing to PyPI is handled via CI/CD automation")
}
