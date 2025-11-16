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
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language"
	"github.com/urfave/cli/v3"
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate <library-name>",
		Description: `Generate a client library from googleapis.

Example:
  librarian generate google-cloud-secretmanager-v1`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return fmt.Errorf("generate requires a library name argument")
			}
			name := cmd.Args().Get(0)
			return runGenerate(ctx, name)
		},
	}
}

func runGenerate(ctx context.Context, name string) error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
		return fmt.Errorf("no googleapis source configured in %s", configPath)
	}

	commit := cfg.Sources.Googleapis.Commit
	if commit == "" {
		return fmt.Errorf("no commit specified for googleapis source in %s", configPath)
	}

	googleapisDir, err := googleapisDir(commit)
	if err != nil {
		return err
	}

	apiPath := strings.ReplaceAll(name, "-", "/")

	var library *config.Library
	for _, lib := range cfg.Libraries {
		if lib.API == apiPath {
			library = lib
			break
		}
	}

	if library == nil {
		if cfg.Default.Generate == nil || !cfg.Default.Generate.All {
			return fmt.Errorf("library %q not found in configuration and generate.all is false", name)
		}

		library = &config.Library{
			API: apiPath,
		}
	}

	applyDefaults(library, cfg.Default)

	apiDir := filepath.Join(googleapisDir, apiPath)
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		return fmt.Errorf("API path %q not found in googleapis", apiPath)
	} else if err != nil {
		return err
	}

	serviceConfigPath, err := findServiceConfig(googleapisDir, apiPath)
	if err != nil {
		return err
	}
	return language.Generate(ctx, library, googleapisDir, serviceConfigPath, cfg.Default.Output)
}

func applyDefaults(library *config.Library, defaults *config.Default) {
	if defaults.Generate != nil {
		if library.Transport == "" {
			library.Transport = defaults.Generate.Transport
		}
		if library.ReleaseLevel == "" {
			library.ReleaseLevel = defaults.Generate.ReleaseLevel
		}
		if library.RestNumericEnums == nil {
			b := defaults.Generate.RestNumericEnums
			library.RestNumericEnums = &b
		}
	}

	if defaults.Rust != nil {
		if library.Rust == nil {
			library.Rust = &config.RustCrate{}
		}
		if len(library.Rust.DisabledRustdocWarnings) == 0 {
			library.Rust.DisabledRustdocWarnings = defaults.Rust.DisabledRustdocWarnings
		}
		if len(library.Rust.PackageDependencies) == 0 {
			library.Rust.PackageDependencies = convertPackageDependencies(defaults.Rust.PackageDependencies)
		}
	}
}

func convertPackageDependencies(deps []*config.RustPackageDependency) []config.RustPackageDependency {
	result := make([]config.RustPackageDependency, len(deps))
	for i, dep := range deps {
		if dep != nil {
			result[i] = *dep
		}
	}
	return result
}

func findServiceConfig(googleapisDir, apiPath string) (string, error) {
	parts := strings.Split(apiPath, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid API path: %q", apiPath)
	}
	version := parts[len(parts)-1]

	dir := filepath.Join(googleapisDir, apiPath)
	pattern := filepath.Join(dir, "*_"+version+".yaml")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	var configs []string
	for _, m := range matches {
		if !strings.HasSuffix(m, "_gapic.yaml") {
			configs = append(configs, m)
		}
	}

	if len(configs) == 0 {
		return "", fmt.Errorf("no service config found in %q", dir)
	}

	return configs[0], nil
}
