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

// Package request provides types and functions for parsing librarian tool requests.
package request

import "fmt"

// Change represents a single commit change for a library.
type Change struct {
	Type          string `json:"type"`
	Subject       string `json:"subject"`
	Body          string `json:"body"`
	PiperCLNumber string `json:"piper_cl_number"`
	CommitHash    string `json:"commit_hash"`
}

// ConventionalCommit returns a conventional commit message string.
func (c *Change) ConventionalCommit() string {
	if c.Body == "" {
		return fmt.Sprintf("%s: %s", c.Type, c.Subject)
	}
	return fmt.Sprintf("%s: %s\n\n%s", c.Type, c.Subject, c.Body)
}
