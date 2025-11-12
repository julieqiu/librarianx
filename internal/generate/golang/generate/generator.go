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

// Package generate provides the core generation logic for creating Go client libraries from API definitions.
package generate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/generate/golang/bazel"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/generate/golang/execv"
	"github.com/googleapis/librarian/internal/generate/golang/postprocessor"
	"github.com/googleapis/librarian/internal/generate/golang/protoc"
)

// Test substitution vars.
var (
	postProcess = postprocessor.PostProcess
	execvRun    = execv.Run
)

// Generate is the main entrypoint for the `generate` command. It orchestrates
// the entire generation process. The high-level steps are:
//
//  1. Validate the configuration.
//  2. Invoke `protoc` for each API specified in the request, generating Go
//     files into a nested directory structure (e.g.,
//     `/output/cloud.google.com/go/...`).
//  3. Fix the permissions of all generated `.go` files to `0644`.
//  4. Flatten the output directory, moving the generated module(s) to the top
//     level of the output directory (e.g., `/output/chronicle`).
//  5. If the `DisablePostProcessor` flag is false, run the post-processor on the
//     generated module(s), updating versions for snippet metadata,
//     running go mod tidy etc.
//
// The `DisablePostProcessor` flag should always be false in production. It can be
// true during development to inspect the "raw" protoc output before any
// post-processing is applied.
func Generate(ctx context.Context, library *config.Library, sourceDir, outputDir string) error {
	slog.Debug("librariangen: generate command started")

	if err := invokeProtoc(ctx, sourceDir, outputDir, library); err != nil {
		return fmt.Errorf("librariangen: gapic generation failed: %w", err)
	}
	if err := fixPermissions(outputDir); err != nil {
		return fmt.Errorf("librariangen: failed to fix permissions: %w", err)
	}
	if err := flattenOutput(outputDir); err != nil {
		return fmt.Errorf("librariangen: failed to flatten output: %w", err)
	}

	if err := applyModuleVersion(outputDir, library.Name, library.GetModulePath()); err != nil {
		return fmt.Errorf("librariangen: failed to apply module version to output directories: %w", err)
	}

	slog.Debug("librariangen: post-processor enabled")
	if len(library.APIs) == 0 {
		return errors.New("librariangen: no APIs in request")
	}
	moduleDir := filepath.Join(outputDir, library.Name)
	if err := postProcess(ctx, library, outputDir, moduleDir); err != nil {
		return fmt.Errorf("librariangen: post-processing failed: %w", err)
	}
	if err := deleteOutputPaths(outputDir, library.DeleteGenerationOutputPaths); err != nil {
		return fmt.Errorf("librariangen: failed to delete paths specified in delete_generation_output_paths: %w", err)
	}

	slog.Debug("librariangen: generate command finished")
	return nil
}

// invokeProtoc handles the protoc GAPIC generation logic for the 'generate' CLI command.
// It reads a request file, and for each API specified, it invokes protoc
// to generate the client library and its corresponding .repo-metadata.json file.
func invokeProtoc(ctx context.Context, sourceDir, outputDir string, library *config.Library) error {
	for i, api := range library.APIs {
		apiServiceDir := filepath.Join(sourceDir, api.Path)
		slog.Info("processing api", "service_dir", apiServiceDir)
		bazelConfig, err := bazel.Parse(apiServiceDir)
		if api.HasDisableGAPIC() {
			bazelConfig.DisableGAPIC()
		}
		if err != nil {
			return fmt.Errorf("librariangen: failed to parse BUILD.bazel for %s: %w", apiServiceDir, err)
		}
		args, err := protoc.Build(api.Path, bazelConfig, sourceDir, outputDir, api.NestedProtos)
		if err != nil {
			return fmt.Errorf("librariangen: failed to build protoc command for api %q in library %q: %w", api.Path, library.Name, err)
		}
		if err := execvRun(ctx, args, outputDir); err != nil {
			return fmt.Errorf("librariangen: protoc failed for api %q in library %q: %w", api.Path, library.Name, err)
		}
		// Generate the .repo-metadata.json file for this API.
		if err := generateRepoMetadata(sourceDir, outputDir, &library.Apis[i], library, bazelConfig); err != nil {
			return fmt.Errorf("librariangen: failed to generate .repo-metadata.json for api %q in library %q: %w", api.Path, library.Name, err)
		}
	}
	return nil
}

