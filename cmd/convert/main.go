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

// Command convert converts .librarian/state.yaml to librarian.yaml format.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Old format structures
type OldState struct {
	Image     string       `yaml:"image"`
	Libraries []OldLibrary `yaml:"libraries"`
}

type OldLibrary struct {
	ID                  string   `yaml:"id"`
	Version             string   `yaml:"version"`
	LastGeneratedCommit string   `yaml:"last_generated_commit"`
	APIs                []OldAPI `yaml:"apis"`
	SourceRoots         []string `yaml:"source_roots"`
	PreserveRegex       []string `yaml:"preserve_regex"`
	RemoveRegex         []string `yaml:"remove_regex"`
	ReleaseExcludePaths []string `yaml:"release_exclude_paths"`
	TagFormat           string   `yaml:"tag_format"`
}

type OldAPI struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config"`
}

type OldConfig struct {
	GlobalFilesAllowlist []any        `yaml:"global_files_allowlist"`
	Libraries            []OldConfigLib `yaml:"libraries"`
}

type OldConfigLib struct {
	ID             string `yaml:"id"`
	ReleaseBlocked bool   `yaml:"release_blocked"`
}

// New format structures
type NewConfig struct {
	Version   string          `yaml:"version"`
	Language  string          `yaml:"language"`
	Container ContainerConfig `yaml:"container,omitempty"`
	Defaults  Defaults        `yaml:"defaults,omitempty"`
	Release   ReleaseConfig   `yaml:"release,omitempty"`
	Libraries []NewLibrary    `yaml:"libraries"`
}

type ContainerConfig struct {
	Image string `yaml:"image"`
	Tag   string `yaml:"tag"`
}

type Defaults struct {
	GeneratedDir     string `yaml:"generated_dir"`
	Transport        string `yaml:"transport"`
	RestNumericEnums bool   `yaml:"rest_numeric_enums"`
	ReleaseLevel     string `yaml:"release_level"`
}

type ReleaseConfig struct {
	TagFormat string `yaml:"tag_format"`
}

type NewLibrary struct {
	Name     string           `yaml:"name"`
	Version  string           `yaml:"version,omitempty"`
	Generate *LibraryGenerate `yaml:"generate,omitempty"`
}

type LibraryGenerate struct {
	APIs   []NewAPI `yaml:"apis"`
	Keep   []string `yaml:"keep,omitempty"`
	Remove []string `yaml:"remove,omitempty"`
}

type NewAPI struct {
	Path string `yaml:"path"`
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// Read state.yaml
	statePath := filepath.Join(homeDir, "code/googleapis/google-cloud-go/.librarian/state.yaml")
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading state.yaml: %v\n", err)
		os.Exit(1)
	}

	var oldState OldState
	if err := yaml.Unmarshal(stateData, &oldState); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing state.yaml: %v\n", err)
		os.Exit(1)
	}

	// Read config.yaml
	configPath := filepath.Join(homeDir, "code/googleapis/google-cloud-go/.librarian/config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config.yaml: %v\n", err)
		os.Exit(1)
	}

	var oldConfig OldConfig
	if err := yaml.Unmarshal(configData, &oldConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config.yaml: %v\n", err)
		os.Exit(1)
	}

	// Build release_blocked set
	releaseBlocked := make(map[string]bool)
	for _, lib := range oldConfig.Libraries {
		if lib.ReleaseBlocked {
			releaseBlocked[lib.ID] = true
		}
	}

	// Convert to new format
	newConfig := NewConfig{
		Version:  "v0.5.0",
		Language: "go",
		Container: ContainerConfig{
			Image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/librarian-go",
			Tag:   "latest",
		},
		Defaults: Defaults{
			GeneratedDir:     "./",
			Transport:        "grpc+rest",
			RestNumericEnums: true,
			ReleaseLevel:     "stable",
		},
		Release: ReleaseConfig{
			TagFormat: "{id}/v{version}",
		},
	}

	// Convert libraries
	for _, oldLib := range oldState.Libraries {
		newLib := NewLibrary{
			Name:    oldLib.ID,
			Version: oldLib.Version,
		}

		// Only add generate section if there are APIs
		if len(oldLib.APIs) > 0 {
			gen := &LibraryGenerate{
				APIs: make([]NewAPI, 0, len(oldLib.APIs)),
			}

			for _, api := range oldLib.APIs {
				gen.APIs = append(gen.APIs, NewAPI{
					Path: api.Path,
				})
			}

			if len(oldLib.PreserveRegex) > 0 {
				gen.Keep = oldLib.PreserveRegex
			}

			if len(oldLib.RemoveRegex) > 0 {
				gen.Remove = oldLib.RemoveRegex
			}

			newLib.Generate = gen
		}

		newConfig.Libraries = append(newConfig.Libraries, newLib)
	}

	// Marshal to YAML
	output, err := yaml.Marshal(newConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling output: %v\n", err)
		os.Exit(1)
	}

	// Write output
	outputPath := "data/go/librarian.yaml"
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Converted %d libraries to %s\n", len(newConfig.Libraries), outputPath)
	fmt.Printf("Release-blocked libraries: %d\n", len(releaseBlocked))
}
