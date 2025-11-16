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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLatestCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"sha": "abc123def456"}`))
	}))
	defer server.Close()

	got, err := latestCommit(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	want := "abc123def456"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDownloadAndExtractTarball(t *testing.T) {
	tarball := createTestTarball(t, map[string]string{
		"test-repo-abc123/README.md":       "# Test\n",
		"test-repo-abc123/google/api/api.proto": "syntax = \"proto3\";\n",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test-repo/archive/abc123.tar.gz" {
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(tarball)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	originalTransport := http.DefaultTransport
	http.DefaultTransport = &redirectTransport{
		testServerURL: server.URL,
		base:          originalTransport,
	}
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	cacheDir := t.TempDir()

	got, err := DownloadAndExtractTarball("example.com/test-repo", "abc123", cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}

	readmeContent, err := os.ReadFile(filepath.Join(got, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "# Test\n"
	if diff := cmp.Diff(want, string(readmeContent)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	protoContent, err := os.ReadFile(filepath.Join(got, "google/api/api.proto"))
	if err != nil {
		t.Fatal(err)
	}
	want = "syntax = \"proto3\";\n"
	if diff := cmp.Diff(want, string(protoContent)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got2, err := DownloadAndExtractTarball("example.com/test-repo", "abc123", cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, got2); diff != "" {
		t.Errorf("second call should return same path, mismatch (-want +got):\n%s", diff)
	}
}

type redirectTransport struct {
	testServerURL string
	base          http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "https" {
		newURL := t.testServerURL + req.URL.Path
		newReq, err := http.NewRequest(req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header
		return t.base.RoundTrip(newReq)
	}
	return t.base.RoundTrip(req)
}

func createTestTarball(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}
