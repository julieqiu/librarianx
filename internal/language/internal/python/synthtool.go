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
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/julieqiu/librarianx/internal/fetch"
)

// InstallSynthtool installs synthtool from the latest commit of github.com/googleapis/synthtool.
// It downloads the tarball to cacheDir and runs pip install.
func InstallSynthtool(ctx context.Context, cacheDir string) error {
	repo := "github.com/googleapis/synthtool"

	commit, err := latestSynthtoolCommit()
	if err != nil {
		return fmt.Errorf("failed to get latest synthtool commit: %w", err)
	}

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

func latestSynthtoolCommit() (string, error) {
	url := "https://api.github.com/repos/googleapis/synthtool/commits/master"
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get latest SHA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP error fetching latest SHA: %s", resp.Status)
	}

	var body struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if body.SHA == "" {
		return "", fmt.Errorf("no SHA found in GitHub response")
	}
	return body.SHA, nil
}
