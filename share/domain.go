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
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type PackageInfo struct {
	Group    string
	Host     string
	Name     string
	Platform string
}

func (p PackageInfo) Valid() bool {
	if len(p.Group) == 0 || len(p.Host) == 0 {
		return false
	}
	return true
}

func (p PackageInfo) String() string {
	if len(p.Platform) == 0 {
		return fmt.Sprintf("[%s] %s:%s", p.Group, p.Host, p.Name)
	}
	return fmt.Sprintf("[%s] %s:%s %s", p.Group, p.Host, p.Name, p.Platform)
}

func NewPackageInfo(m map[string]interface{}) PackageInfo {
	p := PackageInfo{Name: "default"}

	if m == nil {
		return p
	}

	p.Group = GetString(m, "package_group")
	p.Host = GetString(m, "package_host")

	summaryObj := m["summary"]
	summary, ok := summaryObj.(map[string]interface{})
	if !ok {
		return p
	}

	p.Name = GetString(summary, "package_name")

	platformObj := m["platform"]
	platform, ok := platformObj.(map[string]interface{})
	if ok {
		os := platform["os"]
		arch := platform["architecture"]
		p.Platform = fmt.Sprintf("(%s_%s)", os, arch)
	}

	return p
}

func GetSummaryMessage(m map[string]interface{}) string {
	summary := m["summary"]
	if summary == nil {
		fmt.Printf("Not found summary message from server")
		return ""
	}

	if val, ok := summary.(map[string]interface{}); ok {
		return GetString(val, "message")
	}

	fmt.Printf("Not found message in summary")
	return ""
}

func GetSummaryHistory(m map[string]interface{}) []interface{} {
	empty := make([]interface{}, 0)
	summary := m["summary"]
	if summary == nil {
		fmt.Printf("Not found summary message from server")
		return empty
	}

	if val, ok := summary.(map[string]interface{}); ok {
		return GetList(val, "history")
	}

	fmt.Printf("Not found message in summary")
	return empty
}

// GetKeyInMap get key(path) in map. e.g) findingKey = "data.sub.myKey"
func GetKeyInMap(sourceMap map[string]interface{}, findingKey string) string {
	keyPath := strings.Split(findingKey, ".")
	for i, key := range keyPath {
		found, ok := sourceMap[key]
		if !ok {
			break
		}
		if i == len(keyPath)-1 {
			return printString(found)
		}

		// 'found' should be map type
		switch v := found.(type) {
		case map[string]interface{}:
			sourceMap = v
		default:
			// key does not map type value
			return ""
		}
	}

	// not found findingKey
	return ""
}

func printString(src interface{}) string {
	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	rv := reflect.ValueOf(src)
	switch rv.Kind() {
	case reflect.Float64:
		return fmt.Sprintf("%d", int(rv.Float()))
		//return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	}
	return fmt.Sprintf("%v", src)
}

func GetSystemMessage(m map[string]interface{}) string {
	summary := m["system"]
	if summary == nil {
		fmt.Printf("Not found summary message from server")
		return ""
	}

	if val, ok := summary.(map[string]interface{}); ok {
		return GetString(val, "message")
	}

	fmt.Printf("Not found message in system")
	return ""
}
