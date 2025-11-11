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

package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSha256(t *testing.T) {
	const (
		latestShaContents     = "The quick brown fox jumps over the lazy dog"
		latestShaContentsHash = "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"
	)

	for _, test := range []struct {
		name       string
		content    string
		statusCode int
		wantSha256 string
		wantErr    bool
	}{
		{
			name:       "success",
			content:    latestShaContents,
			statusCode: http.StatusOK,
			wantSha256: latestShaContentsHash,
			wantErr:    false,
		},
		{
			name:       "empty content",
			content:    "",
			statusCode: http.StatusOK,
			wantSha256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr:    false,
		},
		{
			name:       "http error",
			content:    "error",
			statusCode: http.StatusBadRequest,
			wantSha256: "",
			wantErr:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.statusCode)
				w.Write([]byte(test.content))
			}))
			defer server.Close()

			got, err := GetSha256(server.URL)
			if (err != nil) != test.wantErr {
				t.Errorf("GetSha256() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !test.wantErr && got != test.wantSha256 {
				t.Errorf("GetSha256() = %q, want %q", got, test.wantSha256)
			}
		})
	}
}

func TestGetLatestSha(t *testing.T) {
	const latestSha = "5d5b1bf126485b0e2c972bac41b376438601e266"

	for _, test := range []struct {
		name       string
		response   string
		statusCode int
		wantSha    string
		wantErr    bool
	}{
		{
			name:       "success",
			response:   latestSha,
			statusCode: http.StatusOK,
			wantSha:    latestSha,
			wantErr:    false,
		},
		{
			name:       "http error",
			response:   "ERROR - bad request",
			statusCode: http.StatusBadRequest,
			wantSha:    "",
			wantErr:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the Accept header is set correctly
				got := r.Header.Get("Accept")
				want := "application/vnd.github.VERSION.sha"
				if got != want {
					t.Fatalf("mismatched Accept header, got=%q, want=%q", got, want)
				}
				w.WriteHeader(test.statusCode)
				w.Write([]byte(test.response))
			}))
			defer server.Close()

			got, err := GetLatestSha(server.URL)
			if (err != nil) != test.wantErr {
				t.Errorf("GetLatestSha() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !test.wantErr && got != test.wantSha {
				t.Errorf("GetLatestSha() = %q, want %q", got, test.wantSha)
			}
		})
	}
}
