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

// Package librarian provides functionality for managing Google Cloud client library configurations.
package librarian

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/language"
	"github.com/urfave/cli/v3"
)

// initCommand creates a new repository configuration.
func initCommand() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Usage:     "initialize librarian in current directory",
		UsageText: "librarian init <language> [--all]",
		Description: `Initialize librarian in current directory.
Creates librarian.yaml with default settings for the specified language.
Supported languages: go, python, rust

Example:
  librarian init go
  librarian init python`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "initialize with wildcard library discovery",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return errors.New("init requires a language argument")
			}
			language := cmd.Args().Get(0)
			version := Version()
			return runInit(language, version)
		},
	}
}

const (
	configPath = "librarian.yaml"
)

func runInit(lang, version string) (err error) {
	if _, err := os.Stat(configPath); err == nil {
		return errConfigAlreadyExists
	}
	commit, err := fetch.LatestGoogleapis()
	if err != nil {
		return err
	}
	if _, err := googleapisDir(commit); err != nil {
		return err
	}

	cfg := &config.Config{
		Language: lang,
		Version:  version,
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: commit,
			},
		},
	}
	cfg.Default, err = language.ConfigDefault(lang)
	if err != nil {
		return err
	}
	if err := cfg.Write(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("Created librarian.yaml\n")
	return nil
}

func googleapisDir(commit string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configCache := filepath.Join(home, ".librarian", "cache")
	if err := os.MkdirAll(configCache, 0o755); err != nil {
		return "", err
	}
	dir, err := fetch.DownloadAndExtractTarball("github.com/googleapis/googleapis", commit, configCache)
	if err != nil {
		return "", err
	}
	return dir, nil
}
