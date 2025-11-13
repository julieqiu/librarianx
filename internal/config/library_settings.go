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
	"strings"
)

// DeriveLaunchStage derives the launch stage from an API version string.
// Uses heuristics:
//   - v1, v2, etc. → GA
//   - v1beta*, v2beta* → BETA
//   - v1alpha*, v2alpha* → ALPHA
//   - Otherwise → GA (default)
//
// Returns empty string if no override is needed (should use default).
func DeriveLaunchStage(version string) string {
	// Extract just the version part (e.g., "v1beta1" from "google.cloud.secretmanager.v1beta1")
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return "GA"
	}
	versionPart := parts[len(parts)-1]

	if strings.Contains(versionPart, "alpha") {
		return "ALPHA"
	}
	if strings.Contains(versionPart, "beta") {
		return "BETA"
	}
	// v1, v2, etc. default to GA
	return "GA"
}

// DeriveDestinations returns the default destinations for a library.
// Returns [PACKAGE_MANAGER] unless overridden in the config.
func DeriveDestinations(cfg *LibraryConfig) []string {
	if cfg != nil && len(cfg.Destinations) > 0 {
		return cfg.Destinations
	}
	return []string{"PACKAGE_MANAGER"}
}

// GetLaunchStage returns the launch stage for a library, using explicit
// override if present, otherwise deriving from the API version.
func GetLaunchStage(cfg *LibraryConfig, apiVersion string) string {
	if cfg != nil && cfg.LaunchStage != "" {
		return cfg.LaunchStage
	}
	return DeriveLaunchStage(apiVersion)
}

// GetJavaPackage returns the Java package name for a library.
// Returns empty string if not configured (should use default package naming).
func GetJavaPackage(cfg *LibraryConfig) string {
	if cfg != nil && cfg.Java != nil {
		return cfg.Java.Package
	}
	return ""
}

// GetJavaServiceClassNames returns the Java service class name mappings.
// Returns nil if not configured (should use default class naming).
func GetJavaServiceClassNames(cfg *LibraryConfig) map[string]string {
	if cfg != nil && cfg.Java != nil {
		return cfg.Java.ServiceClassNames
	}
	return nil
}

// GetPythonRestAsyncIOEnabled returns whether Python REST async I/O is enabled.
func GetPythonRestAsyncIOEnabled(cfg *LibraryConfig) bool {
	if cfg != nil && cfg.Python != nil {
		return cfg.Python.RestAsyncIOEnabled
	}
	return false
}

// GetPythonUnversionedPackageDisabled returns whether Python unversioned package is disabled.
func GetPythonUnversionedPackageDisabled(cfg *LibraryConfig) bool {
	if cfg != nil && cfg.Python != nil {
		return cfg.Python.UnversionedPackageDisabled
	}
	return false
}

// GetGoRenamedServices returns the Go service name mappings.
// Returns nil if not configured (should use default service naming).
func GetGoRenamedServices(cfg *LibraryConfig) map[string]string {
	if cfg != nil && cfg.Go != nil {
		return cfg.Go.RenamedServices
	}
	return nil
}

// GetNodeSelectiveMethods returns the Node.js selective method list.
// Returns nil if not configured (should generate all methods).
func GetNodeSelectiveMethods(cfg *LibraryConfig) []string {
	if cfg != nil && cfg.Node != nil {
		return cfg.Node.SelectiveMethods
	}
	return nil
}

// GetDotnetRenamedServices returns the .NET service name mappings.
// Returns nil if not configured (should use default service naming).
func GetDotnetRenamedServices(cfg *LibraryConfig) map[string]string {
	if cfg != nil && cfg.Dotnet != nil {
		return cfg.Dotnet.RenamedServices
	}
	return nil
}

// GetDotnetRenamedResources returns the .NET resource name mappings.
// Returns nil if not configured (should use default resource naming).
func GetDotnetRenamedResources(cfg *LibraryConfig) map[string]string {
	if cfg != nil && cfg.Dotnet != nil {
		return cfg.Dotnet.RenamedResources
	}
	return nil
}
