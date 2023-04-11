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
// @date 2018. 7. 31. PM 3:09
//

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"throosea.com/fatima-cmd/share"
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
			fmt.Printf(string(usage), os.Args[0])
			return
		}

		if setAllStatus(turnOn) {
			printStatus()
		}
		return
	}

	if len(os.Args) != 3 {
		fmt.Printf(string(usage), os.Args[0])
		return
	}

	mode = strings.ToLower(os.Args[1])
	onoff := strings.ToLower(os.Args[2])
	if onoff == "true" {
		turnOn = true
	} else if onoff == "false" {
		turnOn = false
	} else {
		fmt.Printf(string(usage), os.Args[0])
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
	Active bool   `json:"active"`
	Url    string `json:"url"`
}

type slackConfig map[string]*SlackConfig

func printStatus() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("fail to load slack webhook file : %s\n", err.Error())
		return
	}

	alarm, ok := config["alarm"]
	if !ok {
		fmt.Printf("not found alarm config\n")
	} else {
		fmt.Printf("alarm : %v [%s]\n", alarm.Active, alarm.Url)
	}
	event, ok := config["event"]
	if !ok {
		fmt.Printf("not found event config\n")
	} else {
		fmt.Printf("event : %v [%s]\n", event.Active, event.Url)
	}

	for k, v := range config {
		if k == "alarm" || k == "event" {
			continue
		}
		fmt.Printf("%s : %v [%s]\n", k, v.Active, v.Url)
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
