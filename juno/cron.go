//
// Copyright (c) 2018 SK TECHX.
// All right reserved.
//
// This software is the confidential and proprietary information of SK TECHX.
// You shall not disclose such Confidential Information and
// shall use it only in accordance with the terms of the license agreement
// you entered into with SK TECHX.
//
//
// @project fatima-cmd
// @author 1100282
// @date 2018. 5. 5. PM 6:12
//

package juno

import (
	"encoding/json"
	"fmt"
	"strings"
	"throosea.com/fatima-cmd/share"
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
