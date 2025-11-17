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

	"github.com/julieqiu/librarianx/internal/config"
	"github.com/urfave/cli/v3"
)

func removeCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "remove a library from librarian.yaml",
		UsageText: "librarian remove <library-name>",
		Description: `Remove a library from all sections of librarian.yaml.

This removes the library from:
- versions:
- name_overrides:
- libraries:

Example:
  librarian remove google-cloud-secretmanager-v1`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return fmt.Errorf("remove requires a library name argument")
			}

			name := cmd.Args().Get(0)
			return runRemove(ctx, name)
		},
	}
}

func runRemove(ctx context.Context, name string) error {
	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	if err := Remove(ctx, cfg, name); err != nil {
		return err
	}

	if err := cfg.Write(configPath); err != nil {
		return err
	}

	fmt.Printf("✓ Removed %s from librarian.yaml\n", name)
	return nil
}
