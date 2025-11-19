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
	"os/exec"

	"github.com/julieqiu/librarianx/internal/fetch"
)

// installSynthtool installs synthtool from a specific commit of github.com/googleapis/synthtool.
// It downloads the tarball to cacheDir and runs pip install.
func installSynthtool(ctx context.Context, cacheDir, commit string) error {
	repo := "github.com/googleapis/synthtool"

	synthtoolDir, err := fetch.DownloadAndExtractTarball(repo, commit, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to download synthtool: %w", err)
	}

	cmd := exec.CommandContext(ctx, "pip3", "install", "--user", ".")
	cmd.Dir = synthtoolDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install synthtool: %w", err)
	}

	return nil
}
