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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		repoPath       string
		outputPath     string
		googleapisPath string
	)

	flag.StringVar(&repoPath, "repo", "", "Path to the repository (required)")
	flag.StringVar(&outputPath, "output", "", "Output file path (default: stdout)")
	flag.StringVar(&googleapisPath, "googleapis", "", "Path to googleapis repository for BUILD.bazel files")
	flag.Parse()

	if repoPath == "" {
		return fmt.Errorf("-repo flag is required")
	}

	// Detect language from repository
	language, err := detectLanguage(repoPath)
	if err != nil {
		return fmt.Errorf("failed to detect language: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Detected language: %s\n", language)

	// Read all legacy configuration sources
	reader := &Reader{
		RepoPath:       repoPath,
		GoogleapisPath: googleapisPath,
	}

	fmt.Fprintf(os.Stderr, "Reading legacy configuration from %s...\n", repoPath)
	state, config, buildData, generatorInput, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read legacy configuration: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d libraries in state.yaml\n", len(state.Libraries))

	// Merge all sources into config.Config
	fmt.Fprintf(os.Stderr, "Merging configuration sources...\n")
	cfg, err := merge(state, config, buildData, generatorInput, language)
	if err != nil {
		return fmt.Errorf("failed to merge configuration: %w", err)
	}

	// Deduplicate fields that match defaults
	fmt.Fprintf(os.Stderr, "Deduplicating library-specific fields...\n")
	deduplicate(cfg)

	// Write output
	if outputPath == "" {
		// Write to stdout
		enc := yaml.NewEncoder(os.Stdout)
		enc.SetIndent(2)
		defer enc.Close()

		if err := enc.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode config: %w", err)
		}
	} else {
		// Write to file
		fmt.Fprintf(os.Stderr, "Writing output to %s...\n", outputPath)
		if err := cfg.Write(outputPath); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		// Run yamlfmt if available
		if err := runYamlfmt(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: yamlfmt failed: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "Migration complete!\n")
	}

	return nil
}

// deduplicate removes library-specific fields that match the defaults.
func deduplicate(cfg *config.Config) {
	defaultTransport := ""
	if cfg.Default != nil && cfg.Default.Generate != nil {
		defaultTransport = cfg.Default.Generate.Transport
	}

	for _, lib := range cfg.Libraries {
		// Remove transport if it matches the default
		if defaultTransport != "" && lib.Transport == defaultTransport {
			lib.Transport = ""
		}

		// Simplify API/APIs field
		if len(lib.APIs) == 1 {
			lib.API = lib.APIs[0]
			lib.APIs = nil
		}

		// Remove empty Python section
		if lib.Python != nil && len(lib.Python.OptArgs) == 0 {
			lib.Python = nil
		}
	}
}

// runYamlfmt runs yamlfmt on the output file if the command is available.
func runYamlfmt(path string) error {
	_, err := exec.LookPath("yamlfmt")
	if err != nil {
		// yamlfmt not available, skip
		return nil
	}

	cmd := exec.Command("yamlfmt", path)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// detectLanguage detects the programming language from the repository path.
func detectLanguage(repoPath string) (string, error) {
	// Extract language from repository name
	// Check longer names first to avoid false matches (e.g., "go" in "googleapis")
	languages := []string{"python", "rust", "dart", "java", "node", "ruby", "php", "go"}

	lowerPath := strings.ToLower(repoPath)
	for _, lang := range languages {
		// Look for language in the final path component (repo name)
		if strings.Contains(lowerPath, "cloud-"+lang) || strings.HasSuffix(lowerPath, "-"+lang) {
			return lang, nil
		}
	}

	return "", fmt.Errorf("could not detect language from repository path: %s", repoPath)
}
