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
	"path"
	"strings"
)

// GetLibraryName returns the effective library name for a LibraryEntry.
// In the new format, the library name is stored directly in e.Name.
func (e *LibraryEntry) GetLibraryName(language, packaging string) string {
	return e.Name
}

// GetLibraryPath returns the effective library path for a LibraryEntry.
// If a path override is specified in the config, it uses that.
// Otherwise, it derives the path from the library name and generate_dir template.
func (e *LibraryEntry) GetLibraryPath(language, packaging, generateDir string) (string, error) {
	// Use explicit path if provided
	if e.Config != nil && e.Config.Path != "" {
		return e.Config.Path, nil
	}

	// Get API path from config
	apiPath := ""
	if e.Config != nil && e.Config.API != nil {
		if apiStr, ok := e.Config.API.(string); ok {
			apiPath = apiStr
		}
	}

	// Expand generate_dir template
	result := generateDir
	result = strings.ReplaceAll(result, "{name}", e.Name)
	if apiPath != "" {
		result = strings.ReplaceAll(result, "{api.path}", apiPath)
	}

	// Clean up the path
	return path.Clean(result), nil
}
