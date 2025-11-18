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

	"github.com/julieqiu/librarianx/internal/config"
	"github.com/julieqiu/librarianx/internal/language/internal/python"
	"github.com/julieqiu/librarianx/internal/language/internal/rust"
)

// Init initializes a default config for the given language.
func Init(language string) (*config.Default, error) {
	switch language {
	case "rust":
		if err := rust.SetupWorkspace("."); err != nil {
			return nil, err
		}
		return rust.Init(), nil
	case "python":
		return python.Init(), nil
	}
	return nil, fmt.Errorf("not supported: %q", language)
}
