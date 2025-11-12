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

package bazel

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// APIConfig holds generic configuration extracted from any *_gapic_library rule
// in a googleapis BUILD.bazel file. This structure is language-agnostic and
// supports Go, Python, Java, etc.
type APIConfig struct {
	// Language is the target language (go, python, java, etc.)
	Language string

	// HasGAPIC indicates whether a GAPIC library rule was found
	HasGAPIC bool

	// Common GAPIC fields (shared across languages)
	GRPCServiceConfig string
	ServiceYAML       string
	Transport         string
	RestNumericEnums  bool
	ReleaseLevel      string

	// Language-specific fields
	Go     *GoConfig
	Python *PythonConfig
}

// GoConfig holds Go-specific configuration from go_gapic_library.
type GoConfig struct {
	ImportPath    string
	Metadata      bool
	Diregapic     bool
	HasGoGRPC     bool
	HasLegacyGRPC bool
}

// PythonConfig holds Python-specific configuration from py_gapic_library.
type PythonConfig struct {
	// OptArgs contains additional options passed to the generator
	// E.g., ["warehouse-package-name=google-cloud-secret-manager"]
	OptArgs []string
}

// ParseAPI reads a BUILD.bazel file and extracts API configuration for the specified language.
// The apiPath should be relative to the googleapis root (e.g., "google/cloud/secretmanager/v1").
// The language should be one of: "go", "python", "java", etc.
func ParseAPI(googleapisRoot, apiPath, language string) (*APIConfig, error) {
	buildPath := filepath.Join(googleapisRoot, apiPath, "BUILD.bazel")
	data, err := os.ReadFile(buildPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read BUILD.bazel file %s: %w", buildPath, err)
	}
	content := string(data)

	cfg := &APIConfig{Language: language}

	switch language {
	case "go":
		if err := parseGoGapic(content, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse Go config from %s: %w", buildPath, err)
		}
	case "python":
		if err := parsePythonGapic(content, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse Python config from %s: %w", buildPath, err)
		}
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	return cfg, nil
}

// parseGoGapic extracts configuration from go_gapic_library rule.
func parseGoGapic(content string, cfg *APIConfig) error {
	// Find the go_gapic_library block
	re := regexp.MustCompile(`go_gapic_library\((?s:.)*?\)`)
	gapicBlock := re.FindString(content)
	if gapicBlock == "" {
		// No GAPIC library found - this might be a proto-only library
		cfg.HasGAPIC = false
		return nil
	}

	cfg.HasGAPIC = true
	cfg.Go = &GoConfig{}

	// Extract common fields
	cfg.GRPCServiceConfig = findString(gapicBlock, "grpc_service_config")
	cfg.ServiceYAML = strings.TrimPrefix(findString(gapicBlock, "service_yaml"), ":")
	cfg.Transport = findString(gapicBlock, "transport")
	cfg.ReleaseLevel = findString(gapicBlock, "release_level")

	var err error
	if cfg.RestNumericEnums, err = findBool(gapicBlock, "rest_numeric_enums"); err != nil {
		return err
	}

	// Extract Go-specific fields
	cfg.Go.ImportPath = findString(gapicBlock, "importpath")
	if cfg.Go.Metadata, err = findBool(gapicBlock, "metadata"); err != nil {
		return err
	}
	if cfg.Go.Diregapic, err = findBool(gapicBlock, "diregapic"); err != nil {
		return err
	}

	// Check for go_grpc_library vs go_proto_library
	if strings.Contains(content, "go_grpc_library") {
		cfg.Go.HasGoGRPC = true
	}

	goProtoPattern := regexp.MustCompile(`go_proto_library\((?s:.)*?\)`)
	goProtoBlock := goProtoPattern.FindString(content)
	if goProtoBlock != "" {
		if cfg.Go.HasGoGRPC {
			return fmt.Errorf("misconfiguration: both go_grpc_library and go_proto_library present")
		}
		cfg.Go.HasLegacyGRPC = strings.Contains(goProtoBlock, "@io_bazel_rules_go//proto:go_grpc")
	}

	return nil
}

// parsePythonGapic extracts configuration from py_gapic_library rule.
func parsePythonGapic(content string, cfg *APIConfig) error {
	// Find the py_gapic_library block
	re := regexp.MustCompile(`py_gapic_library\((?s:.)*?\)`)
	gapicBlock := re.FindString(content)
	if gapicBlock == "" {
		// No GAPIC library found - this might be a proto-only library
		cfg.HasGAPIC = false
		return nil
	}

	cfg.HasGAPIC = true
	cfg.Python = &PythonConfig{}

	// Extract common fields
	cfg.GRPCServiceConfig = findString(gapicBlock, "grpc_service_config")
	cfg.ServiceYAML = strings.TrimPrefix(findString(gapicBlock, "service_yaml"), ":")
	cfg.Transport = findString(gapicBlock, "transport")
	cfg.ReleaseLevel = findString(gapicBlock, "release_level")

	var err error
	if cfg.RestNumericEnums, err = findBool(gapicBlock, "rest_numeric_enums"); err != nil {
		return err
	}

	// Extract Python-specific fields: opt_args
	cfg.Python.OptArgs = findStringList(gapicBlock, "opt_args")

	return nil
}

<<<<<<< HEAD
// findStringList finds a list of strings in a Bazel rule block.
// E.g., opt_args = ["foo", "bar"].
=======
// findStringList finds a list of strings in a Bazel rule block
// E.g., opt_args = ["foo", "bar"]
>>>>>>> ef6ef5a (feat: generate python successfully)
func findStringList(content, name string) []string {
	// Match: name = [ "item1", "item2", ... ]
	re := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*\[((?:[^]]*?))\]`, name))
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil
	}

	// Extract individual quoted strings
	itemRe := regexp.MustCompile(`"([^"]+)"`)
	items := itemRe.FindAllStringSubmatch(match[1], -1)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if len(item) > 1 {
			result = append(result, item[1])
		}
	}
	return result
}
