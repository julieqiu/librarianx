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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/parser"
)

// RepoMetadata represents the .repo-metadata.json file structure.
type RepoMetadata struct {
	Name                 string `json:"name,omitempty"`
	NamePretty           string `json:"name_pretty"`
	ProductDocumentation string `json:"product_documentation,omitempty"`
	ClientDocumentation  string `json:"client_documentation"`
	IssueTracker         string `json:"issue_tracker,omitempty"`
	ReleaseLevel         string `json:"release_level"`
	Language             string `json:"language"`
	LibraryType          string `json:"library_type"`
	Repo                 string `json:"repo"`
	DistributionName     string `json:"distribution_name"`
	APIID                string `json:"api_id"`
	DefaultVersion       string `json:"default_version,omitempty"`
	APIShortname         string `json:"api_shortname"`
	APIDescription       string `json:"api_description,omitempty"`
}

// GenerateRepoMetadata generates the .repo-metadata.json file by parsing the service YAML.
func GenerateRepoMetadata(library *Library, language, repo, serviceConfigPath, outdir string, apiPaths []string) error {
	// Use sidekick's service config parser
	svcCfg, err := parser.ReadServiceConfig(serviceConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read service config: %w", err)
	}

	// Select the default version from all API paths, preferring stable versions
	defaultVersion := SelectDefaultVersion(apiPaths)

	// Build client documentation URL based on language
	clientDocURL := buildClientDocURL(language, extractNameFromAPIID(svcCfg.GetName()))

	// Create metadata
	metadata := &RepoMetadata{
		APIID:                svcCfg.GetName(),
		NamePretty:           CleanTitle(svcCfg.GetTitle()),
		ClientDocumentation:  clientDocURL,
		ReleaseLevel:         library.ReleaseLevel,
		Language:             language,
		LibraryType:          "GAPIC_AUTO",
		Repo:                 repo,
		DistributionName:     library.Name,
		DefaultVersion:       defaultVersion,
	}

	// Add optional fields if available
	if svcCfg.GetPublishing() != nil {
		publishing := svcCfg.GetPublishing()
		if publishing.GetDocumentationUri() != "" {
			metadata.ProductDocumentation = publishing.GetDocumentationUri()
		}
		if publishing.GetApiShortName() != "" {
			metadata.APIShortname = publishing.GetApiShortName()
			metadata.Name = publishing.GetApiShortName()
		}
	}

	if svcCfg.GetDocumentation() != nil && svcCfg.GetDocumentation().GetSummary() != "" {
		metadata.APIDescription = strings.TrimSpace(svcCfg.GetDocumentation().GetSummary())
	}

	// Set default release level if not specified
	if metadata.ReleaseLevel == "" {
		metadata.ReleaseLevel = "stable"
	}

	// Write metadata file
	data, err := json.MarshalIndent(metadata, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := filepath.Join(outdir, ".repo-metadata.json")
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// buildClientDocURL builds the client documentation URL based on language.
func buildClientDocURL(language, serviceName string) string {
	switch language {
	case "python":
		return fmt.Sprintf("https://cloud.google.com/python/docs/reference/%s/latest", serviceName)
	case "rust":
		// Rust uses docs.rs
		return fmt.Sprintf("https://docs.rs/google-cloud-%s/latest", serviceName)
	default:
		return ""
	}
}

// SelectDefaultVersion selects the best default version from a list of API paths.
// It prefers stable versions (v1, v2) over beta/alpha versions (v1beta1, v1alpha1).
// Among stable versions, it selects the highest. Among beta versions, it selects the highest.
func SelectDefaultVersion(apiPaths []string) string {
	if len(apiPaths) == 0 {
		return ""
	}

	var stableVersions []string
	var betaVersions []string

	for _, apiPath := range apiPaths {
		version := DeriveDefaultVersion(apiPath)
		if version == "" {
			continue
		}
		// Check if it's a stable version (vN where N is just digits)
		if isStableVersion(version) {
			stableVersions = append(stableVersions, version)
		} else {
			betaVersions = append(betaVersions, version)
		}
	}

	// Prefer stable versions
	if len(stableVersions) > 0 {
		return selectHighestVersion(stableVersions)
	}
	if len(betaVersions) > 0 {
		return selectHighestVersion(betaVersions)
	}
	return ""
}

// isStableVersion returns true if the version is stable (e.g., v1, v2) and not beta/alpha.
func isStableVersion(version string) bool {
	// Strip the "v" prefix
	if !strings.HasPrefix(version, "v") {
		return false
	}
	versionNum := strings.TrimPrefix(version, "v")
	// Check if it contains only digits (stable) or has alpha/beta (not stable)
	for _, r := range versionNum {
		if !('0' <= r && r <= '9') {
			return false
		}
	}
	return true
}

// selectHighestVersion selects the highest version from a list of versions.
// For example, given ["v1", "v2", "v3"], it returns "v3".
// For beta versions like ["v1beta1", "v1beta2"], it returns "v1beta2".
func selectHighestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	if len(versions) == 1 {
		return versions[0]
	}

	// Simple lexicographic comparison works for most cases
	// v2 > v1, v1beta2 > v1beta1, etc.
	highest := versions[0]
	for _, v := range versions[1:] {
		if v > highest {
			highest = v
		}
	}
	return highest
}

// DeriveDefaultVersion extracts the version from an API path.
// Example: "google/cloud/secretmanager/v1" -> "v1"
func DeriveDefaultVersion(apiPath string) string {
	parts := strings.Split(apiPath, "/")
	if len(parts) == 0 {
		return ""
	}
	lastPart := parts[len(parts)-1]
	// Check if it looks like a version (v1, v1beta1, etc.)
	if strings.HasPrefix(lastPart, "v") {
		return lastPart
	}
	return ""
}

// CleanTitle removes "API" suffix from title to get name_pretty.
// Example: "Secret Manager API" -> "Secret Manager"
func CleanTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.TrimSuffix(title, " API")
	return strings.TrimSpace(title)
}

// extractNameFromAPIID extracts the service name from the API ID.
// Example: "secretmanager.googleapis.com" -> "secretmanager"
func extractNameFromAPIID(apiID string) string {
	parts := strings.Split(apiID, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return apiID
}
