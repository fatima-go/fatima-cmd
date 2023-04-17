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
	"strconv"
	"strings"
	"time"
)

func AddJunoProc(flags share.FatimaCmdFlags, procName string, groupId string) error {
	serviceUrl := flags.BuildJupiterServiceUrl(v1ProcRegistUrl)

	m := make(map[string]interface{})
	m["process"] = procName
	m["group_id"] = groupId

	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("fail to marshal to json : %s\n", err.Error())
	}

	headers, resp, err := share.CallFatimaApi(serviceUrl, flags, b)
	if err != nil {
		return err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	share.PrintPreface(headers, nil)

	message := share.GetSystemMessage(respMap)
	fmt.Printf("%s\n", message)

	return nil
}

const (
	v1ProcRegistUrl   = "proc/regist/v1"
	v1ProcUnregistUrl = "proc/unregist/v1"
	v1ProcStartUrl    = "process/start/v1"
	v1ProcStopUrl     = "process/stop/v1"
	v1ProcClricUrl    = "process/clric/v1"
	v1ProcHistoryUrl  = "process/history/v1"
)

func RemoveJunoProc(flags share.FatimaCmdFlags, procName string) error {
	serviceUrl := flags.BuildJupiterServiceUrl(v1ProcUnregistUrl)

	m := make(map[string]interface{})
	m["process"] = procName

	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("fail to marshal to json : %s\n", err.Error())
	}

	headers, resp, err := share.CallFatimaApi(serviceUrl, flags, b)
	if err != nil {
		return err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	share.PrintPreface(headers, nil)

	message := share.GetSystemMessage(respMap)
	fmt.Printf("%s\n", message)

	return nil
}

func StartJunoProc(flags share.FatimaCmdFlags, group string, all bool, procName string) error {
	url := flags.BuildJunoServiceUrl(v1ProcStartUrl)

	m := make(map[string]interface{})

	if all {
		m["all"] = ""
	} else if len(group) > 0 {
		m["group"] = group
	} else {
		m["process"] = procName
	}

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

func StopJunoProc(flags share.FatimaCmdFlags, group string, all bool, procName string) error {
	url := flags.BuildJunoServiceUrl(v1ProcStopUrl)

	m := make(map[string]interface{})

	if all {
		m["all"] = ""
	} else if len(group) > 0 {
		m["group"] = group
	} else {
		m["process"] = procName
	}

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

func ClearIcJunoProc(flags share.FatimaCmdFlags, group string, all bool, procName string) error {
	url := flags.BuildJunoServiceUrl(v1ProcClricUrl)

	m := make(map[string]interface{})

	if all {
		m["all"] = ""
	} else if len(group) > 0 {
		m["group"] = group
	} else {
		m["process"] = procName
	}

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

func DeploymentHistoryJunoProc(flags share.FatimaCmdFlags, group string, all bool, procName string) error {
	url := flags.BuildJunoServiceUrl(v1ProcHistoryUrl)

	m := make(map[string]interface{})

	if all {
		m["all"] = ""
	} else if len(group) > 0 {
		m["group"] = group
	} else {
		m["process"] = procName
	}

	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("fail to marshal to json : %s\n", err.Error())
	}

	headers, resp, err := callJuno(url, flags, b)
	if err != nil {
		return err
	}

	share.PrintPreface(headers, resp)

	history := share.GetSummaryHistory(resp)

	message := share.GetSummaryMessage(resp)
	fmt.Printf("\n%s\n", message)

	if len(history) > 0 {
		fmt.Printf("\n--------------------------------------------\n")
	}

	// print deployment history records
	for _, v := range history {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		// print deployment history record
		deploymentTime := share.GetKeyInMap(m, "deployment_time")
		fmt.Print("- deployment datetime : ")
		if len(deploymentTime) > 0 {
			i, err := strconv.Atoi(deploymentTime)
			if err == nil {
				dtime := time.UnixMilli(int64(i)).Local().Format(TimeYyyymmddhhmmss)
				fmt.Printf("%s", dtime)
			}
		}

		fmt.Printf("\n + build user : %s", share.GetKeyInMap(m, "build.user"))
		fmt.Printf("\n + build time : %s", share.GetKeyInMap(m, "build.time"))
		fmt.Printf("\n + git branch : %s", share.GetKeyInMap(m, "build.git.branch"))
		fmt.Printf("\n + commit hash : %s", share.GetKeyInMap(m, "build.git.commit"))
		commitMessage := share.GetKeyInMap(m, "build.git.message")
		fmt.Printf("\n + commit message : %s", strings.TrimSpace(commitMessage))
		fmt.Printf("\n--------------------------------------------\n")
	}

	fmt.Println()
	return nil
}

const (
	TimeYyyymmddhhmmss = "2006-01-02 15:04:05"
)
