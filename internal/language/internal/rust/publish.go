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
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	rustrelease "github.com/googleapis/librarian/internal/sidekick/rust_release"
)

// Publish publishes all crates that have changed since the last release.
func Publish(ctx context.Context, cfg *config.Config, dryRun bool, skipSemverChecks bool) error {
	if cfg.Default == nil || cfg.Default.Release == nil {
		return fmt.Errorf("default.release configuration is required")
	}

	// Convert librarian config to sidekick config
	releaseConfig := toSidekickReleaseConfig(cfg.Default.Release)

	return rustrelease.Publish(releaseConfig, dryRun, skipSemverChecks)
}

// toSidekickReleaseConfig converts librarian config to sidekick config format.
func toSidekickReleaseConfig(release *config.DefaultRelease) *sidekickconfig.Release {
	remote := release.Remote
	if remote == "" {
		remote = "origin"
	}
	branch := release.Branch
	if branch == "" {
		branch = "main"
	}

	return &sidekickconfig.Release{
		Remote: remote,
		Branch: branch,
	}
}
