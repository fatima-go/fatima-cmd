/*
 * Copyright 2026 github.com/fatima-go
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package juno

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fatima-go/fatima-cmd/share"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadJunoClipBinaryOverwritesAfterSuccessfulTransfer(t *testing.T) {
	filename := "mydata.tar.gz"
	expected := []byte{0x00, 0x01, 0x7f, 0xff}
	outputDir := t.TempDir()
	target := filepath.Join(outputDir, filename)
	require.NoError(t, os.WriteFile(target, []byte("old data"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/clip/v1", req.URL.Path)
		assert.Equal(t, clipBinaryContentType, req.Header.Get("Accept"))
		assert.Equal(t, "test-token", req.Header.Get("Fatima-Auth-Token"))

		var request clipBinaryRequest
		require.NoError(t, json.NewDecoder(req.Body).Decode(&request))
		assert.Equal(t, filename, request.Filename)

		res.Header().Set("Content-Type", clipBinaryContentType)
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write(expected)
	}))
	defer server.Close()

	err := downloadJunoClipBinary(testClipFlags(server.URL), filename, outputDir)
	require.NoError(t, err)
	actual, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
	assertNoClipTemporaryFiles(t, outputDir)
}

func TestDownloadJunoClipBinaryFallsBackForLegacyJuno(t *testing.T) {
	outputDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = fmt.Fprintln(res, `{"content":"legacy clipboard"}`)
	}))
	defer server.Close()

	err := downloadJunoClipBinary(testClipFlags(server.URL), "mydata.tar.gz", outputDir)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "mydata.tar.gz"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	assertNoClipTemporaryFiles(t, outputDir)
}

func TestDownloadJunoClipBinaryPreservesExistingFileOnServerError(t *testing.T) {
	filename := "mydata.tar.gz"
	outputDir := t.TempDir()
	target := filepath.Join(outputDir, filename)
	require.NoError(t, os.WriteFile(target, []byte("old data"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		res.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintln(res, `{"message":"file not found: mydata.tar.gz"}`)
	}))
	defer server.Close()

	err := downloadJunoClipBinary(testClipFlags(server.URL), filename, outputDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
	assertFileContent(t, target, []byte("old data"))
	assertNoClipTemporaryFiles(t, outputDir)
}

func TestDownloadJunoClipBinaryPreservesExistingFileOnInterruptedTransfer(t *testing.T) {
	filename := "mydata.tar.gz"
	outputDir := t.TempDir()
	target := filepath.Join(outputDir, filename)
	require.NoError(t, os.WriteFile(target, []byte("old data"), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", clipBinaryContentType)
		res.Header().Set("Content-Length", "10")
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte("short"))
	}))
	defer server.Close()

	err := downloadJunoClipBinary(testClipFlags(server.URL), filename, outputDir)
	require.Error(t, err)
	assertFileContent(t, target, []byte("old data"))
	assertNoClipTemporaryFiles(t, outputDir)
}

func TestDownloadJunoClipBinaryRejectsOversizedResponseBeforeWriting(t *testing.T) {
	filename := "mydata.tar.gz"
	outputDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", clipBinaryContentType)
		res.Header().Set("Content-Length", strconv.FormatInt(maxClipBinaryBytes+1, 10))
		res.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := downloadJunoClipBinary(testClipFlags(server.URL), filename, outputDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the size limit")
	_, err = os.Stat(filepath.Join(outputDir, filename))
	assert.ErrorIs(t, err, os.ErrNotExist)
	assertNoClipTemporaryFiles(t, outputDir)
}

func TestSaveClipBinaryFileHandlesZeroLengthAndLengthMismatch(t *testing.T) {
	outputDir := t.TempDir()

	written, err := saveClipBinaryFile(outputDir, "empty.bin", strings.NewReader(""), 0)
	require.NoError(t, err)
	assert.Zero(t, written)
	assertFileContent(t, filepath.Join(outputDir, "empty.bin"), []byte{})

	target := filepath.Join(outputDir, "existing.bin")
	require.NoError(t, os.WriteFile(target, []byte("old data"), 0600))
	_, err = saveClipBinaryFile(outputDir, "existing.bin", strings.NewReader("short"), 10)
	require.Error(t, err)
	assertFileContent(t, target, []byte("old data"))
	assertNoClipTemporaryFiles(t, outputDir)
}

func TestValidateClipBinaryFilename(t *testing.T) {
	assert.NoError(t, validateClipBinaryFilename("mydata.tar.gz"))
	assert.NoError(t, validateClipBinaryFilename("한글.bin"))

	for _, filename := range []string{"", ".", "..", "../secret", "/tmp/secret", `folder\\secret`, "bad\nname"} {
		t.Run(filename, func(t *testing.T) {
			assert.Error(t, validateClipBinaryFilename(filename))
		})
	}
}

func testClipFlags(endpoint string) share.FatimaCmdFlags {
	return share.FatimaCmdFlags{
		Endpoint: endpoint,
		Token:    "test-token",
	}
}

func assertFileContent(t *testing.T, path string, expected []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func assertNoClipTemporaryFiles(t *testing.T, outputDir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(outputDir, ".roclip-*"))
	require.NoError(t, err)
	assert.Empty(t, matches)
}
