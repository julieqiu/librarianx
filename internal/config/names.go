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
	"fmt"
	"strings"
)

// DeriveLibraryName converts an API path to a library name based on one_library_per mode.
//   - "channel": Each API version gets its own library (Rust, Dart)
//     Example: google/cloud/secretmanager/v1 → google-cloud-secretmanager-v1
//   - "api": All versions bundled into one library (Python, Go)
//     Example: google/cloud/secretmanager/v1 → google-cloud-secretmanager
func DeriveLibraryName(oneLibraryPer, apiPath string) (string, error) {
	switch oneLibraryPer {
	case "channel":
		return strings.ReplaceAll(apiPath, "/", "-"), nil
	case "api":
		name := strings.ReplaceAll(apiPath, "/", "-")
		return stripVersionSuffix(name), nil
	default:
		return "", fmt.Errorf("unsupported one_library_per mode: %q (must be \"channel\" or \"api\")", oneLibraryPer)
	}
}

// DeriveAPIPath converts a library name to an API path based on one_library_per mode.
// Note: api mode can only derive api path, not full path with version.
// - "channel": google-cloud-secretmanager-v1 → google/cloud/secretmanager/v1.
// - "api": google-cloud-secretmanager → google/cloud/secretmanager.
func DeriveAPIPath(oneLibraryPer, libraryName string) (string, error) {
	if oneLibraryPer != "channel" && oneLibraryPer != "api" {
		return "", fmt.Errorf("unsupported one_library_per mode: %q (must be \"channel\" or \"api\")", oneLibraryPer)
	}
	// Both modes: replace - with /
	// Note: api mode can only derive api path, not full path with version
	return strings.ReplaceAll(libraryName, "-", "/"), nil
}

func stripVersionSuffix(name string) string {
	parts := strings.Split(name, "-")
	if len(parts) == 0 {
		return name
	}

	lastPart := parts[len(parts)-1]
	if isVersionSuffix(lastPart) {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return name
}

func isVersionSuffix(s string) bool {
	return len(s) > 0 && s[0] == 'v' && len(s) > 1 && (s[1] >= '0' && s[1] <= '9')
}
