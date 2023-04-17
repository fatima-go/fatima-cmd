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
	"net/http"
	"os"
)

func GetString(m map[string]interface{}, key string) string {
	val, ok := m[key]
	if !ok {
		return ""
	}

	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

func GetMap(m map[string]interface{}, key string) map[string]interface{} {
	val, ok := m[key]
	if !ok {
		return make(map[string]interface{})
	}

	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	return make(map[string]interface{})
}

func GetList(m map[string]interface{}, key string) []interface{} {
	empty := make([]interface{}, 0)
	val, ok := m[key]
	if !ok {
		return empty
	}

	if m, ok := val.([]interface{}); ok {
		return m
	}
	return empty
}

func GetInt(m map[string]interface{}, key string) int {
	val, ok := m[key]
	if !ok {
		return -1
	}

	if f, ok := val.(float64); ok {
		return int(f)
	}
	return -1
}

func GetStringFromHeader(header http.Header, key string) string {
	val := header[key]
	if val == nil || len(val) == 0 {
		return ""
	}

	return val[0]
}

func AsString(val interface{}) string {
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

func AsInt(val interface{}) int {
	if f, ok := val.(float64); ok {
		return int(f)
	}
	return 0
}

func AsHaString(val int) string {
	switch val {
	case 1:
		return "ACTIVE"
	case 2:
		return "STANDBY"
	}
	return "UNKNOWN"
}

func AsPsString(val int) string {
	switch val {
	case 1:
		return "PRIMARY"
	case 2:
		return "SECONDARY"
	}
	return "UNKNOWN"
}

func IsFileExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
