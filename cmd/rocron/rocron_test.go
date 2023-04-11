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
// @date 2018. 5. 5. PM 7:33
//

package main

import (
	"encoding/json"
	"testing"
	"throosea.com/fatima-cmd/juno"
)

var sample = `{
	"commands": [
		{
			"jobs": [
				{
					"desc":"일별 음원 메타파일 동기화",
					"name":"dailymusicmeta",
					"sample" : " (e.g 20170701)"
				},
				{
					"desc":"시간별 음원 메타파일 동기화",
					"name":"hourlymusicmeta",
					"sample":"yyyyMMdd HH (e.g 20170701 13)"
				}
			],
			"process":"batmeta"
		},
		{
			"jobs": [
				{
					"desc":"일별 음원 메타파일 동기화2",
					"name":"dailymusicmeta2",
					"sample" : " (e.g 20170701)"
				},
				{
					"desc":"시간별 음원 메타파일 동기화2",
					"name":"hourlymusicmeta2",
					"sample":"yyyyMMdd HH (e.g 20170701 13)"
				}
			],
			"process":"batmeta2"
		}
	]
}`

var empty = `{
	"commands": [
	]
}`

func TestCron(t *testing.T) {
	var cronCommands juno.FatimaCronCommands
	err := json.Unmarshal([]byte(sample), &cronCommands)
	if err != nil {
		t.Fatalf("fail to unmarshal : %s", err.Error())
	}

	if len(cronCommands.Commands) != 2 {
		t.Fatalf("invalid command length should 2")
	}

	err = json.Unmarshal([]byte(empty), &cronCommands)
	if err != nil {
		t.Fatalf("fail to unmarshal : %s", err.Error())
	}

	if len(cronCommands.Commands) != 0 {
		t.Fatalf("invalid command length should 0")
	}
}
