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

package naming

import (
	"regexp"
	"strings"
)

// ParseAPIPath extracts service, namespace, and version from an API path.
// Examples:
//   - google/cloud/secretmanager/v1 → (secretmanager, cloud, v1)
//   - google/ai/generativelanguage/v1beta → (generativelanguage, ai, v1beta)
//   - google/type → (type, "", "")
func ParseAPIPath(apiPath string) (service, namespace, version string) {
	// Remove leading/trailing slashes
	apiPath = strings.Trim(apiPath, "/")

	// Split into parts
	parts := strings.Split(apiPath, "/")

	// Remove "google" prefix if present
	if len(parts) > 0 && parts[0] == "google" {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		return "", "", ""
	}

	// Check if last part is a version
	versionRegex := regexp.MustCompile(`^v\d+(alpha\d*|beta\d*)?$`)
	if versionRegex.MatchString(parts[len(parts)-1]) {
		version = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}

	// Extract namespace and service
	switch len(parts) {
	case 0:
		// Only version (unlikely but handle it)
		return "", "", version
	case 1:
		// No namespace: google/{service}/{version}
		service = parts[0]
		return service, "", version
	default:
		// Has namespace: google/{namespace}/.../{service}/{version}
		// The namespace is the first part, service is the last part before version
		namespace = parts[0]
		service = parts[len(parts)-1]
		return service, namespace, version
	}
}
