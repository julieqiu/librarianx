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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"gopkg.in/yaml.v3"
)

// Reader reads all legacy configuration sources.
type Reader struct {
	// RepoPath is the path to the google-cloud-python repository.
	RepoPath string

	// GoogleapisPath is the path to the googleapis repository.
	GoogleapisPath string
}

// ReadAll reads all configuration sources and returns the parsed data.
func (r *Reader) ReadAll(language string) (*LegacyState, *LegacyConfig, *BuildBazelData, *LegacyGeneratorInputData, error) {
	state, err := r.readState()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to read state.yaml: %w", err)
	}

	config, err := r.readConfig()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	buildData := r.readBuildBazel(state, language)

	generatorInput, err := r.readGeneratorInput()
	if err != nil {
		// Log warning but continue - generator input is optional
		fmt.Fprintf(os.Stderr, "Warning: failed to read generator-input: %v\n", err)
		generatorInput = &LegacyGeneratorInputData{Files: make(map[string][]byte)}
	}
	return state, config, buildData, generatorInput, nil
}

// readState reads .librarian/state.yaml.
func (r *Reader) readState() (*LegacyState, error) {
	path := filepath.Join(r.RepoPath, ".librarian", "state.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var state LegacyState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state.yaml: %w", err)
	}

	return &state, nil
}

// readConfig reads .librarian/config.yaml.
func (r *Reader) readConfig() (*LegacyConfig, error) {
	path := filepath.Join(r.RepoPath, ".librarian", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		// config.yaml is optional
		if os.IsNotExist(err) {
			return &LegacyConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var config LegacyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config.yaml: %w", err)
	}

	return &config, nil
}

// readBuildBazel reads BUILD.bazel files from googleapis for each library.
func (r *Reader) readBuildBazel(state *LegacyState, language string) *BuildBazelData {
	if r.GoogleapisPath == "" {
		return &BuildBazelData{Libraries: make(map[string]*BuildLibrary)}
	}

	data := &BuildBazelData{Libraries: make(map[string]*BuildLibrary)}

	for _, lib := range state.Libraries {
		if len(lib.APIs) == 0 {
			continue
		}

		// Use the first API to locate the BUILD.bazel file
		apiPath := lib.APIs[0].Path

		// Use the new config.ReadBuildBazel function
		bazelConfig, err := config.ReadBuildBazel(r.GoogleapisPath, apiPath, language)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse BUILD.bazel for %s: %v\n", apiPath, err)
			continue
		}

		// Convert config.BazelConfig to BuildLibrary
		buildLib := &BuildLibrary{
			ID:                lib.ID,
			Transport:         bazelConfig.Transport,
			OptArgs:           bazelConfig.OptArgs,
			ServiceYAML:       bazelConfig.ServiceYAML,
			GRPCServiceConfig: bazelConfig.GRPCServiceConfig,
			RestNumericEnums:  bazelConfig.RestNumericEnums,
			IsProtoOnly:       bazelConfig.IsProtoOnly,
			ImportPath:        bazelConfig.ImportPath,
			Metadata:          bazelConfig.Metadata,
			ReleaseLevel:      bazelConfig.ReleaseLevel,
		}

		data.Libraries[lib.ID] = buildLib
	}

	return data
}

// readGeneratorInput reads files from .librarian/generator-input/.
func (r *Reader) readGeneratorInput() (*LegacyGeneratorInputData, error) {
	inputDir := filepath.Join(r.RepoPath, ".librarian", "generator-input")

	// Check if directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return &LegacyGeneratorInputData{Files: make(map[string][]byte)}, nil
	}

	data := &LegacyGeneratorInputData{Files: make(map[string][]byte)}

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		data.Files[relPath] = content
		return nil
	})

	if err != nil {
		return nil, err
	}

	return data, nil
}
