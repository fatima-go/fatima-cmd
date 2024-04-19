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

package share

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func CallFatimaApi(url string, flags FatimaCmdFlags, b []byte) (http.Header, []byte, error) {
	var netTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 2 * time.Second,
	}

	client := &http.Client{
		Timeout:   time.Second * 15,
		Transport: netTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	headers := flags.BuildHeader()
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
	if b != nil && len(b) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(b))
	}

	if flags.Debug {
		h, _ := json.Marshal(req.Header)
		fmt.Printf("=== POST %s ============>\n", url)
		fmt.Printf("headers : %v\n", string(h))
		if b != nil && len(b) > 0 {
			fmt.Printf("body : %v\n", screenPasswordString(b))
		} else {
			fmt.Printf("body : \n")
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if flags.Debug {
		fmt.Printf("<== %s =============\n", resp.Status)
		h, _ := json.Marshal(resp.Header)
		fmt.Printf("headers : %v\n", string(h))
	}

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		if respBytes == nil {
			return resp.Header, nil, fmt.Errorf("invalid response : %d", resp.StatusCode)
		}
		resBody := string(respBytes)
		return resp.Header, nil, fmt.Errorf("invalid response : %d\n%s", resp.StatusCode, resBody)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, nil, fmt.Errorf("invalid response body : %s", err.Error())
	}

	if flags.Debug {
		fmt.Printf("body : %v\n", string(respBytes))
	}

	return resp.Header, respBytes, nil
}

func screenPasswordString(b []byte) string {
	m := make(map[string]interface{})
	err := json.Unmarshal(b, &m)
	if err != nil {
		return string(b)
	}

	replaced := screenPasswordMap(m)

	b2, err := json.Marshal(replaced)
	if err != nil {
		return string(b)
	}
	return string(b2)
}

func screenPasswordMap(m map[string]interface{}) map[string]interface{} {
	replaced := make(map[string]interface{})
	for k, v := range m {
		k1 := strings.ToLower(k)

		if m2, ok := v.(map[string]interface{}); ok {
			v2 := screenPasswordMap(m2)
			replaced[k] = v2
			continue
		}

		if k1 == "passwd" || k1 == "password" {
			replaced[k] = "########"
			continue
		}

		replaced[k] = v
	}

	return replaced
}

func isSuccess(m map[string]interface{}) bool {
	system := m["system"]
	if system == nil {
		fmt.Printf("Not found system message from server")
		return false
	}
	if val, ok := system.(map[string]interface{}); ok {
		var cval int
		code := val["code"]
		if v1, ok := code.(float64); ok {
			cval = int(v1)
		} else {
			fmt.Printf("invalid code type. real type=%v\n", reflect.ValueOf(code).Type())
			return false
		}
		if cval == 200 {
			return true
		} else {
			return false
		}
	}
	fmt.Printf("Invalid system message structure. real type=%v\n", reflect.ValueOf(system).Type())
	return false
}

const (
	yyyyMMddHHmmss = "2006-01-02 15:04:05"
)

func CallFarUpload(url string, flags FatimaCmdFlags, desc map[string]interface{}, path string) (http.Header, []byte, error) {
	b, _ := json.Marshal(desc)

	var netTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 2 * time.Second,
	}

	client := &http.Client{
		Timeout:   time.Second * 60,
		Transport: netTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := newfileUploadRequest(url, string(b), path)
	if err != nil {
		return nil, nil, err
	}

	headers := flags.BuildHeader()
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	if flags.Debug {
		h, _ := json.Marshal(req.Header)
		fmt.Printf("=== POST %s ============>\n", url)
		fmt.Printf("headers : %v\n", string(h))
		fmt.Printf("body : json[%v], file[%s]\n", desc, path)
	}

	fmt.Printf("%s start transfer...\n", time.Now().Format(yyyyMMddHHmmss))
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if flags.Debug {
		fmt.Printf("<== %s =============\n", resp.Status)
		h, _ := json.Marshal(resp.Header)
		fmt.Printf("headers : %v\n", string(h))
	}

	if resp.StatusCode != http.StatusOK {
		return resp.Header, nil, fmt.Errorf("invalid response : %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, nil, fmt.Errorf("invalid response body : %s", err.Error())
	}

	if flags.Debug {
		fmt.Printf("body : %v\n", string(respBytes))
	}

	return resp.Header, respBytes, nil
}

func newfileUploadRequest(uri string, far string, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("far", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	_ = writer.WriteField("json", far)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	reader := &CustomBodyReader{buff: body}
	req, err := http.NewRequest("POST", uri, reader)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

type CustomBodyReader struct {
	written int
	buff    *bytes.Buffer
}

func (c *CustomBodyReader) Read(p []byte) (n int, err error) {
	n, err = c.buff.Read(p)
	if errors.Is(err, io.EOF) {
		fmt.Printf("%s transfer %s finished.  waiting server response...\n",
			time.Now().Format(yyyyMMddHHmmss), ByteSize(uint64(c.written)))
	}
	c.written += n
	return
}

const (
	BYTE = 1.0 << (10 * iota)
	KILOBYTE
	MEGABYTE
	GIGABYTE
	TERABYTE
)

var (
	bytesPattern             = regexp.MustCompile(`(?i)^(-?\d+(?:\.\d+)?)([KMGT]i?B?|B)$`)
	invalidByteQuantityError = errors.New("byte quantity must be a positive integer with a unit of measurement like M, MB, MiB, G, GiB, or GB")
)

// ByteSize returns a human-readable byte string of the form 10M, 12.5K, and so forth.  The following units are available:
//
//	T: Terabyte
//	G: Gigabyte
//	M: Megabyte
//	K: Kilobyte
//	B: Byte
//
// The unit that results in the smallest number greater than or equal to 1 is always chosen.
func ByteSize(bytes uint64) string {
	unit := ""
	value := float32(bytes)

	switch {
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes >= BYTE:
		unit = "B"
	case bytes == 0:
		return "0"
	}

	stringValue := fmt.Sprintf("%.1f", value)
	stringValue = strings.TrimSuffix(stringValue, ".0")
	return fmt.Sprintf("%s%s", stringValue, unit)
}

// ToMegabytes parses a string formatted by ByteSize as megabytes.
func ToMegabytes(s string) (uint64, error) {
	bytes, err := ToBytes(s)
	if err != nil {
		return 0, err
	}

	return bytes / MEGABYTE, nil
}

// ToBytes parses a string formatted by ByteSize as bytes. Note binary-prefixed and SI prefixed units both mean a base-2 units
// KB = K = KiB	= 1024
// MB = M = MiB = 1024 * K
// GB = G = GiB = 1024 * M
// TB = T = TiB = 1024 * G
func ToBytes(s string) (uint64, error) {
	parts := bytesPattern.FindStringSubmatch(strings.TrimSpace(s))
	if len(parts) < 3 {
		return 0, invalidByteQuantityError
	}

	value, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || value <= 0 {
		return 0, invalidByteQuantityError
	}

	var bytes uint64
	unit := strings.ToUpper(parts[2])
	switch unit[:1] {
	case "T":
		bytes = uint64(value * TERABYTE)
	case "G":
		bytes = uint64(value * GIGABYTE)
	case "M":
		bytes = uint64(value * MEGABYTE)
	case "K":
		bytes = uint64(value * KILOBYTE)
	case "B":
		bytes = uint64(value * BYTE)
	}

	return bytes, nil
}
