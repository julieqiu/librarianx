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
	"strings"
)

// DeriveLibraryName derives a library name from an API path based on language conventions.
// The derivation follows the rules in doc/config.md:
//
// Python (service-level): google/cloud/secretmanager/v1 → google-cloud-secretmanager
// Go (service-level): google/cloud/secretmanager/v1 → secretmanager
// Rust (version-level): google/cloud/secretmanager/v1 → google-cloud-secretmanager-v1.
// Dart (version-level): google/cloud/secretmanager/v1 → google_cloud_secretmanager_v1.
func DeriveLibraryName(apiPath, language, packaging string) string {
	switch language {
	case "python":
		return derivePythonName(apiPath, packaging)
	case "go":
		return deriveGoName(apiPath, packaging)
	case "rust":
		return deriveRustName(apiPath, packaging)
	case "dart":
		return deriveDartName(apiPath, packaging)
	default:
		// Fallback: use the last segment before version
		service, _, _ := ParseAPIPath(apiPath)
		return service
	}
}

// derivePythonName derives a Python library name from an API path.
// Service-level packaging: google/cloud/secretmanager/v1 → google-cloud-secretmanager.
// Version-level packaging: google/cloud/secretmanager/v1 → google-cloud-secretmanager-v1.
func derivePythonName(apiPath, packaging string) string {
	service, namespace, version := ParseAPIPath(apiPath)

	// Build name: google-{namespace}-{service}
	var parts []string
	parts = append(parts, "google")
	if namespace != "" {
		parts = append(parts, namespace)
	}
	parts = append(parts, service)

	name := strings.Join(parts, "-")

	// Add version for version-level packaging
	if packaging == "version" && version != "" {
		name = name + "-" + version
	}

	return name
}

// deriveGoName derives a Go library name from an API path.
// Service-level packaging: google/cloud/secretmanager/v1 → secretmanager.
// Version-level packaging: google/cloud/secretmanager/v1 → secretmanager-v1.
func deriveGoName(apiPath, packaging string) string {
	service, _, version := ParseAPIPath(apiPath)

	// For service-level packaging, just use the service name
	if packaging != "version" {
		return service
	}

	// For version-level packaging, include version
	if version != "" {
		return service + "-" + version
	}
	return service
}

// deriveRustName derives a Rust library name from an API path.
// Version-level packaging: google/cloud/secretmanager/v1 → google-cloud-secretmanager-v1.
func deriveRustName(apiPath, _ string) string {
	service, namespace, version := ParseAPIPath(apiPath)

	// Build name: google-{namespace}-{service}-{version}
	var parts []string
	parts = append(parts, "google")
	if namespace != "" {
		parts = append(parts, namespace)
	}
	parts = append(parts, service)
	if version != "" {
		parts = append(parts, version)
	}

	return strings.Join(parts, "-")
}

// deriveDartName derives a Dart library name from an API path.
// Version-level packaging: google/cloud/secretmanager/v1 → google_cloud_secretmanager_v1.
func deriveDartName(apiPath, _ string) string {
	service, namespace, version := ParseAPIPath(apiPath)

	// Build name: google_{namespace}_{service}_{version}
	var parts []string
	parts = append(parts, "google")
	if namespace != "" {
		parts = append(parts, namespace)
	}
	parts = append(parts, service)
	if version != "" {
		parts = append(parts, version)
	}

	return strings.Join(parts, "_")
}
