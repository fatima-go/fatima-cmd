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
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
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

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}
