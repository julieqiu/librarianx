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

// Package python provides functionality for generating Python client libraries.
package python

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/julieqiu/librarianx/internal/config"
)

// Generate generates a Python client library.
// Files and directories specified in library.Keep will be preserved during regeneration.
// If library.Keep is not specified, a default list of paths is used.
func Generate(ctx context.Context, language, repo string, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput, defaultAPI string) error {
	// Determine output directory
	outdir := library.Path
	if outdir == "" {
		// Use default output pattern if no explicit path
		if defaults != nil {
			outdir = strings.ReplaceAll(defaults.Output, "{name}", library.Name)
		}
	}

	// Convert to absolute path since protoc runs from a different directory
	var err error
	outdir, err = filepath.Abs(outdir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}
	fmt.Println(outdir)

	// Clean output directory before generation
	if err := cleanOutputDirectory(outdir, library.Keep, library.Name); err != nil {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get API paths to generate
	apiPaths := config.GetLibraryAPIs(library)
	if len(apiPaths) == 0 {
		return fmt.Errorf("no APIs specified for library %s", library.Name)
	}

	// Get transport from library or defaults
	transport := library.Transport
	if transport == "" && defaults != nil && defaults.Generate != nil {
		transport = defaults.Generate.Transport
	}

	// Get rest_numeric_enums from library or defaults
	restNumericEnums := defaults != nil && defaults.Generate != nil && defaults.Generate.RestNumericEnums
	if library.RestNumericEnums != nil {
		restNumericEnums = *library.RestNumericEnums
	}

	// Generate .repo-metadata.json BEFORE running protoc so it can use it for README generation
	// Only generate once (not per API) to avoid the last API overwriting it
	metadataPath := filepath.Join(outdir, ".repo-metadata.json")
	if serviceConfigPath != "" && repo != "" {
		// Only generate if it doesn't exist yet (for multi-API libraries called multiple times)
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			if err := config.GenerateRepoMetadata(library, language, repo, serviceConfigPath, outdir, apiPaths); err != nil {
				return fmt.Errorf("failed to generate .repo-metadata.json: %w", err)
			}
		}
	}

	// Generate each API with its own service config
	for apiPath, apiServiceConfig := range library.APIServiceConfigs {
		// Only generate unversioned package for the default (latest stable) API
		isDefaultAPI := apiPath == defaultAPI

		if err := generateAPI(ctx, apiPath, library, googleapisDir, apiServiceConfig, outdir, transport, restNumericEnums, isDefaultAPI); err != nil {
			return fmt.Errorf("failed to generate API %s: %w", apiPath, err)
		}
	}

	// Fix generated files that have incorrect package names
	// For beta versions (v1beta1, v1beta2), use historical package name without hyphen
	betaPackageName := strings.ReplaceAll(library.Name, "-secret-manager", "-secretmanager")
	if err := fixGeneratedPackageNames(outdir, library.Name, betaPackageName); err != nil {
		return fmt.Errorf("failed to fix package names in generated files: %w", err)
	}

	// Run isort to sort imports, then black to format code
	if err := runIsort(outdir); err != nil {
		return fmt.Errorf("failed to run isort: %w", err)
	}
	if err := runBlackFormatter(outdir); err != nil {
		return fmt.Errorf("failed to run black formatter: %w", err)
	}

	// Copy README.rst to docs/README.rst
	if err := copyReadmeToDocsDir(outdir); err != nil {
		return fmt.Errorf("failed to copy README to docs: %w", err)
	}

	// Clean up files that shouldn't be in the final output
	if err := cleanupPostProcessingFiles(outdir); err != nil {
		return fmt.Errorf("failed to cleanup post-processing files: %w", err)
	}

	return nil
}

