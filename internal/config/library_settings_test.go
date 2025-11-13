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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeriveLaunchStage(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		want    string
	}{
		{"v1", "google.cloud.secretmanager.v1", "GA"},
		{"v2", "google.cloud.bigquery.v2", "GA"},
		{"v1beta1", "google.cloud.aiplatform.v1beta1", "BETA"},
		{"v1beta", "google.analytics.admin.v1beta", "BETA"},
		{"v1alpha", "google.analytics.admin.v1alpha", "ALPHA"},
		{"v1alpha1", "google.cloud.alloydb.v1alpha", "ALPHA"},
		{"just v1", "v1", "GA"},
		{"just v1beta", "v1beta", "BETA"},
		{"just v1alpha", "v1alpha", "ALPHA"},
		{"empty", "", "GA"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveLaunchStage(test.version)
			if got != test.want {
				t.Errorf("DeriveLaunchStage(%q) = %q, want %q", test.version, got, test.want)
			}
		})
	}
}

func TestDeriveDestinations(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *LibraryConfig
		want []string
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: []string{"PACKAGE_MANAGER"},
		},
		{
			name: "empty config",
			cfg:  &LibraryConfig{},
			want: []string{"PACKAGE_MANAGER"},
		},
		{
			name: "explicit destinations",
			cfg: &LibraryConfig{
				Destinations: []string{"PACKAGE_MANAGER", "GITHUB"},
			},
			want: []string{"PACKAGE_MANAGER", "GITHUB"},
		},
		{
			name: "empty destinations array",
			cfg: &LibraryConfig{
				Destinations: []string{},
			},
			want: []string{"PACKAGE_MANAGER"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveDestinations(test.cfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetLaunchStage(t *testing.T) {
	for _, test := range []struct {
		name       string
		cfg        *LibraryConfig
		apiVersion string
		want       string
	}{
		{
			name:       "derived GA",
			cfg:        nil,
			apiVersion: "google.cloud.secretmanager.v1",
			want:       "GA",
		},
		{
			name:       "derived BETA",
			cfg:        nil,
			apiVersion: "google.cloud.aiplatform.v1beta1",
			want:       "BETA",
		},
		{
			name: "explicit override",
			cfg: &LibraryConfig{
				LaunchStage: "GA",
			},
			apiVersion: "google.cloud.service.v1beta1", // Would derive BETA, but override to GA
			want:       "GA",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetLaunchStage(test.cfg, test.apiVersion)
			if got != test.want {
				t.Errorf("GetLaunchStage() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetJavaPackage(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *LibraryConfig
		want string
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: "",
		},
		{
			name: "no java config",
			cfg:  &LibraryConfig{},
			want: "",
		},
		{
			name: "empty java config",
			cfg: &LibraryConfig{
				Java: &JavaLibrary{},
			},
			want: "",
		},
		{
			name: "with package",
			cfg: &LibraryConfig{
				Java: &JavaLibrary{
					Package: "com.google.cloud.logging.v2",
				},
			},
			want: "com.google.cloud.logging.v2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetJavaPackage(test.cfg)
			if got != test.want {
				t.Errorf("GetJavaPackage() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetJavaServiceClassNames(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *LibraryConfig
		want map[string]string
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: nil,
		},
		{
			name: "no java config",
			cfg:  &LibraryConfig{},
			want: nil,
		},
		{
			name: "empty java config",
			cfg: &LibraryConfig{
				Java: &JavaLibrary{},
			},
			want: nil,
		},
		{
			name: "with service class names",
			cfg: &LibraryConfig{
				Java: &JavaLibrary{
					ServiceClassNames: map[string]string{
						"google.logging.v2.LoggingServiceV2": "Logging",
						"google.logging.v2.ConfigServiceV2":  "Config",
					},
				},
			},
			want: map[string]string{
				"google.logging.v2.LoggingServiceV2": "Logging",
				"google.logging.v2.ConfigServiceV2":  "Config",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetJavaServiceClassNames(test.cfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetPythonSettings(t *testing.T) {
	t.Run("RestAsyncIOEnabled", func(t *testing.T) {
		for _, test := range []struct {
			name string
			cfg  *LibraryConfig
			want bool
		}{
			{"nil config", nil, false},
			{"no python config", &LibraryConfig{}, false},
			{"empty python config", &LibraryConfig{Python: &PythonLibrary{}}, false},
			{"enabled", &LibraryConfig{Python: &PythonLibrary{RestAsyncIOEnabled: true}}, true},
		} {
			t.Run(test.name, func(t *testing.T) {
				got := GetPythonRestAsyncIOEnabled(test.cfg)
				if got != test.want {
					t.Errorf("GetPythonRestAsyncIOEnabled() = %v, want %v", got, test.want)
				}
			})
		}
	})

	t.Run("UnversionedPackageDisabled", func(t *testing.T) {
		for _, test := range []struct {
			name string
			cfg  *LibraryConfig
			want bool
		}{
			{"nil config", nil, false},
			{"no python config", &LibraryConfig{}, false},
			{"empty python config", &LibraryConfig{Python: &PythonLibrary{}}, false},
			{"disabled", &LibraryConfig{Python: &PythonLibrary{UnversionedPackageDisabled: true}}, true},
		} {
			t.Run(test.name, func(t *testing.T) {
				got := GetPythonUnversionedPackageDisabled(test.cfg)
				if got != test.want {
					t.Errorf("GetPythonUnversionedPackageDisabled() = %v, want %v", got, test.want)
				}
			})
		}
	})
}

func TestGetGoRenamedServices(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *LibraryConfig
		want map[string]string
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: nil,
		},
		{
			name: "no go config",
			cfg:  &LibraryConfig{},
			want: nil,
		},
		{
			name: "empty go config",
			cfg: &LibraryConfig{
				Go: &GoLibrary{},
			},
			want: nil,
		},
		{
			name: "with renamed services",
			cfg: &LibraryConfig{
				Go: &GoLibrary{
					RenamedServices: map[string]string{
						"Publisher":  "TopicAdmin",
						"Subscriber": "SubscriptionAdmin",
					},
				},
			},
			want: map[string]string{
				"Publisher":  "TopicAdmin",
				"Subscriber": "SubscriptionAdmin",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetGoRenamedServices(test.cfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetNodeSelectiveMethods(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *LibraryConfig
		want []string
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: nil,
		},
		{
			name: "no node config",
			cfg:  &LibraryConfig{},
			want: nil,
		},
		{
			name: "empty node config",
			cfg: &LibraryConfig{
				Node: &NodeLibrary{},
			},
			want: nil,
		},
		{
			name: "with selective methods",
			cfg: &LibraryConfig{
				Node: &NodeLibrary{
					SelectiveMethods: []string{
						"google.storage.v2.Storage.GetBucket",
						"google.storage.v2.Storage.CreateBucket",
					},
				},
			},
			want: []string{
				"google.storage.v2.Storage.GetBucket",
				"google.storage.v2.Storage.CreateBucket",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := GetNodeSelectiveMethods(test.cfg)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetDotnetSettings(t *testing.T) {
	t.Run("RenamedServices", func(t *testing.T) {
		for _, test := range []struct {
			name string
			cfg  *LibraryConfig
			want map[string]string
		}{
			{"nil config", nil, nil},
			{"no dotnet config", &LibraryConfig{}, nil},
			{"empty dotnet config", &LibraryConfig{Dotnet: &DotnetLibrary{}}, nil},
			{
				"with renamed services",
				&LibraryConfig{
					Dotnet: &DotnetLibrary{
						RenamedServices: map[string]string{
							"Publisher": "PublisherServiceApi",
						},
					},
				},
				map[string]string{
					"Publisher": "PublisherServiceApi",
				},
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				got := GetDotnetRenamedServices(test.cfg)
				if diff := cmp.Diff(test.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("RenamedResources", func(t *testing.T) {
		for _, test := range []struct {
			name string
			cfg  *LibraryConfig
			want map[string]string
		}{
			{"nil config", nil, nil},
			{"no dotnet config", &LibraryConfig{}, nil},
			{"empty dotnet config", &LibraryConfig{Dotnet: &DotnetLibrary{}}, nil},
			{
				"with renamed resources",
				&LibraryConfig{
					Dotnet: &DotnetLibrary{
						RenamedResources: map[string]string{
							"automl.googleapis.com/Dataset": "AutoMLDataset",
						},
					},
				},
				map[string]string{
					"automl.googleapis.com/Dataset": "AutoMLDataset",
				},
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				got := GetDotnetRenamedResources(test.cfg)
				if diff := cmp.Diff(test.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})
}
