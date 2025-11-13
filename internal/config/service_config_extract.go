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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

// ExtractServiceConfigSettings reads a service config YAML file and extracts
// language-specific settings to populate LibraryConfig fields.
//
// This function:
// 1. Finds the service config YAML file for the given API path
// 2. Parses publishing.library_settings for language-specific configuration
// 3. Populates Java, Python, Go, Node, Dotnet fields in LibraryConfig
// 4. Ignores version, launch_stage, and destinations (these are derived)
//
// Returns nil if no service config is found (not an error - API may not have one).
func ExtractServiceConfigSettings(googleapisRoot string, apiPath string, language string) (*LibraryConfig, error) {
	if googleapisRoot == "" {
		return nil, nil
	}

	// Find service config YAML file
	serviceConfigPath := findServiceConfigForAPI(googleapisRoot, apiPath)
	if serviceConfigPath == "" {
		return nil, nil // No service config found (not an error)
	}

	// Parse service config
	svcConfig, err := readServiceConfigYAML(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service config %s: %w", serviceConfigPath, err)
	}

	// Extract language-specific settings from library_settings
	config := &LibraryConfig{}
	extractLanguageSettings(svcConfig, language, config)

	// Return nil if no settings were extracted
	if isEmpty(config) {
		return nil, nil
	}

	return config, nil
}

// findServiceConfigForAPI finds the service config YAML file for an API path.
// Returns empty string if not found.
func findServiceConfigForAPI(googleapisRoot string, apiPath string) string {
	// Service config files are typically named <service>_<version>.yaml
	// Example: google/cloud/secretmanager/v1 -> secretmanager_v1.yaml

	dir := filepath.Join(googleapisRoot, apiPath)

	// Try common naming patterns
	parts := strings.Split(apiPath, "/")
	if len(parts) < 2 {
		return ""
	}

	serviceName := parts[len(parts)-2] // e.g., "secretmanager"
	version := parts[len(parts)-1]     // e.g., "v1"

	// Try: <service>_<version>.yaml
	pattern1 := filepath.Join(dir, serviceName+"_"+version+".yaml")
	if _, err := os.Stat(pattern1); err == nil {
		return pattern1
	}

	// Try: <service>.yaml
	pattern2 := filepath.Join(dir, serviceName+".yaml")
	if _, err := os.Stat(pattern2); err == nil {
		return pattern2
	}

	return ""
}

// readServiceConfigYAML reads and parses a service config YAML file.
func readServiceConfigYAML(path string) (*serviceconfig.Service, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Convert YAML to JSON (protobuf unmarshaler expects JSON)
	var yamlData interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JSON: %w", err)
	}

	// Parse into protobuf message
	svc := &serviceconfig.Service{}
	unmarshalOpts := protojson.UnmarshalOptions{
		DiscardUnknown: true, // Ignore fields not in the protobuf definition
	}
	if err := unmarshalOpts.Unmarshal(jsonData, svc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service config: %w", err)
	}

	return svc, nil
}

// extractLanguageSettings extracts language-specific settings from service config
// and populates the appropriate fields in LibraryConfig.
func extractLanguageSettings(svc *serviceconfig.Service, language string, config *LibraryConfig) {
	if svc.Publishing == nil || len(svc.Publishing.LibrarySettings) == 0 {
		return
	}

	// Use the first library_settings entry (most services only have one)
	settings := svc.Publishing.LibrarySettings[0]

	switch language {
	case "java":
		if javaSettings := settings.GetJavaSettings(); javaSettings != nil {
			config.Java = &JavaLibrary{}
			if pkg := javaSettings.GetLibraryPackage(); pkg != "" {
				config.Java.Package = pkg
			}
			if classNames := javaSettings.GetServiceClassNames(); len(classNames) > 0 {
				config.Java.ServiceClassNames = classNames
			}
		}

	case "python":
		if pythonSettings := settings.GetPythonSettings(); pythonSettings != nil {
			config.Python = &PythonLibrary{}
			if expFeatures := pythonSettings.GetExperimentalFeatures(); expFeatures != nil {
				config.Python.RestAsyncIOEnabled = expFeatures.GetRestAsyncIoEnabled()
				// Note: unversioned_package_disabled doesn't exist in the protobuf
				// It may need to be extracted from opt_args or another field
			}
		}

	case "go":
		// Go settings extraction not yet implemented
		// The protobuf definition may not have renamed_services directly
		// and may need custom parsing from the YAML
		_ = settings.GetGoSettings()

	case "node":
		if nodeSettings := settings.GetNodeSettings(); nodeSettings != nil {
			if common := nodeSettings.GetCommon(); common != nil {
				if selective := common.GetSelectiveGapicGeneration(); selective != nil {
					config.Node = &NodeLibrary{
						SelectiveMethods: selective.GetMethods(),
					}
				}
			}
		}

	case "dotnet":
		if dotnetSettings := settings.GetDotnetSettings(); dotnetSettings != nil {
			config.Dotnet = &DotnetLibrary{}
			if renamedServices := dotnetSettings.GetRenamedServices(); len(renamedServices) > 0 {
				config.Dotnet.RenamedServices = renamedServices
			}
			if renamedResources := dotnetSettings.GetRenamedResources(); len(renamedResources) > 0 {
				config.Dotnet.RenamedResources = renamedResources
			}
		}
	}
}

// isEmpty checks if a LibraryConfig has no language-specific settings.
func isEmpty(config *LibraryConfig) bool {
	if config == nil {
		return true
	}
	return config.Java == nil &&
		config.Python == nil &&
		config.Go == nil &&
		config.Node == nil &&
		config.Dotnet == nil
}
