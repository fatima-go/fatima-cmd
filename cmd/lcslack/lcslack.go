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
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
	"path/filepath"
	"strings"
)

var usage = `usage: %s [options]

display/control slack notification for package
examples)
lcslack 
lcslack alarm true
lcslack event false
lcslack true
lcslack false
`

var mode string
var turnOn bool

func main() {
	if len(os.Args) < 2 {
		printStatus()
		return
	}

	if len(os.Args) == 2 {
		mode = strings.ToLower(os.Args[1])

		if mode == "true" {
			turnOn = true
		} else if mode == "false" {
			turnOn = false
		} else {
			fmt.Printf(usage, os.Args[0])
			return
		}

		if setAllStatus(turnOn) {
			printStatus()
		}
		return
	}

	if len(os.Args) != 3 {
		fmt.Printf(usage, os.Args[0])
		return
	}

	mode = strings.ToLower(os.Args[1])
	onoff := strings.ToLower(os.Args[2])

	if onoff == "true" {
		turnOn = true
	} else if onoff == "false" {
		turnOn = false
	} else {
		fmt.Printf(usage, os.Args[0])
		return
	}

	if setStatus(mode, turnOn) {
		printStatus()
	}
}

const (
	saturnWebhookFile = "/data/saturn/webhook.slack"
)

func getSlackWebhookConfig() string {
	return filepath.Join(os.Getenv(share.EnvFatimaHome), saturnWebhookFile)
}

func loadConfig() (slackConfig, error) {
	var config map[string]*SlackConfig

	dataBytes, err := os.ReadFile(getSlackWebhookConfig())
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(dataBytes, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

type SlackConfig struct {
	Active  bool   `json:"active"`
	Url     string `json:"url"`
	Channel string `json:"channel,omitempty"`
}

func (sc SlackConfig) String() string {
	if len(sc.Channel) > 0 {
		return fmt.Sprintf("activate: [%t], hookUri: [%s], channel: [%s]", sc.Active, sc.Url, sc.Channel)
	}

	return fmt.Sprintf("activate: [%t], hookUri: [%s]", sc.Active, sc.Url)
}

type slackConfig map[string]*SlackConfig

func printStatus() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("fail to load slack webhook file : %s\n", err.Error())
		return
	}

	for key, value := range config {
		fmt.Printf("%s: %s\n", key, value)
	}

	if _, existsAlarm := config["alarm"]; !existsAlarm {
		fmt.Println("not found alarm config")
	}

	if _, existsEvent := config["event"]; !existsEvent {
		fmt.Println("not found event config")
	}
}

func setAllStatus(turnOn bool) bool {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("fail to load slack webhook file : %s\n", err.Error())
		return false
	}

	for _, v := range config {
		v.Active = turnOn
	}

	err = saveConfig(config)
	if err != nil {
		fmt.Printf("fail to save config : %s\n", err.Error())
		return false
	}

	fmt.Printf("successfully saved\n")
	return true
}

func setStatus(partName string, turnOn bool) bool {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("fail to load slack webhook file : %s\n", err.Error())
		return false
	}

	part, ok := config[partName]
	if !ok {
		fmt.Printf("not found %s part in config\n", partName)
		return false
	}

	part.Active = turnOn
	err = saveConfig(config)
	if err != nil {
		fmt.Printf("fail to save config : %s\n", err.Error())
		return false
	}

	fmt.Printf("set %s to %v\n", partName, turnOn)
	return true
}

func saveConfig(config slackConfig) error {
	bytes, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(getSlackWebhookConfig(), bytes, 0755)
	if err != nil {
		fmt.Printf("fail to save config : %s\n", err.Error())
		return err
	}

	return nil
}
