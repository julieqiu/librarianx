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
	"fmt"

	"github.com/googleapis/librarian/internal/language/internal/python"
	"github.com/googleapis/librarian/internal/language/internal/rust"
)

// DeriveLibraryName converts an API path to a standard library name for the given language.
func DeriveLibraryName(language, apiPath string) (string, error) {
	switch language {
	case "rust":
		return rust.DeriveLibraryName(apiPath), nil
	case "python":
		return python.DeriveLibraryName(apiPath), nil
	default:
		return "", fmt.Errorf("unsupported language: %s", language)
	}
}

// DeriveAPIPath converts a library name to the likely API path for the given language.
func DeriveAPIPath(language, libraryName string) (string, error) {
	switch language {
	case "rust":
		return rust.DeriveAPIPath(libraryName), nil
	case "python":
		return python.DeriveAPIPath(libraryName), nil
	default:
		return "", fmt.Errorf("unsupported language: %s", language)
	}
}
