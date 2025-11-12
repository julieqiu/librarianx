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

// Package python provides utilities for generating Python client libraries.
package python

import (
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/generate/golang/request"
)

// ProtocCommand represents a protoc command with its arguments.
type ProtocCommand struct {
	Command string
	Args    []string
}

// BuildGapicCommand constructs the protoc command for GAPIC generation.
func BuildGapicCommand(api *request.API, sourceDir, outputDir string, opts *GapicOptions) (*ProtocCommand, error) {
	if api.Path == "" {
		return nil, fmt.Errorf("API path cannot be empty")
	}

	apiDir := filepath.Join(sourceDir, api.Path)
	protoPattern := filepath.Join(apiDir, "*.proto")

	args := []string{
		"--proto_path=" + sourceDir,
		"--python_gapic_out=" + outputDir,
	}

	// Build gapic options
	var gapicOpts []string
	if opts != nil {
		gapicOpts = append(gapicOpts, "metadata")

		if opts.GrpcServiceConfig != "" {
			gapicOpts = append(gapicOpts, fmt.Sprintf("retry-config=%s", filepath.Join(apiDir, opts.GrpcServiceConfig)))
		}
		if opts.ServiceYAML != "" {
			gapicOpts = append(gapicOpts, fmt.Sprintf("service-yaml=%s", filepath.Join(apiDir, opts.ServiceYAML)))
		}
		if opts.Transport != "" {
			gapicOpts = append(gapicOpts, fmt.Sprintf("transport=%s", opts.Transport))
		}
		if opts.RestNumericEnums {
			gapicOpts = append(gapicOpts, "rest-numeric-enums")
		}
		for _, opt := range opts.OptArgs {
			gapicOpts = append(gapicOpts, opt)
		}
	}

	if len(gapicOpts) > 0 {
		optsStr := ""
		for i, opt := range gapicOpts {
			if i > 0 {
				optsStr += ","
			}
			optsStr += opt
		}
		args = append(args, "--python_gapic_opt="+optsStr)
	}

	args = append(args, protoPattern)

	return &ProtocCommand{
		Command: "protoc",
		Args:    args,
	}, nil
}

// BuildProtoCommand constructs the protoc command for proto-only generation.
func BuildProtoCommand(api *request.API, sourceDir, outputDir string) (*ProtocCommand, error) {
	if api.Path == "" {
		return nil, fmt.Errorf("API path cannot be empty")
	}

	apiDir := filepath.Join(sourceDir, api.Path)
	protoPattern := filepath.Join(apiDir, "*.proto")

	args := []string{
		"--proto_path=" + sourceDir,
		"--python_out=" + outputDir,
		"--pyi_out=" + outputDir,
		protoPattern,
	}

	return &ProtocCommand{
		Command: "protoc",
		Args:    args,
	}, nil
}

// GapicOptions contains options for GAPIC generation.
type GapicOptions struct {
	GrpcServiceConfig string
	ServiceYAML       string
	Transport         string
	RestNumericEnums  bool
	OptArgs           []string
}
