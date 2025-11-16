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
	"regexp"
	"strings"

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
func (r *Reader) ReadAll() (*LegacyState, *LegacyConfig, *BuildBazelData, *LegacyGeneratorInputData, error) {
	state, err := r.readState()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to read state.yaml: %w", err)
	}

	config, err := r.readConfig()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	buildData, err := r.readBuildBazel(state)
	if err != nil {
		// Log warning but continue - BUILD.bazel data is optional
		fmt.Fprintf(os.Stderr, "Warning: failed to read BUILD.bazel files: %v\n", err)
		buildData = &BuildBazelData{Libraries: make(map[string]*BuildLibrary)}
	}

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
func (r *Reader) readBuildBazel(state *LegacyState) (*BuildBazelData, error) {
	if r.GoogleapisPath == "" {
		return &BuildBazelData{Libraries: make(map[string]*BuildLibrary)}, nil
	}

	data := &BuildBazelData{Libraries: make(map[string]*BuildLibrary)}

	for _, lib := range state.Libraries {
		if len(lib.APIs) == 0 {
			continue
		}

		// Use the first API to locate the BUILD.bazel file
		apiPath := lib.APIs[0].Path
		buildPath := filepath.Join(r.GoogleapisPath, apiPath, "BUILD.bazel")

		buildLib, err := r.parseBuildBazel(buildPath, lib.ID)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", buildPath, err)
			continue
		}

		data.Libraries[lib.ID] = buildLib
	}

	return data, nil
}

// parseBuildBazel parses a BUILD.bazel file and extracts py_gapic_library data.
func (r *Reader) parseBuildBazel(path, libID string) (*BuildLibrary, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("BUILD.bazel not found")
		}
		return nil, err
	}

	text := string(content)
	lib := &BuildLibrary{ID: libID}

	// Extract py_gapic_library section
	// Look for py_gapic_library( ... )
	pyGapicStart := strings.Index(text, "py_gapic_library(")
	if pyGapicStart == -1 {
		return lib, nil
	}

	// Find the matching closing parenthesis
	pyGapicSection := r.extractSection(text[pyGapicStart:])

	// Extract transport
	lib.Transport = r.extractField(pyGapicSection, "transport")

	// Extract opt_args
	lib.OptArgs = r.extractListField(pyGapicSection, "opt_args")

	// Extract service_yaml
	lib.ServiceYAML = r.extractField(pyGapicSection, "service_yaml")

	return lib, nil
}

// extractSection extracts a balanced parenthetical section.
func (r *Reader) extractSection(text string) string {
	depth := 0
	for i, ch := range text {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				return text[:i+1]
			}
		}
	}
	return text
}

// extractField extracts a simple field value from BUILD.bazel content.
// Example: transport = "grpc+rest" -> returns "grpc+rest"
func (r *Reader) extractField(content, fieldName string) string {
	pattern := fmt.Sprintf(`%s\s*=\s*"([^"]*)"`, regexp.QuoteMeta(fieldName))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractListField extracts a list field value from BUILD.bazel content.
// Example: opt_args = ["a", "b"] -> returns ["a", "b"]
func (r *Reader) extractListField(content, fieldName string) []string {
	pattern := fmt.Sprintf(`%s\s*=\s*\[(.*?)\]`, regexp.QuoteMeta(fieldName))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil
	}

	listContent := matches[1]
	// Extract quoted strings
	stringPattern := `"([^"]*)"`
	stringRe := regexp.MustCompile(stringPattern)
	stringMatches := stringRe.FindAllStringSubmatch(listContent, -1)

	var result []string
	for _, match := range stringMatches {
		if len(match) > 1 {
			result = append(result, match[1])
		}
	}
	return result
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
