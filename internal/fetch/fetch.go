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
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

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
