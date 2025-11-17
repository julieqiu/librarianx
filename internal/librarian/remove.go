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
	"fmt"

	"github.com/googleapis/librarian/internal/config"
)

// Remove removes a library from all sections of librarian.yaml.
// Modifies cfg in place.
func Remove(ctx context.Context, cfg *config.Config, name string) error {
	found := false

	// Remove from versions
	if cfg.Versions != nil {
		if _, exists := cfg.Versions[name]; exists {
			delete(cfg.Versions, name)
			found = true
		}
	}

	// Remove from name_overrides (find all entries with this name as value)
	if cfg.NameOverrides != nil {
		for apiPath, libraryName := range cfg.NameOverrides {
			if libraryName == name {
				delete(cfg.NameOverrides, apiPath)
				found = true
			}
		}
	}

	// Remove from libraries
	var newLibraries []*config.Library
	for _, lib := range cfg.Libraries {
		if lib.Name != name {
			newLibraries = append(newLibraries, lib)
		} else {
			found = true
		}
	}
	cfg.Libraries = newLibraries

	// Check if anything was removed
	if !found {
		return fmt.Errorf("library %q not found in librarian.yaml", name)
	}

	return nil
}
