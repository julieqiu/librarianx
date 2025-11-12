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

// Package configure provides configuration generation for API client libraries.
package configure

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/generate/golang/execv"
	"github.com/googleapis/librarian/internal/generate/golang/module"
	"github.com/googleapis/librarian/internal/generate/golang/request"
	"gopkg.in/yaml.v3"
)

// External string template vars.
var (
	//go:embed _README.md.txt
	readmeTmpl string
	//go:embed _version.go.txt
	versionTmpl string
)

// NewAPIStatus is the API.Status value used to represent "this is a new API being configured".
const NewAPIStatus = "new"

// Test substitution vars.
var (
	execvRun     = execv.Run
	requestParse = Parse
	responseSave = saveResponse
)

// Configure configures a new library, or a new API within an existing library.
// This is effectively the entry point of the "configure" container command.
func Configure(ctx context.Context, librarianDir, outputDir, sourceDir, repoDir string, cfg *config.Config) (*config.Library, error) {
	slog.Debug("librariangen: configure command started")

	library, api, err := findLibraryAndAPIToConfigure(cfg)
	if err != nil {
		return nil, err
	}

	response, err := configureLibrary(ctx, outputDir, sourceDir, repoDir, library, api)
	if err != nil {
		return nil, err
	}
	if err := saveConfigureResp(response, librarianDir); err != nil {
		return nil, fmt.Errorf("librariangen: failed to save response: %w", err)
	}

	return response, nil
}

// readConfigureReq reads generate-request.json from the librarian-tool input directory.
// The request file tells librariangen which library and APIs to generate.
// It is prepared by the Librarian tool and mounted at /librarian.
func readConfigureReq(librarianDir string) (*Request, error) {
	reqPath := filepath.Join(librarianDir, "configure-request.json")
	slog.Debug("librariangen: reading configure request", "path", reqPath)

	configureReq, err := requestParse(reqPath)
	if err != nil {
		return nil, err
	}
	slog.Debug("librariangen: successfully unmarshalled request")
	return configureReq, nil
}

// saveConfigureResp saves the response in configure-response.json in the librarian-tool input directory.
// The response file tells Librarian how to reconfigure the library in its state file.
func saveConfigureResp(resp *request.Library, librarianDir string) error {
	respPath := filepath.Join(librarianDir, "configure-response.json")
	slog.Debug("librariangen: saving configure response", "path", respPath)

	if err := responseSave(resp, respPath); err != nil {
		return err
	}
	slog.Debug("librariangen: successfully marshalled response")
	return nil
}

// findLibraryAndAPIToConfigure examines a config, and finds a single library
// containing a single new API, returning both of them. An error is returned
// if there is not exactly one library containing exactly one new API.
func findLibraryAndAPIToConfigure(cfg *config.Config) (*config.Library, *config.API, error) {
	var (
		library *config.Library
		api     *config.API
	)

	for i := range cfg.Libraries {
		candidate := &cfg.Libraries[i]
		var newAPI *config.API
		for j := range candidate.APIs {
			if candidate.APIs[j].Status == NewAPIStatus {
				if newAPI != nil {
					return nil, nil, fmt.Errorf("librariangen: library %s has multiple new APIs", candidate.Name)
				}
				newAPI = &candidate.APIs[j]
			}
		}

		if newAPI != nil {
			if library != nil {
				return nil, nil, fmt.Errorf("librariangen: multiple libraries have new APIs (at least %s and %s)", library.Name, candidate.Name)
			}
			library = candidate
			api = newAPI
		}
	}

	if library == nil {
		return nil, nil, fmt.Errorf("librariangen: no libraries have new APIs")
	}

	return library, api, nil
}

