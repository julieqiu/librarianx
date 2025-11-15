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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// GetSha256 computes the SHA256 checksum of a file downloaded from the given URL.
func GetSha256(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, response.Body); err != nil {
		return "", err
	}
	got := fmt.Sprintf("%x", hasher.Sum(nil))
	return got, nil
}

// GetLatestSha fetches the latest commit SHA from GitHub API.
func GetLatestSha(url string) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github.VERSION.sha")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

// LatestGoogleapis fetches the latest commit SHA from googleapis/googleapis
// and computes the SHA256 checksum for the tarball.
func LatestGoogleapis() (*config.Source, error) {
	latestSha, err := GetLatestSha("https://api.github.com/repos/googleapis/googleapis/commits/master")
	if err != nil {
		return nil, fmt.Errorf("failed to get latest SHA: %w", err)
	}

	tarballURL := fmt.Sprintf("https://github.com/googleapis/googleapis/archive/%s.tar.gz", latestSha)

	// Compute SHA256 checksum
	sha256sum, err := GetSha256(tarballURL)
	if err != nil {
		return nil, fmt.Errorf("failed to compute SHA256: %w", err)
	}

	return &config.Source{
		URL:    tarballURL,
		SHA256: sha256sum,
	}, nil
}

// DownloadAndExtractTarball downloads a tarball from the given source,
// verifies its SHA256 checksum, and extracts it to a temporary directory.
// It returns the path to the extracted googleapis directory.
// The caller is responsible for cleaning up the temporary directory.
func DownloadAndExtractTarball(source *config.Source) (string, error) {
	// Create a temporary directory for extraction
	tmpDir, err := os.MkdirTemp("", "googleapis-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Download the tarball
	resp, err := http.Get(source.URL)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download tarball: HTTP %d - %s", resp.StatusCode, resp.Status)
	}

	// Verify SHA256
	hasher := sha256.New()
	teeReader := io.TeeReader(resp.Body, hasher)

	tarballPath := filepath.Join(tmpDir, "source.tar.gz")
	file, err := os.Create(tarballPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create tarball file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, teeReader); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write tarball to file: %w", err)
	}

	if fmt.Sprintf("%x", hasher.Sum(nil)) != source.SHA256 {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("SHA256 checksum mismatch for %s", source.URL)
	}

	// Extract the tarball
	if err := extractTarball(tarballPath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}

	// The tarball usually extracts into a directory named after the archive, e.g., googleapis-sha
	// Find the actual extracted directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to read temp directory after extraction: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "googleapis-") {
			return filepath.Join(tmpDir, entry.Name()), nil
		}
	}

	os.RemoveAll(tmpDir)
	return "", fmt.Errorf("could not find extracted googleapis directory in %s", tmpDir)
}

// extractTarball extracts a gzipped tarball to a destination directory.
func extractTarball(tarballPath, destDir string) error {
	file, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}

	return nil
}
