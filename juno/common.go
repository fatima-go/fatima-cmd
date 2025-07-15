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
	"github.com/fatima-go/fatima-cmd/share"
	"net/http"
	"strings"
)

func NewSortingOption(sort string) SortingOption {
	s := SortingOption{}
	s.sortType = ToSortType(sort)
	if s.sortType == SortTypeNone {
		s.sortType = SortTypeName // default
	}
	s.order = OrderAsc // default
	return s
}

type SortingOption struct {
	sortType SortType
	order    Order
}

const (
	SortTypeNone  = "none"
	SortTypeName  = "name"
	SortTypeIndex = "index"
)

type SortType string

func ToSortType(s string) SortType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "name":
		return SortTypeName
	case "index":
		return SortTypeIndex
	}
	return SortTypeNone
}

const (
	OrderNone = "none"
	OrderAsc  = "asc"
	OrderDesc = "desc"
)

type Order string

func ToOrder(s string) Order {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "asc":
		return OrderAsc
	case "desc":
		return OrderDesc
	}
	return OrderNone
}

func callJuno(url string, flags share.FatimaCmdFlags, b []byte) (http.Header, map[string]interface{}, error) {
	headers, resp, err := share.CallFatimaApi(url, flags, b)
	if err != nil {
		return nil, nil, err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	return headers, respMap, nil
}
