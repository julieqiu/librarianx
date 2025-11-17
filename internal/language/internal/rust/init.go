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
	"github.com/julieqiu/librarianx/internal/config"
)

// ConfigDefault initializes a default Rust config.
func ConfigDefault() *config.Default {
	return &config.Default{
		Output: "src/generated/",
		Generate: &config.DefaultGenerate{
			All:           true,
			OneLibraryPer: "version",
			ReleaseLevel:  "stable",
		},

		Release: &config.DefaultRelease{
			TagFormat: "{name}/v{version}",
			Remote:    "upstream",
			Branch:    "main",
		},

		Rust: &config.RustDefault{
			DisabledRustdocWarnings: []string{
				"redundant_explicit_links",
				"broken_intra_doc_links",
			},

			PackageDependencies: []*config.RustPackageDependency{
				{
					Name:    "api",
					Package: "google-cloud-api",
					Source:  "google.api",
				},
				{
					Name:    "async-trait",
					Package: "async-trait",
					UsedIf:  "services",
				},
				{
					Name:      "bytes",
					Package:   "bytes",
					ForceUsed: true,
				},
				{
					Name:    "cloud_common",
					Package: "google-cloud-common",
					Source:  "google.cloud.common",
				},
				{
					Name:    "gax",
					Package: "google-cloud-gax",
					UsedIf:  "services",
				},
				{
					Name:    "gaxi",
					Package: "google-cloud-gax-internal",
					UsedIf:  "services",
					Feature: "_internal-http-client",
				},
				{
					Name:    "grafeas",
					Package: "google-cloud-grafeas-v1",
					Source:  "grafeas.v1",
				},
				{
					Name:    "gtype",
					Package: "google-cloud-type",
					Source:  "google.type",
				},
				{
					Name:    "iam_v1",
					Package: "google-cloud-iam-v1",
					Source:  "google.iam.v1",
				},
				{
					Name:    "lazy_static",
					Package: "lazy_static",
					UsedIf:  "services",
				},
				{
					Name:    "location",
					Package: "google-cloud-location",
					Source:  "google.cloud.location",
				},
				{
					Name:    "logging_type",
					Package: "google-cloud-logging-type",
					Source:  "google.logging.type",
				},
				{
					Name:    "longrunning",
					Package: "google-cloud-longrunning",
					Source:  "google.longrunning",
				},
				{
					Name:    "lro",
					Package: "google-cloud-lro",
					UsedIf:  "lro",
				},
				{
					Name:    "reqwest",
					Package: "reqwest",
					UsedIf:  "services",
				},
				{
					Name:    "rpc",
					Package: "google-cloud-rpc",
					Source:  "google.rpc",
				},
				{
					Name:    "rpc_context",
					Package: "google-cloud-rpc-context",
					Source:  "google.rpc.context",
				},
				{
					Name:      "serde",
					Package:   "serde",
					ForceUsed: true,
				},
				{
					Name:      "serde_json",
					Package:   "serde_json",
					ForceUsed: true,
				},
				{
					Name:      "serde_with",
					Package:   "serde_with",
					ForceUsed: true,
				},
				{
					Name:    "tracing",
					Package: "tracing",
					UsedIf:  "services",
				},
				{
					Name:    "uuid",
					Package: "uuid",
					UsedIf:  "autopopulated",
				},
				{
					Name:      "wkt",
					Package:   "google-cloud-wkt",
					Source:    "google.protobuf",
					ForceUsed: true,
				},
			},
		},
	}
}
