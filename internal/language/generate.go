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

package language

import (
	"context"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language/internal/rust"
)

// Generate generates a client library for the specified language.
func Generate(ctx context.Context, library *config.Library, googleapisDir, serviceConfigPath, defaultOutput string) error {
	return rust.Generate(ctx, library, googleapisDir, serviceConfigPath, defaultOutput)
}

// APIToGenerate represents an API to be generated.
type APIToGenerate struct {
	Path              string
	ServiceConfigPath string
}

// GenerateAll generates all discovered APIs.
func GenerateAll(ctx context.Context, googleapisDir, defaultOutput string, apis []APIToGenerate) error {
	for _, api := range apis {
		// Create a minimal library config for this API
		library := &config.Library{
			API:  api.Path,
			Rust: &config.RustCrate{},
		}

		if err := rust.Generate(ctx, library, googleapisDir, api.ServiceConfigPath, defaultOutput); err != nil {
			return err
		}
	}
	return nil
}