// generateAPI generates code for a single API.
func generateAPI(ctx context.Context, apiPath string, library *config.Library, googleapisDir, serviceConfigPath, outdir, transport string, restNumericEnums, isDefaultAPI bool) error {
	// Check if this is a proto-only library
	isProtoOnly := library.Python != nil && library.Python.IsProtoOnly

	protoPattern := filepath.Join(apiPath, "*.proto")
	var args []string
	var cmdStr string

	if isProtoOnly {
		// Proto-only library: generate Python proto files only
		args = []string{
			protoPattern,
			fmt.Sprintf("--python_out=%s", outdir),
			fmt.Sprintf("--pyi_out=%s", outdir),
		}
		cmdStr = "protoc " + strings.Join(args, " ")
	} else {
		// GAPIC library: generate full client library
		var opts []string

		// Add transport option
		if transport != "" {
			opts = append(opts, fmt.Sprintf("transport=%s", transport))
		}

		// Add rest_numeric_enums option
		if restNumericEnums {
			opts = append(opts, "rest-numeric-enums")
		}

		// Disable unversioned package for non-default APIs
		if !isDefaultAPI {
			opts = append(opts, "unversioned-package-disabled")
		}

		// Add Python-specific options
		if library.Python != nil && len(library.Python.OptArgs) > 0 {
			opts = append(opts, library.Python.OptArgs...)
		}

		// Add gapic-version from library version
		if library.Version != "" {
			opts = append(opts, fmt.Sprintf("gapic-version=%s", library.Version))
		}

		// Add gRPC service config (retry/timeout settings)
		// Try library config first, then auto-discover
		grpcConfigPath := ""
		if library.GRPCServiceConfig != "" {
			// GRPCServiceConfig is relative to the API directory
			grpcConfigPath = filepath.Join(googleapisDir, apiPath, library.GRPCServiceConfig)
		} else {
			// Auto-discover: look for *_grpc_service_config.json in the API directory
			apiDir := filepath.Join(googleapisDir, apiPath)
			matches, err := filepath.Glob(filepath.Join(apiDir, "*_grpc_service_config.json"))
			if err == nil && len(matches) > 0 {
				grpcConfigPath = matches[0]
			}
		}
		if grpcConfigPath != "" {
			opts = append(opts, fmt.Sprintf("retry-config=%s", grpcConfigPath))
		}

		// Add service YAML (API metadata) if provided
		if serviceConfigPath != "" {
			opts = append(opts, fmt.Sprintf("service-yaml=%s", serviceConfigPath))
		}

		args = []string{
			protoPattern,
			fmt.Sprintf("--python_gapic_out=%s", outdir),
		}

		// Add options if any
		if len(opts) > 0 {
			optString := "metadata," + strings.Join(opts, ",")
			args = append(args, fmt.Sprintf("--python_gapic_opt=%s", optString))
		}

		cmdStr = "protoc " + strings.Join(args, " ")
	}

	// Debug: print the protoc command
	fmt.Fprintf(os.Stderr, "\nRunning: %s\n", cmdStr)

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = googleapisDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("protoc command failed: %w", err)
	}

	return nil
}

// fixGeneratedPackageNames fixes package names in generated files that protoc generates incorrectly.
// This handles multiple issues:
// 1. py.typed files that reference the wrong package name
// 2. docs/conf.py that has the wrong project name and Python 2 'u' prefix
// 3. noxfile.py that has wrong package name
// 4. Sample Python files with wrong pip install commands (stable versions use stablePackageName, beta versions use betaPackageName)
// 5. JSON snippet metadata files with wrong package names
// For historical compatibility, beta API versions (v1beta1, v1beta2) use a different package name.
func fixGeneratedPackageNames(outdir, stablePackageName, betaPackageName string) error {
	// Walk through the directory to find all files that need fixing
	return filepath.Walk(outdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Determine which package name to use based on API version
		isBetaVersion := strings.Contains(path, "v1beta1") || strings.Contains(path, "v1beta2")
		packageName := stablePackageName
		if isBetaVersion {
			packageName = betaPackageName
		}

		// Fix py.typed files (skip beta versions - they don't need fixing)
		if info.Name() == "py.typed" {
			if !isBetaVersion {
				if err := fixPyTypedFile(path, packageName); err != nil {
					return fmt.Errorf("failed to fix %s: %w", path, err)
				}
			}
		}

		// Fix docs/conf.py files
		if info.Name() == "conf.py" && strings.Contains(path, "/docs/") {
			if err := fixDocsConfPy(path, stablePackageName); err != nil {
				return fmt.Errorf("failed to fix %s: %w", path, err)
			}
		}

		// Fix noxfile.py
		if info.Name() == "noxfile.py" {
			if err := fixPythonFile(path, stablePackageName); err != nil {
				return fmt.Errorf("failed to fix %s: %w", path, err)
			}
		}

		// Fix Python sample files (use beta package name for beta versions)
		if strings.HasSuffix(info.Name(), ".py") && strings.Contains(path, "/samples/") {
			if err := fixPythonFile(path, packageName); err != nil {
				return fmt.Errorf("failed to fix %s: %w", path, err)
			}
		}

		// Fix JSON snippet metadata files (use beta package name for beta versions)
		if strings.HasSuffix(info.Name(), ".json") && strings.Contains(path, "snippet_metadata") {
			if err := fixJSONFile(path, packageName); err != nil {
				return fmt.Errorf("failed to fix %s: %w", path, err)
			}
		}

		return nil
	})
}

// fixPyTypedFile fixes the package name comment in a py.typed file.
func fixPyTypedFile(path, correctPackageName string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Replace any occurrence of "google-cloud-<service>" with the correct package name
	// Pattern: "# The google-cloud-<something> package uses inline types."
	re := regexp.MustCompile(`# The (google-[a-z0-9-]+) package uses inline types\.`)
	newContent := re.ReplaceAllString(string(content), fmt.Sprintf("# The %s package uses inline types.", correctPackageName))

	if newContent != string(content) {
		return os.WriteFile(path, []byte(newContent), 0644)
	}
	return nil
}

