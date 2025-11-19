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

package language

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/julieqiu/librarianx/internal/config"
	"github.com/julieqiu/librarianx/internal/language/internal/python"
	"github.com/julieqiu/librarianx/internal/language/internal/rust"
)

// Create creates a new client library for the specified language.
func Create(ctx context.Context, language, repo string, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
	switch language {
	case "rust":
		return rust.Create(ctx, library, defaults, googleapisDir, serviceConfigPath, defaultOutput)
	case "python":
		return python.Create(ctx, library, defaults, googleapisDir, serviceConfigPath, defaultOutput)
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

// PostProcess runs only the post-processing step (e.g., synthtool) for the specified language.
func PostProcess(ctx context.Context, language, repo string, library *config.Library, defaults *config.Default) error {
	switch language {
	case "python":
		return python.PostProcess(ctx, repo, library, defaults)
	case "rust":
		return fmt.Errorf("post-processing not supported for rust")
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

// Generate generates a library based on the one_library_per mode.
// For "api" mode: generates once for all APIs in the library.
// For "channel" mode: generates separately for each API version.
func Generate(ctx context.Context, oneLibraryPer, language, repo string, library *config.Library, defaults *config.Default, googleapisDir string) error {
	switch oneLibraryPer {
	case "api":
		return generateForAPI(ctx, language, repo, library, defaults, googleapisDir)
	case "channel":
		return generateForChannel(ctx, language, repo, library, defaults, googleapisDir)
	default:
		return fmt.Errorf("invalid one_library_per value %q: must be \"api\" or \"channel\"", oneLibraryPer)
	}
}

// generateAPI generates code for a single API using language-specific generators.
// This is used internally by generateForAPI and generateForChannel.
func generateAPI(ctx context.Context, language, repo string, library *config.Library, defaults *config.Default, googleapisDir, serviceConfigPath, defaultOutput string) error {
	switch language {
	case "rust":
		return rust.Generate(ctx, library, defaults, googleapisDir, serviceConfigPath, defaultOutput)
	case "python":
		defaultAPI := getDefaultAPI(library)
		return python.Generate(ctx, language, repo, library, defaults, googleapisDir, serviceConfigPath, defaultOutput, defaultAPI)
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

// generateForAPI generates a single library containing all API versions.
// Used for languages with "one_library_per: api" (Python, Go).
func generateForAPI(ctx context.Context, language, repo string, library *config.Library, defaults *config.Default, googleapisDir string) error {
	// Use the first service config path (all point to the same service YAML)
	var primaryServiceConfigPath string
	for _, serviceConfigPath := range library.APIServiceConfigs {
		primaryServiceConfigPath = serviceConfigPath
		break
	}

	return generateAPI(ctx, language, repo, library, defaults, googleapisDir, primaryServiceConfigPath, defaults.Output)
}

// generateForChannel generates a separate library for each API version.
// Used for languages with "one_library_per: channel" (Rust, Dart).
func generateForChannel(ctx context.Context, language, repo string, library *config.Library, defaults *config.Default, googleapisDir string) error {
	for apiPath, serviceConfigPath := range library.APIServiceConfigs {
		// Create a single-API library for this version
		singleAPILibrary := *library
		singleAPILibrary.API = apiPath
		singleAPILibrary.APIServiceConfigs = map[string]string{apiPath: serviceConfigPath}

		if err := generateAPI(ctx, language, repo, &singleAPILibrary, defaults, googleapisDir, serviceConfigPath, defaults.Output); err != nil {
			return err
		}
	}
	return nil
}

// getDefaultAPI returns the default API path for a library.
// The default is the latest stable version, or if no stable versions exist, the latest pre-release.
// Version ordering: v2, v1, v2beta1, v1beta1, v2alpha1, v1alpha1.
func getDefaultAPI(lib *config.Library) string {
	apis := config.GetLibraryAPIs(lib)
	if len(apis) == 0 {
		return ""
	}
	if len(apis) == 1 {
		return apis[0]
	}

	// Sort APIs by version, latest stable first
	sorted := sortAPIsByVersion(apis)
	fmt.Fprintf(os.Stderr, "Selecting default API for %s from %v: %s\n", lib.Name, sorted, sorted[0])
	return sorted[0]
}

// sortAPIsByVersion sorts API paths by version in descending order.
// Stable versions come before pre-release versions (beta, alpha).
// Within each group, higher versions come first.
func sortAPIsByVersion(apis []string) []string {
	result := make([]string, len(apis))
	copy(result, apis)

	sort.Slice(result, func(i, j int) bool {
		vi := parseVersion(result[i])
		vj := parseVersion(result[j])

		// Stable versions before pre-release
		if vi.stable != vj.stable {
			return vi.stable
		}

		// If both pre-release, beta before alpha
		if !vi.stable {
			if vi.prerelease != vj.prerelease {
				return vi.prerelease > vj.prerelease
			}
		}

		// Higher major version first
		if vi.major != vj.major {
			return vi.major > vj.major
		}

		// Higher pre-release number first
		if !vi.stable {
			return vi.prereleaseNum > vj.prereleaseNum
		}

		return false
	})

	return result
}

type version struct {
	major         int
	stable        bool
	prerelease    int // 2=beta, 1=alpha, 0=other
	prereleaseNum int
}

func parseVersion(apiPath string) version {
	// Extract version from API path (last segment)
	parts := strings.Split(apiPath, "/")
	if len(parts) == 0 {
		return version{}
	}

	versionStr := parts[len(parts)-1]
	if len(versionStr) < 2 || versionStr[0] != 'v' {
		return version{}
	}

	v := version{stable: true}

	// Parse version string (e.g., "v1", "v2beta1", "v1alpha1")
	rest := versionStr[1:] // Remove 'v'

	// Extract major version number
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		v.major = v.major*10 + int(rest[i]-'0')
		i++
	}

	if i < len(rest) {
		// Has pre-release suffix
		v.stable = false
		suffix := rest[i:]

		if strings.HasPrefix(suffix, "beta") {
			v.prerelease = 2
			suffix = suffix[4:]
		} else if strings.HasPrefix(suffix, "alpha") {
			v.prerelease = 1
			suffix = suffix[5:]
		}

		// Extract pre-release number
		for _, c := range suffix {
			if c >= '0' && c <= '9' {
				v.prereleaseNum = v.prereleaseNum*10 + int(c-'0')
			}
		}
	}

	return v
}