// configureLibrary performs the real work of configuring a new or updated module,
// creating files and populating the state file entry.
// In theory we could just have a return type of "error", but logically this is
// returning the configure-response... it just happens to be "the library being configured"
// at the moment. If the format of configure-response ever changes, we'll need fewer
// changes if we don't make too many assumptions now.
func configureLibrary(ctx context.Context, outputDir, sourceDir, repoDir string, library *config.Library, api *config.API) (*config.Library, error) {
	moduleRoot := filepath.Join(outputDir, library.Name)
	if err := os.Mkdir(moduleRoot, 0755); err != nil {
		return nil, err
	}
	// Only a single API path can be added on each configure call, so we can tell
	// if this is a new library if it's got exactly one API path.
	// In that case, we need to add:
	// - CHANGES.md (static text: "# Changes")
	// - README.md
	// - internal/version.go
	// - go.mod
	if len(library.APIs) == 1 {
		if err := generateReadme(outputDir, sourceDir, library); err != nil {
			return nil, err
		}
		if err := generateChanges(outputDir, library); err != nil {
			return nil, err
		}
		if err := module.GenerateInternalVersionFile(moduleRoot, library.Version); err != nil {
			return nil, err
		}
		if err := goModEditReplaceInSnippets(ctx, outputDir, repoDir, library.GetModulePath(), "../../../"+library.Name); err != nil {
			return nil, err
		}
		// The postprocessor for the generate command will run "go mod init" and "go mod tidy"
		// - because it has the source code at that point. It *won't* have the version files we've
		// created here though. That's okay so long as our version.go files don't have any dependencies.
	}

	// Whether it's a new library or not, generate a version file for the new client directory.
	if err := generateClientVersionFile(outputDir, sourceDir, library, api.Path); err != nil {
		return nil, err
	}

	return library, nil
}

// generateReadme generates a README.md file in the module's root directory,
// using the service config for the first API in the library to obtain the
// service's title.
func generateReadme(outputDir, sourceDir string, library *config.Library) error {
	readmePath := filepath.Join(outputDir, library.Name, "README.md")
	serviceYAMLPath := filepath.Join(sourceDir, library.APIs[0].Path, library.APIs[0].ServiceConfig)
	title, err := readTitleFromServiceYAML(serviceYAMLPath)
	if err != nil {
		return fmt.Errorf("librariangen: failed to read title from service yaml: %w", err)
	}

	slog.Info("librariangen: creating file", "path", readmePath)
	readmeFile, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	defer readmeFile.Close()
	t := template.Must(template.New("readme").Parse(readmeTmpl))
	readmeData := struct {
		Name       string
		ModulePath string
	}{
		Name:       title,
		ModulePath: "cloud.google.com/go/" + library.Name,
	}
	return t.Execute(readmeFile, readmeData)
}

// generateChanges generates a CHANGES.md file at the root of the module.
func generateChanges(outputDir string, library *config.Library) error {
	changesPath := filepath.Join(outputDir, library.Name, "CHANGES.md")
	slog.Info("librariangen: creating file", "path", changesPath)
	content := "# Changes\n"
	return os.WriteFile(changesPath, []byte(content), 0644)
}

// generateClientVersionFile creates a version.go file for a client.
func generateClientVersionFile(outputDir, sourceDir string, moduleConfig *config.Library, apiPath string) error {
	var apiConfig *config.API
	for _, a := range moduleConfig.APIs {
		if a.Path == apiPath {
			apiConfig = &config.API{Path: a.Path}
			break
		}
	}
	if apiConfig == nil {
		apiConfig = &config.API{Path: apiPath}
	}
	clientDir, err := apiConfig.GetClientDirectory(moduleConfig.Name)
	if err != nil {
		return err
	}

	fullClientDir := filepath.Join(outputDir, moduleConfig.Name, clientDir)
	if err := os.MkdirAll(fullClientDir, 0755); err != nil {
		return err
	}
	versionPath := filepath.Join(fullClientDir, "version.go")
	slog.Info("librariangen: creating file", "path", versionPath)
	t := template.Must(template.New("version").Parse(versionTmpl))
	versionData := struct {
		Year               int
		Package            string
		ModuleRootInternal string
	}{
		Year:    time.Now().Year(),
		Package: filepath.Base(filepath.Dir(fullClientDir)), // The package name is the name of the directory containing the client directory (e.g. `apiv1beta1`).

		ModuleRootInternal: moduleConfig.GetModulePath() + "/internal",
	}
	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, versionData)
}

