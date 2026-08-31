/*
 * Copyright 2023 github.com/fatima-go
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
 *
 * @project fatima-core
 * @author jin
 * @date 23. 4. 14. 오후 5:07
 */

package juno

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/fatima-go/fatima-cmd/share"
)

func PrintJunoClipboard(flags share.FatimaCmdFlags) error {
	serviceUrl := flags.BuildJunoServiceUrl(v1ClipboardDisUrl)

	headers, resp, err := callJuno(serviceUrl, flags, nil)
	if err != nil {
		return err
	}
	return printJunoClipboard(headers, resp)
}

func printJunoClipboard(headers http.Header, resp map[string]interface{}) error {
	content, ok := resp["content"].(string)
	if !ok {
		return fmt.Errorf("invalid clipboard response")
	}
	share.PrintPreface(headers, resp)
	fmt.Printf("\n%s", content)
	return nil
}

func DownloadJunoClipBinary(flags share.FatimaCmdFlags, filename string) error {
	if err := validateClipBinaryFilename(filename); err != nil {
		return err
	}

	outputDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("fail to determine current directory: %w", err)
	}
	return downloadJunoClipBinary(flags, filename, outputDir)
}

func downloadJunoClipBinary(flags share.FatimaCmdFlags, filename string, outputDir string) error {
	requestBody, err := json.Marshal(clipBinaryRequest{Filename: filename})
	if err != nil {
		return fmt.Errorf("fail to build binary clip request: %w", err)
	}

	serviceURL := flags.BuildJunoServiceUrl(v1ClipboardDisUrl)
	resp, err := share.CallFatimaStream(serviceURL, flags, requestBody, map[string]string{
		"Accept": clipBinaryContentType,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return clipBinaryResponseError(resp)
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("invalid response content type: %q", resp.Header.Get("Content-Type"))
	}

	switch mediaType {
	case clipBinaryContentType:
		written, err := saveClipBinaryFile(outputDir, filename, resp.Body, resp.ContentLength)
		if err != nil {
			return err
		}
		fmt.Printf("saved %s (%s)\n", filename, share.ByteSize(uint64(written)))
		return nil
	case clipJSONContentType:
		return fallbackToJunoClipboard(resp.Header, resp.Body)
	default:
		return fmt.Errorf("unexpected response content type: %s", mediaType)
	}
}

func fallbackToJunoClipboard(headers http.Header, body io.Reader) error {
	responseBody, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("invalid clipboard response body: %w", err)
	}

	var resp map[string]interface{}
	if err = json.Unmarshal(responseBody, &resp); err != nil {
		return fmt.Errorf("invalid clipboard response structure: %w", err)
	}
	if _, ok := resp["content"].(string); !ok {
		return fmt.Errorf("invalid clipboard response")
	}

	fmt.Fprintln(os.Stderr, "target juno does not support binary mode; displaying clipboard instead")
	return printJunoClipboard(headers, resp)
}

func clipBinaryResponseError(resp *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxClipErrorResponseBytes))
	if readErr != nil {
		return fmt.Errorf("invalid response: %d (%s)", resp.StatusCode, readErr.Error())
	}

	message := strings.TrimSpace(string(body))
	var serverError struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &serverError) == nil && serverError.Message != "" {
		message = serverError.Message
	}
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("invalid response: %d (%s)", resp.StatusCode, message)
}

func saveClipBinaryFile(outputDir string, filename string, body io.Reader, contentLength int64) (int64, error) {
	if contentLength > maxClipBinaryBytes {
		return 0, fmt.Errorf("binary file exceeds the size limit: %d > %d bytes", contentLength, maxClipBinaryBytes)
	}

	tempFile, err := os.CreateTemp(outputDir, ".roclip-"+filename+"-*")
	if err != nil {
		return 0, fmt.Errorf("fail to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	closed := false
	committed := false
	defer func() {
		if !closed {
			_ = tempFile.Close()
		}
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()

	written, err := io.Copy(tempFile, io.LimitReader(body, maxClipBinaryBytes+1))
	if err != nil {
		return written, fmt.Errorf("fail to download binary file: %w", err)
	}
	if written > maxClipBinaryBytes {
		return written, fmt.Errorf("binary file exceeds the size limit: %d > %d bytes", written, maxClipBinaryBytes)
	}
	if contentLength >= 0 && written != contentLength {
		return written, fmt.Errorf("incomplete binary file: expected %d bytes, received %d", contentLength, written)
	}
	if err = tempFile.Sync(); err != nil {
		return written, fmt.Errorf("fail to sync binary file: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		return written, fmt.Errorf("fail to close binary file: %w", err)
	}
	closed = true

	targetPath := filepath.Join(outputDir, filename)
	if err = os.Rename(tempPath, targetPath); err != nil {
		return written, fmt.Errorf("fail to replace %s: %w", filename, err)
	}
	committed = true
	return written, nil
}

func validateClipBinaryFilename(filename string) error {
	if filename == "" || filename == "." || filename == ".." || filepath.IsAbs(filename) {
		return fmt.Errorf("invalid binary filename: %q", filename)
	}
	if filepath.Clean(filename) != filename || filepath.Base(filename) != filename || strings.ContainsAny(filename, "/\\") {
		return fmt.Errorf("invalid binary filename: %q", filename)
	}
	for _, r := range filename {
		if unicode.IsControl(r) {
			return fmt.Errorf("invalid binary filename: %q", filename)
		}
	}
	return nil
}

type clipBinaryRequest struct {
	Filename string `json:"filename"`
}

const (
	v1ClipboardDisUrl               = "clip/v1"
	clipBinaryContentType           = "application/octet-stream"
	clipJSONContentType             = "application/json"
	maxClipBinaryBytes        int64 = 1 << 30
	maxClipErrorResponseBytes       = 64 << 10
)