// fixDocsConfPy fixes the project name and removes Python 2 'u' prefix in docs/conf.py.
func fixDocsConfPy(path, correctPackageName string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	originalContent := string(content)
	newContent := originalContent

	// Fix project name and remove 'u' prefix
	// Pattern: project = u"google-cloud-<something>"
	re1 := regexp.MustCompile(`project = u"(google-[a-z0-9-]+)"`)
	newContent = re1.ReplaceAllString(newContent, fmt.Sprintf(`project = "%s"`, correctPackageName))

	// Also fix cases without 'u' prefix but wrong name
	re2 := regexp.MustCompile(`project = "(google-[a-z0-9-]+)"`)
	newContent = re2.ReplaceAllString(newContent, fmt.Sprintf(`project = "%s"`, correctPackageName))

	// Fix copyright and author to remove 'u' prefix
	newContent = strings.ReplaceAll(newContent, `copyright = u"`, `copyright = "`)
	newContent = strings.ReplaceAll(newContent, `author = u"`, `author = "`)

	if newContent != originalContent {
		return os.WriteFile(path, []byte(newContent), 0644)
	}
	return nil
}

// fixPythonFile fixes package names in Python sample files.
func fixPythonFile(path, correctPackageName string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Fix pip install commands
	// Pattern: pip install google-cloud-<something> OR python3 -m pip install google-cloud-<something>
	re := regexp.MustCompile(`(python3 -m pip install |pip3 install |pip install )(google-[a-z0-9-]+)`)
	newContent := re.ReplaceAllString(string(content), fmt.Sprintf("${1}%s", correctPackageName))

	if newContent != string(content) {
		return os.WriteFile(path, []byte(newContent), 0644)
	}
	return nil
}

// fixJSONFile fixes package names in JSON snippet metadata files.
func fixJSONFile(path, correctPackageName string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Fix package name in JSON
	// Pattern: "name": "google-cloud-<something>"
	re := regexp.MustCompile(`("name":\s*")(google-[a-z0-9-]+)(")`)
	newContent := re.ReplaceAllString(string(content), fmt.Sprintf(`${1}%s${3}`, correctPackageName))

	if newContent != string(content) {
		return os.WriteFile(path, []byte(newContent), 0644)
	}
	return nil
}

// runIsort runs the isort import sorter on Python files in the output directory.
// The --fss flag forces strict alphabetical sorting within sections.
func runIsort(outdir string) error {
	fmt.Fprintf(os.Stderr, "\nRunning: isort --fss %s\n", outdir)
	cmd := exec.Command("isort", "--fss", outdir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("isort failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// runBlackFormatter runs the black code formatter on Python files in the output directory.
// Black enforces double quotes and consistent Python formatting.
func runBlackFormatter(outdir string) error {
	fmt.Fprintf(os.Stderr, "\nRunning: black %s\n", outdir)
	cmd := exec.Command("black", outdir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("black formatter failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// copyReadmeToDocsDir copies README.rst to docs/README.rst.
// This handles symlinks properly by reading content and writing a real file.
func copyReadmeToDocsDir(outdir string) error {
	sourcePath := filepath.Join(outdir, "README.rst")
	docsPath := filepath.Join(outdir, "docs")
	destPath := filepath.Join(docsPath, "README.rst")

	// If source doesn't exist, nothing to copy
	if _, err := os.Lstat(sourcePath); os.IsNotExist(err) {
		return nil
	}

	// Read content from source (follows symlinks)
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Create docs directory if it doesn't exist
	if err := os.MkdirAll(docsPath, 0755); err != nil {
		return err
	}

	// Remove any existing symlink at destination
	if info, err := os.Lstat(destPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(destPath); err != nil {
				return err
			}
		}
	}

	// Write content to destination as a real file
	return os.WriteFile(destPath, content, 0644)
}

// cleanupPostProcessingFiles removes files that shouldn't be in the final output.
func cleanupPostProcessingFiles(outdir string) error {
	// Remove directories
	if err := os.RemoveAll(filepath.Join(outdir, ".nox")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove .nox: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(outdir, "owl-bot-staging")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove owl-bot-staging: %w", err)
	}

	// Remove CHANGELOG.md files (they should be symlinks, any generated copies are removed)
	os.Remove(filepath.Join(outdir, "CHANGELOG.md"))
	os.Remove(filepath.Join(outdir, "docs", "CHANGELOG.md"))

	// Remove client-post-processing YAML files
	scriptsPath := filepath.Join(outdir, "scripts", "client-post-processing")
	if yamlFiles, err := filepath.Glob(filepath.Join(scriptsPath, "*.yaml")); err == nil {
		for _, f := range yamlFiles {
			os.Remove(f)
		}
	}

	return nil
}