// goModEditReplaceInSnippets copies internal/generated/snippets/go.mod from
// repoDir to outputDir, then runs go mod edit to replace the specified
// modulePath with relativeDir which is expected to the location of the module
// relative to internal/generated/snippets.
func goModEditReplaceInSnippets(ctx context.Context, outputDir, repoDir, modulePath, relativeDir string) error {
	outputSnippetsDir := filepath.Join(outputDir, "internal", "generated", "snippets")
	if err := os.MkdirAll(outputSnippetsDir, 0755); err != nil {
		return err
	}
	copyRepoFileToOutput(outputDir, repoDir, "internal/generated/snippets/go.mod")
	replaceStr := fmt.Sprintf("%s=%s", modulePath, relativeDir)
	args := []string{"go", "mod", "edit", "-replace", replaceStr}
	slog.Info("librariangen: running go mod edit -replace", "replace", replaceStr, "directory", outputSnippetsDir)
	return execvRun(ctx, args, outputSnippetsDir)
}

// copyRepoFileToOutput copies a single file (identified via path)
// from repoDir to outputDir.
func copyRepoFileToOutput(outputDir, repoDir, path string) error {
	src := filepath.Join(repoDir, path)
	dst := filepath.Join(outputDir, path)
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// updateLibraryState updates the library to add any required removal/preservation
// regexes for the specified API.
func updateLibraryState(moduleConfig *config.Library, library *configureLibrary, api *configureAPI) error {
	var apiConfig *config.API
	for _, a := range moduleConfig.APIs {
		if a.Path == api.Path {
			apiConfig = &config.API{Path: a.Path}
			break
		}
	}
	if apiConfig == nil {
		apiConfig = &config.API{Path: api.Path}
	}
	clientDirectory, err := apiConfig.GetClientDirectory(moduleConfig.Name)
	if err != nil {
		return err
	}
	apiParts := strings.Split(api.Path, "/")
	protobufDir := apiParts[len(apiParts)-2] + "pb/.*"
	generatedPaths := []string{
		"[^/]*_client\\.go",
		"[^/]*_client_example_go123_test\\.go",
		"[^/]*_client_example_test\\.go",
		"auxiliary\\.go",
		"auxiliary_go123\\.go",
		"doc\\.go",
		"gapic_metadata\\.json",
		"helpers\\.go",
		"\\.repo-metadata\\.json",
		protobufDir,
	}
	for _, generatedPath := range generatedPaths {
		library.RemoveRegex = append(library.RemoveRegex, "^"+path.Join(library.ID, clientDirectory, generatedPath)+"$")
	}
	return nil
}

// readTitleFromServiceYAML reads the service YAML file and returns the title.
func readTitleFromServiceYAML(path string) (string, error) {
	slog.Info("librariangen: reading service yaml", "path", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("librariangen: failed to read service yaml file: %w", err)
	}
	var serviceConfig struct {
		Title string `yaml:"title"`
	}
	if err := yaml.Unmarshal(data, &serviceConfig); err != nil {
		return "", fmt.Errorf("librariangen: failed to unmarshal service yaml: %w", err)
	}
	if serviceConfig.Title == "" {
		return "", errors.New("librariangen: title not found in service yaml")
	}
	return serviceConfig.Title, nil
}

// Request corresponds to a librarian configure request.
// It is unmarshalled from the configure-request.json file. Note that
// this request is in a different form from most other requests, as it
// contains all libraries.
type Request struct {
	// All libraries configured within the repository.
	Libraries []*configureLibrary `json:"libraries"`
}

// Parse reads a configure-request.json file from the given path and unmarshals
// it into a ConfigureRequest struct.
func Parse(path string) (*Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("librariangen: failed to read request file from %s: %w", path, err)
	}

	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("librariangen: failed to unmarshal request file %s: %w", path, err)
	}

	return &req, nil
}