// fixPermissions recursively finds all .go files in the given directory and sets
// their permissions to 0644.
func fixPermissions(dir string) error {
	slog.Debug("librariangen: changing file permissions to 644", "dir", dir)
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			if err := os.Chmod(path, 0644); err != nil {
				return fmt.Errorf("librariangen: failed to chmod %s: %w", path, err)
			}
		}
		return nil
	})
}

// flattenOutput moves the contents of /output/cloud.google.com/go/ to the top
// level of /output.
func flattenOutput(outputDir string) error {
	slog.Debug("librariangen: flattening output directory", "dir", outputDir)
	goDir := filepath.Join(outputDir, "cloud.google.com", "go")
	if _, err := os.Stat(goDir); os.IsNotExist(err) {
		return fmt.Errorf("librariangen: go directory does not exist in path: %s", goDir)
	}
	if err := moveFiles(goDir, outputDir); err != nil {
		return err
	}
	// Remove the now-empty cloud.google.com directory.
	if err := os.RemoveAll(filepath.Join(outputDir, "cloud.google.com")); err != nil {
		return fmt.Errorf("librariangen: failed to remove cloud.google.com: %w", err)
	}
	return nil
}

// applyModuleVersion reorganizes the (already flattened) output directory
// appropriately for versioned modules. For a module path of the form
// cloud.google.com/go/{module-id}/{version}, we expect to find
// /output/{id}/{version} and /output/internal/generated/snippets/{module-id}/{version}.
// In most cases, we only support a single major version of the module, rooted at
// /{module-id} in the repository, so the content of these directories are moved into
// /output/{module-id} and /output/internal/generated/snippets/{id}.
//
// However, when we need to support multiple major versions, we use {module-id}/{version}
// as the *library* ID (in the state file etc). That indicates that the module is rooted
// in that versioned directory (e.g. "pubsub/v2"). In that case, the flattened code is
// already in the right place, so this function doesn't need to do anything.
func applyModuleVersion(outputDir, libraryID, modulePath string) error {
	parts := strings.Split(modulePath, "/")
	// Just cloud.google.com/go/xyz
	if len(parts) == 3 {
		return nil
	}
	if len(parts) != 4 {
		return fmt.Errorf("librariangen: unexpected module path format: %s", modulePath)
	}
	// e.g. dataproc
	id := parts[2]
	// e.g. v2
	version := parts[3]

	if libraryID == id+"/"+version {
		return nil
	}

	srcDir := filepath.Join(outputDir, id)
	srcVersionDir := filepath.Join(srcDir, version)
	snippetsDir := filepath.Join(outputDir, "internal", "generated", "snippets", id)
	snippetsVersionDir := filepath.Join(snippetsDir, version)

	if err := moveFiles(srcVersionDir, srcDir); err != nil {
		return err
	}
	if err := os.RemoveAll(srcVersionDir); err != nil {
		return fmt.Errorf("librariangen: failed to remove %s: %w", srcVersionDir, err)
	}

	if err := moveFiles(snippetsVersionDir, snippetsDir); err != nil {
		return err
	}
	if err := os.RemoveAll(snippetsVersionDir); err != nil {
		return fmt.Errorf("librariangen: failed to remove %s: %w", snippetsVersionDir, err)
	}
	return nil
}

// moveFiles moves all files (and directories) from sourceDir to targetDir.
func moveFiles(sourceDir, targetDir string) error {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("librariangen: failed to read dir %s: %w", sourceDir, err)
	}
	for _, f := range files {
		oldPath := filepath.Join(sourceDir, f.Name())
		newPath := filepath.Join(targetDir, f.Name())
		slog.Debug("librariangen: moving file", "from", oldPath, "to", newPath)
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("librariangen: failed to move %s to %s: %w", oldPath, newPath, err)
		}
	}
	return nil
}

// deleteOutputPaths deletes the specified paths, which may be files
// or directories, relative to the output directory. This is an emergency
// escape hatch for situations where files are generated that we don't want
// to include, such as the internal/generated/snippets/storage/internal directory.
// This is configured in repo-config.yaml at the library level, with the key
// delete_generation_output_paths.
func deleteOutputPaths(outputDir string, pathsToDelete []string) error {
	for _, path := range pathsToDelete {
		// This is so rare that it's useful to be able to validate it easily.
		slog.Info("deleting output path", "path", path)
		if err := os.RemoveAll(filepath.Join(outputDir, path)); err != nil {
			return err
		}
	}
	return nil
}
