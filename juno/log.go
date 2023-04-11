//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with p work for additional information
// regarding copyright ownership.  The ASF licenses p file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use p file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// @project fatima-cmd
// @author DeockJin Chung (jin.freestyle@gmail.com)
// @date 2017. 10. 28. PM 2:52
//

package juno

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"throosea.com/fatima-cmd/share"
)

func PrintJunoLogLevels(flags share.FatimaCmdFlags) error {
	serviceUrl := flags.BuildJunoServiceUrl(v1LoglevelDisUrl)

	headers, resp, err := callJuno(serviceUrl, flags, nil)
	if err != nil {
		return err
	}

	share.PrintPreface(headers, resp)
	h := []string{"name", "level"}

	summaryObj := resp["summary"]
	summary, ok := summaryObj.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response structure")
	}

	data := make([][]string, 0)
	for _, v := range buildLogLevelInfoList(summary["loglevels"]) {
		data = append(data, v.ToList())
	}
	share.PrintTable(h, data)

	return nil
}

const (
	v1LoglevelDisUrl    = "loglevel/dis/v1"
	v1LoglevelChangeUrl = "loglevel/chg/v1"
)

type LogLevel struct {
	Name  string
	Level string
}

func (p LogLevel) ToList() []string {
	list := make([]string, 0)
	list = append(list, p.Name)
	list = append(list, p.Level)
	return list
}

type ByLogName []LogLevel

func (a ByLogName) Len() int           { return len(a) }
func (a ByLogName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLogName) Less(i, j int) bool { return strings.Compare(a[i].Name, a[j].Name) < 0 }

func buildLogLevelInfo(m map[string]interface{}) LogLevel {
	p := LogLevel{Name: "-", Level: "-"}

	for k, v := range m {
		switch k {
		case "name":
			p.Name = share.AsString(v)
		case "level":
			p.Level = share.AsString(v)
		}
	}

	return p
}

func buildLogLevelInfoList(data interface{}) []LogLevel {
	list := make([]LogLevel, 0)

	if val, ok := data.([]interface{}); ok {
		for _, v := range val {
			if m, ok := v.(map[string]interface{}); ok {
				list = append(list, buildLogLevelInfo(m))
			}
		}
	}

	sort.Sort(ByLogName(list))
	return list
}

func ChangeLogLevel(flags share.FatimaCmdFlags) error {
	serviceUrl := flags.BuildJunoServiceUrl(v1LoglevelChangeUrl)

	m := make(map[string]interface{})
	m["process"] = flags.Args[0]
	m["loglevel"] = flags.Args[1]

	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("fail to marshal to json : %s\n", err.Error())
	}

	headers, resp, err := callJuno(serviceUrl, flags, b)
	if err != nil {
		return err
	}

	share.PrintPreface(headers, resp)

	message := share.GetSummaryMessage(resp)
	fmt.Printf("%s\n", message)

	return nil
}
