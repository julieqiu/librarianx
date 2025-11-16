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

package python

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Generate generates a Python client library.
// Files and directories specified in library.Keep will be preserved during regeneration.
// If library.Keep is not specified, a default list of paths is used.
func Generate(ctx context.Context, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
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

	// Get keep paths - use library.Keep if specified, otherwise use defaults
	keepPaths := library.Keep
	if len(keepPaths) == 0 {
		keepPaths = defaultKeepPaths(library.Name)
	}

	// Backup files in keep list before generation
	backupDir, err := backupKeepFiles(outdir, keepPaths)
	if err != nil {
		return fmt.Errorf("failed to backup keep files: %w", err)
	}
	defer os.RemoveAll(backupDir) // Clean up backup after we're done

	// Create output directory
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get API paths to generate
	apiPaths := library.APIs
	if len(apiPaths) == 0 && library.API != "" {
		apiPaths = []string{library.API}
	}

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

	// Generate each API
	for _, apiPath := range apiPaths {
		if err := generateAPI(ctx, apiPath, library, googleapisDir, serviceConfigPath, outdir, transport, restNumericEnums); err != nil {
			return fmt.Errorf("failed to generate API %s: %w", apiPath, err)
		}
	}

	// Restore backed up files
	if err := restoreKeepFiles(backupDir, outdir, keepPaths); err != nil {
		return fmt.Errorf("failed to restore keep files: %w", err)
	}

	return nil
}

// backupKeepFiles backs up files/directories in the keep list to a temporary directory.
// Returns the backup directory path.
func backupKeepFiles(outdir string, keepPaths []string) (string, error) {
	// Create temporary backup directory
	backupDir, err := os.MkdirTemp("", "librarian-keep-*")
	if err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Backup each keep path
	for _, keepPath := range keepPaths {
		srcPath := filepath.Join(outdir, keepPath)

		// Skip if path doesn't exist
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		// Create backup path
		dstPath := filepath.Join(backupDir, keepPath)

		// Create parent directory for backup
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return backupDir, fmt.Errorf("failed to create backup parent directory: %w", err)
		}

		// Copy file or directory
		if err := copyPath(srcPath, dstPath); err != nil {
			return backupDir, fmt.Errorf("failed to backup %s: %w", keepPath, err)
		}
	}

	return backupDir, nil
}

// restoreKeepFiles restores backed up files from the backup directory to the output directory.
func restoreKeepFiles(backupDir, outdir string, keepPaths []string) error {
	for _, keepPath := range keepPaths {
		srcPath := filepath.Join(backupDir, keepPath)

		// Skip if backup doesn't exist
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		dstPath := filepath.Join(outdir, keepPath)

		// Remove generated version if it exists
		if err := os.RemoveAll(dstPath); err != nil {
			return fmt.Errorf("failed to remove generated %s: %w", keepPath, err)
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Restore from backup
		if err := copyPath(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to restore %s: %w", keepPath, err)
		}
	}

	return nil
}

// copyPath copies a file or directory from src to dst.
func copyPath(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Copy file mode
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateAPI generates code for a single API.
func generateAPI(ctx context.Context, apiPath string, library *config.Library, googleapisDir, serviceConfigPath, outdir, transport string, restNumericEnums bool) error {
	// Build generator options
	var opts []string

	// Add transport option
	if transport != "" {
		opts = append(opts, fmt.Sprintf("transport=%s", transport))
	}

	// Add rest_numeric_enums option
	if restNumericEnums {
		opts = append(opts, "rest-numeric-enums")
	}

	// Add Python-specific options
	if library.Python != nil && len(library.Python.OptArgs) > 0 {
		opts = append(opts, library.Python.OptArgs...)
	}

	// Add gapic-version from library version
	if library.Version != "" {
		opts = append(opts, fmt.Sprintf("gapic-version=%s", library.Version))
	}

	// Add gRPC service config (retry/timeout settings) from library config if set
	if library.GRPCServiceConfig != "" {
		// GRPCServiceConfig is relative to the API directory
		grpcConfigPath := filepath.Join(googleapisDir, apiPath, library.GRPCServiceConfig)
		opts = append(opts, fmt.Sprintf("retry-config=%s", grpcConfigPath))
	}

	// Add service YAML (API metadata) if provided
	if serviceConfigPath != "" {
		opts = append(opts, fmt.Sprintf("service-yaml=%s", serviceConfigPath))
	}

	// Build protoc command
	protoPattern := filepath.Join(apiPath, "*.proto")

	args := []string{
		protoPattern,
		fmt.Sprintf("--python_gapic_out=%s", outdir),
	}

	// Add options if any
	if len(opts) > 0 {
		optString := "metadata," + strings.Join(opts, ",")
		args = append(args, fmt.Sprintf("--python_gapic_opt=%s", optString))
	}

	// Construct the full command as a shell command
	// We need shell=true because of the glob pattern *.proto
	cmdStr := "protoc " + strings.Join(args, " ")

	// Debug: print the protoc command
	fmt.Fprintf(os.Stderr, "Running: %s\n", cmdStr)

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = googleapisDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("protoc command failed: %w", err)
	}

	return nil
}

// defaultKeepPaths returns the default list of files/directories to preserve during regeneration.
// The library name will be substituted for {name} in the paths.
func defaultKeepPaths(libraryName string) []string {
	paths := []string{
		"packages/{name}/CHANGELOG.md",
		"docs/CHANGELOG.md",
		"docs/README.rst",
		"samples/README.txt",
		"scripts/client-post-processing/",
		"samples/snippets/README.rst",
		"tests/system/",
		"tests/unit/gapic/type/test_type.py",
	}

	result := make([]string, len(paths))
	for i, p := range paths {
		result[i] = strings.ReplaceAll(p, "{name}", libraryName)
	}
	return result
}
