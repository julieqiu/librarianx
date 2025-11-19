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
	"os"
	"path/filepath"

	"github.com/julieqiu/librarianx/internal/config"
	"github.com/julieqiu/librarianx/internal/fetch"
)

const (
	// GoogleCloudPythonCommit is the commit SHA for google-cloud-python repository.
	GoogleCloudPythonCommit = "a4229b8183bee296fb8ca1a4698ff258f1791b06"
	// SynthtoolCommit is the commit SHA for synthtool repository.
	SynthtoolCommit = "6702a344265de050bceaff45d62358bb0023ba7d"
)

// Init initializes a default Python config and sets up the Python environment.
// It returns the default config and the Python sources configuration.
func Init(ctx context.Context, cacheDir string) (*config.Default, *config.PythonSources, error) {
	if err := downloadGoogleCloudPython(cacheDir, GoogleCloudPythonCommit); err != nil {
		return nil, nil, err
	}
	if err := copySynthtoolInput(); err != nil {
		return nil, nil, err
	}
	if err := installSynthtool(ctx, cacheDir, SynthtoolCommit); err != nil {
		return nil, nil, err
	}

	sources := &config.PythonSources{
		GoogleCloudPython: &config.Source{
			Commit: GoogleCloudPythonCommit,
		},
		Synthtool: &config.Source{
			Commit: SynthtoolCommit,
		},
	}

	defaults := &config.Default{
		Output: "packages/{name}/",
		Generate: &config.DefaultGenerate{
			Auto:             true,
			OneLibraryPer:    "api",
			Transport:        "grpc+rest",
			RestNumericEnums: true,
			ReleaseLevel:     "stable",
		},

		Release: &config.DefaultRelease{
			TagFormat: "{name}/v{version}",
			Remote:    "origin",
			Branch:    "main",
		},
	}

	return defaults, sources, nil
}

func downloadGoogleCloudPython(cacheDir, commit string) error {
	_, err := fetch.DownloadAndExtractTarball("github.com/googleapis/google-cloud-python", commit, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to download google-cloud-python: %w", err)
	}
	return nil
}

func copySynthtoolInput() error {
	srcDir := ".librarian/generator-input/client-post-processing"
	dstDir := "synthtool-input"

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(dstDir); err != nil {
		return fmt.Errorf("failed to remove existing synthtool-input: %w", err)
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// ConfigDefault initializes a default Python config.
func ConfigDefault() *config.Default {
	return &config.Default{
		Output: "packages/{name}/",
		Generate: &config.DefaultGenerate{
			Auto:             true,
			OneLibraryPer:    "api",
			Transport:        "grpc+rest",
			RestNumericEnums: true,
			ReleaseLevel:     "stable",
		},

		Release: &config.DefaultRelease{
			TagFormat: "{name}/v{version}",
			Remote:    "origin",
			Branch:    "main",
		},
	}
}
