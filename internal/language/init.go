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
	"fmt"

	"github.com/julieqiu/librarianx/internal/config"
	golang "github.com/julieqiu/librarianx/internal/language/internal/go"
	"github.com/julieqiu/librarianx/internal/language/internal/python"
	"github.com/julieqiu/librarianx/internal/language/internal/rust"
)

// Init initializes a default config for the given language.
// It returns the default config and updates cfg.Sources with language-specific sources.
func Init(ctx context.Context, language, cacheDir string, cfg *config.Config) (*config.Default, error) {
	switch language {
	case "go":
		return golang.Init(), nil
	case "rust":
		if err := rust.SetupWorkspace("."); err != nil {
			return nil, err
		}
	case "python":
		defaults, pythonSources, err := python.Init(ctx, cacheDir)
		if err != nil {
			return nil, err
		}
		if cfg.Sources == nil {
			cfg.Sources = &config.Sources{}
		}
		cfg.Sources.Python = pythonSources
		return defaults, nil
	}
	return nil, fmt.Errorf("not supported: %q", language)
}
