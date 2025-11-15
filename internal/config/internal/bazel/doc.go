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

/*
Package bazel provides utilities for parsing Bazel BUILD files to extract API metadata.

# Generic Parser

The generic parser (ParseAPI) supports multiple languages and extracts configuration
from *_gapic_library rules in googleapis BUILD.bazel files.

Supported languages:
  - go (go_gapic_library)
  - python (py_gapic_library)
  - More languages can be added by extending parseXXXGapic functions

Example usage:

	// Parse Python configuration
	cfg, err := bazel.ParseAPI(googleapisRoot, "google/cloud/secretmanager/v1", "python")
	if err != nil {
	    log.Fatal(err)
	}

	if cfg.HasGAPIC {
	    fmt.Printf("Transport: %s\n", cfg.Transport)
	    fmt.Printf("Service YAML: %s\n", cfg.ServiceYAML)
	    fmt.Printf("Python opt_args: %v\n", cfg.Python.OptArgs)
	}

	// Parse Go configuration
	cfg, err := bazel.ParseAPI(googleapisRoot, "google/cloud/asset/v1", "go")
	if err != nil {
	    log.Fatal(err)
	}

	if cfg.HasGAPIC {
	    fmt.Printf("Import path: %s\n", cfg.Go.ImportPath)
	    fmt.Printf("Release level: %s\n", cfg.ReleaseLevel)
	}

# Language-Specific Parser (Go only)

For backward compatibility, the Parse function provides Go-specific parsing:

	cfg, err := bazel.Parse(dir)
	if err != nil {
	    log.Fatal(err)
	}

	if cfg.HasGAPIC() {
	    fmt.Printf("GAPIC import path: %s\n", cfg.GAPICImportPath())
	}

# Common Fields

The following fields are extracted across all supported languages:
  - GRPCServiceConfig: gRPC service configuration JSON file
  - ServiceYAML: API service configuration YAML file
  - Transport: Transport protocol (e.g., "grpc", "rest", "grpc+rest")
  - RestNumericEnums: Whether to use numeric enums in REST
  - ReleaseLevel: Release level (e.g., "ga", "beta")

# Language-Specific Fields

Go:
  - ImportPath: Go package import path
  - Metadata: Whether to generate gapic_metadata.json
  - Diregapic: Whether this is a DIREGAPIC (Discovery REST GAPIC)
  - HasGoGRPC: Whether go_grpc_library is used
  - HasLegacyGRPC: Whether legacy gRPC protoc plugin is used

Python:
  - OptArgs: Additional generator options (e.g., warehouse-package-name)

# Adding Support for New Languages

To add support for a new language:

1. Add a language-specific config struct (e.g., JavaConfig)
2. Add a field to APIConfig (e.g., Java *JavaConfig)
3. Implement parseXXXGapic function
4. Update ParseAPI switch statement
5. Add comprehensive tests

Example:

	type JavaConfig struct {
	    // Java-specific fields
	}

	func parseJavaGapic(content string, cfg *APIConfig) error {
	    re := regexp.MustCompile(`java_gapic_library\((?s:.)*?\)`)
	    gapicBlock := re.FindString(content)
	    if gapicBlock == "" {
	        cfg.HasGAPIC = false
	        return nil
	    }

	    cfg.HasGAPIC = true
	    cfg.Java = &JavaConfig{}

	    // Extract common fields
	    cfg.GRPCServiceConfig = findString(gapicBlock, "grpc_service_config")
	    // ... extract other fields

	    return nil
	}
*/
package bazel
