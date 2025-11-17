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

package rust

import "strings"

// DeriveLibraryName converts an API path to the standard Rust library name.
// Simply replaces / with -.
// Examples:
//   - google/cloud/secretmanager/v1 → google-cloud-secretmanager-v1
//   - google/api/apikeys/v2 → google-api-apikeys-v2
//   - grafeas/v1 → grafeas-v1
//
// Special cases (like google/cloud/translate/v3 → google-cloud-translation-v3)
// should be handled via name_overrides in librarian.yaml.
func DeriveLibraryName(apiPath string) string {
	return strings.ReplaceAll(apiPath, "/", "-")
}

// DeriveAPIPath converts a Rust library name to the likely API path.
// Simply replaces - with /.
// Examples:
//   - google-cloud-secretmanager-v1 → google/cloud/secretmanager/v1
//   - google-api-apikeys-v2 → google/api/apikeys/v2
//   - grafeas-v1 → grafeas/v1
func DeriveAPIPath(libraryName string) string {
	return strings.ReplaceAll(libraryName, "-", "/")
}
