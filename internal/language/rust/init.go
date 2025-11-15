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

import (
	"fmt"
	"os"

	"github.com/googleapis/librarian/internal/config"
)

// Init initializes a Rust repository with pinned source dependencies.
func Init(version string, all bool) error {
	// Check if librarian.yaml already exists
	const configPath = "librarian.yaml"
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("librarian.yaml already exists in current directory")
	}

	// Create config with pinned Rust sources
	cfg := &config.Config{
		Version:  version,
		Language: "rust",
		Sources: config.Sources{
			Googleapis: &config.Source{
				URL:    "https://github.com/googleapis/googleapis/archive/9fcfbea0aa5b50fa22e190faceb073d74504172b.tar.gz",
				SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
			},
			Discovery: &config.Source{
				URL:    "https://github.com/googleapis/discovery-artifact-manager/archive/b27c80574e918a7e2a36eb21864d1d2e45b8c032.tar.gz",
				SHA256: "67c8d3792f0ebf5f0582dce675c379d0f486604eb0143814c79e788954aa1212",
			},
			ProtobufSrc: &config.Source{
				URL:    "https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protobuf-29.3.tar.gz",
				SHA256: "008a11cc56f9b96679b4c285fd05f46d317d685be3ab524b2a310be0fbad987e",
			},
			Conformance: &config.Source{
				URL:    "https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protobuf-29.3.tar.gz",
				SHA256: "008a11cc56f9b96679b4c285fd05f46d317d685be3ab524b2a310be0fbad987e",
			},
		},
		Release: &config.Release{
			TagFormat: "{name}/v{version}",
		},
	}

	// Add wildcard library and Rust defaults if --all is specified
	if all {
		cfg.Libraries = []config.LibraryEntry{
			{Name: "*", Config: nil},
		}
		cfg.Defaults = &config.Defaults{
			Output:        "src/generated/",
			OneLibraryPer: "version",
			ReleaseLevel:  "stable",
			Rust: &config.RustDefaults{
				DisabledRustdocWarnings: []string{
					"redundant_explicit_links",
					"broken_intra_doc_links",
				},
			},
		}
	}

	// Write config to librarian.yaml
	if err := cfg.Write(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Created librarian.yaml\n")
	return nil
}
