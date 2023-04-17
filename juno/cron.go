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
	"strings"
)

const (
	v1CronList  = "cron/list/v1"
	v1CronRerun = "cron/rerun/v1"
)

//	type FatimaCronCommands struct {
//		Commands []struct {
//			Jobs []struct {
//				Desc   string `json:"desc"`
//				Name   string `json:"name"`
//				Sample string `json:"sample"`
//			} `json:"jobs"`
//			Process string `json:"process"`
//		} `json:"commands"`
//	}
type FatimaCronCommands struct {
	Commands []CronCommand `json:"commands"`
}

type CronCommand struct {
	Jobs    []CronJob `json:"jobs"`
	Process string    `json:"process"`
}

type CronJob struct {
	Desc   string `json:"desc"`
	Name   string `json:"name"`
	Sample string `json:"sample"`
}

func ListCronCommands(flags share.FatimaCmdFlags) (FatimaCronCommands, error) {
	var cronCommands FatimaCronCommands

	url := flags.BuildJunoServiceUrl(v1CronList)

	headers, resp, err := callJuno(url, flags, nil)
	if err != nil {
		return cronCommands, err
	}

	share.PrintPreface(headers, resp)

	summaryObj := resp["summary"]
	summary, ok := summaryObj.(map[string]interface{})
	if !ok {
		return cronCommands, fmt.Errorf("invalid summary structure")
	}
	commandsObj, ok := summary["commands"]
	if !ok {
		return cronCommands, fmt.Errorf("invalid commands structure")
	}

	commands := make(map[string]interface{})
	commands["commands"] = commandsObj
	b, _ := json.Marshal(commands)
	err = json.Unmarshal(b, &cronCommands)
	if err != nil {
		return cronCommands, fmt.Errorf("invalid cron command sturcture : %s", err.Error())
	}

	return cronCommands, nil
}

func RerunCronCommands(flags share.FatimaCmdFlags, proc string, job string, args string) error {
	url := flags.BuildJunoServiceUrl(v1CronRerun)

	m := make(map[string]interface{})
	m["process"] = proc
	m["command"] = job
	m["sample"] = strings.TrimSpace(args)

	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("fail to marshal to json : %s\n", err.Error())
	}

	headers, resp, err := callJuno(url, flags, b)
	if err != nil {
		return err
	}

	share.PrintPreface(headers, resp)

	message := share.GetSummaryMessage(resp)
	fmt.Printf("%s\n", message)

	return nil
}
