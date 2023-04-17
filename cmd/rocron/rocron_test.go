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

package main

import (
	"encoding/json"
	"github.com/fatima-go/fatima-cmd/juno"
	"testing"
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
