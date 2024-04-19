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
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
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
