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
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/julieqiu/librarianx/internal/fetch"
	cp "github.com/otiai10/copy"
)

//go:embed templates/*
var templates embed.FS

// SetupWorkspace fetches the latest google-cloud-rust commit, downloads the
// tarball, and prepares a new workspace with the necessary files.
func SetupWorkspace(destDir string) error {
	// 1. Get the latest commit of google-cloud-rust.
	commit, err := fetch.Latest("googleapis/google-cloud-rust")
	if err != nil {
		return fmt.Errorf("failed to get latest commit: %w", err)
	}

	// 2. Download the tarball.
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configCache := filepath.Join(home, ".librarian", "cache")
	if err := os.MkdirAll(configCache, 0o755); err != nil {
		return err
	}
	srcDir, err := fetch.DownloadAndExtractTarball("github.com/googleapis/google-cloud-rust", commit, configCache)
	if err != nil {
		return fmt.Errorf("failed to download and extract tarball: %w", err)
	}

	// 3. Copy over the member directories.
	members := []string{
		"src/auth",
		"src/gax",
		"src/gax-internal",
		"src/wkt",
		"src/generated/rpc/types",
	}
	for _, member := range members {
		src := filepath.Join(srcDir, member)
		dest := filepath.Join(destDir, member)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
		if err := cp.Copy(src, dest); err != nil {
			return fmt.Errorf("failed to copy member directory %q: %w", member, err)
		}
	}

	// 4. Create Cargo.toml and .typos.toml from templates.
	return writeTemplates(destDir, nil)
}

func writeTemplates(destDir string, data any) error {
	files, err := templates.ReadDir("templates")
	if err != nil {
		return err
	}
	for _, f := range files {
		name := f.Name()
		path := filepath.Join("templates", name)
		// remove .tmpl extension
		destPath := filepath.Join(destDir, name[:len(name)-5])

		tmpl, err := template.ParseFS(templates, path)
		if err != nil {
			return err
		}
		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()
		if err := tmpl.Execute(out, data); err != nil {
			return err
		}
	}
	return nil
}
