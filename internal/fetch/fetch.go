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

// Package fetch provides functions for fetching data from remote sources.
package fetch

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// LatestGoogleapis returns the latest commit SHA on master for googleapis/googleapis.
func LatestGoogleapis() (string, error) {
	return latestCommit("https://api.github.com/repos/googleapis/googleapis/commits/master")
}

func latestCommit(url string) (string, error) {
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

type Info struct {
	SHA256 string `json:"sha256"`
}

// DownloadAndExtractTarball downloads a tarball from the given source,
// verifies its SHA256 checksum, and extracts it to a temporary directory.
// It returns the path to the librarian cache.
//
// <cacheDir>/cache/download/<repo>@<commit>.tar.gz
// <cacheDir>/cache/download/<repo>@<commit>.info
// <cacheDir>/<repo>@<commit>/<files…>
func DownloadAndExtractTarball(repo, commit, cacheDir string) (string, error) {
	repoPath := filepath.Join(strings.Split(repo, "/")...)
	extractDir := filepath.Join(cacheDir, repoPath, fmt.Sprintf("%s@%s", filepath.Base(repo), commit))
	if hasFiles(extractDir) {
		return extractDir, nil
	}

	downloadDir := filepath.Join(cacheDir, "download", repoPath)
	tarballPath := filepath.Join(downloadDir, fmt.Sprintf("%s@%s.tar.gz", filepath.Base(repo), commit))
	infoPath := filepath.Join(downloadDir, fmt.Sprintf("%s@%s.info", filepath.Base(repo), commit))

	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", downloadDir, err)
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", extractDir, err)
	}

	needDownload := true
	if _, err := os.Stat(tarballPath); err == nil {
		b, err := os.ReadFile(infoPath)
		if err != nil {
			return "", err
		}
		var info Info
		err = json.Unmarshal(b, &info)
		if err != nil {
			return "", err
		}

		sum, err := computeSHA256(tarballPath)
		if err != nil {
			return "", err
		}
		if sum == info.SHA256 {
			needDownload = false
		}
	}
	if needDownload {
		sourceURL := fmt.Sprintf("https://%s/archive/%s.tar.gz", repo, commit)
		if err := downloadTarballWithInfo(sourceURL, tarballPath, infoPath); err != nil {
			return "", err
		}
	}
	if err := extractTarballFlattened(tarballPath, extractDir); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}
	return extractDir, nil
}

func hasFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

func computeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func downloadTarballWithInfo(sourceURL, tarballPath, infoPath string) error {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("failed downloading tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tarball download failed: HTTP %d %s (%q)", resp.StatusCode, resp.Status, sourceURL)
	}

	h := sha256.New()
	out, err := os.Create(tarballPath)
	if err != nil {
		return fmt.Errorf("failed creating tarball file: %w", err)
	}

	tee := io.TeeReader(resp.Body, h)
	if _, err := io.Copy(out, tee); err != nil {
		out.Close()
		return fmt.Errorf("failed writing tarball: %w", err)
	}
	out.Close()

	sha := fmt.Sprintf("%x", h.Sum(nil))
	info := Info{SHA256: sha}
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(infoPath), 0o755); err != nil {
		return fmt.Errorf("failed to create info dir: %w", err)
	}
	if err := os.WriteFile(infoPath, b, 0o644); err != nil {
		return fmt.Errorf("failed to write .info file: %w", err)
	}
	return nil
}

func extractTarballFlattened(tarballPath, destDir string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Remove the GitHub top-level "repo-<commit>/" prefix
		name := hdr.Name
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			name = parts[1]
		} else {
			continue
		}

		target := filepath.Join(destDir, name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}

			out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}

			out.Close()
		}
	}
}
