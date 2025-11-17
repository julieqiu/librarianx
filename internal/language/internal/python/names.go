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

import "strings"

// DeriveLibraryName converts an API path to the standard Python library name.
// Python uses service-level naming (strips version).
// Examples:
//   - google/cloud/secretmanager/v1 → google-cloud-secretmanager
//   - google/cloud/secretmanager/v2 → google-cloud-secretmanager
//   - google/api/apikeys/v2 → google-api-apikeys
func DeriveLibraryName(apiPath string) string {
	parts := strings.Split(apiPath, "/")
	if len(parts) == 0 {
		return apiPath
	}

	// Remove version if present (last part starting with 'v' followed by digit)
	lastPart := parts[len(parts)-1]
	if len(lastPart) > 0 && lastPart[0] == 'v' && len(lastPart) > 1 && (lastPart[1] >= '0' && lastPart[1] <= '9') {
		parts = parts[:len(parts)-1]
	}

	return strings.ReplaceAll(strings.Join(parts, "/"), "/", "-")
}

// DeriveAPIPath converts a Python library name to the likely API path (service path only).
// Python library names don't include version, so we can only derive the service path.
// Examples:
//   - google-cloud-secretmanager → google/cloud/secretmanager
//   - google-api-apikeys → google/api/apikeys
func DeriveAPIPath(libraryName string) string {
	return strings.ReplaceAll(libraryName, "-", "/")
}
