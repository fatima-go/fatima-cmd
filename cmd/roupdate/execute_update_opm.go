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
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ExecuteUpdateOpm struct {
}

func (i ExecuteUpdateOpm) Name() string {
	return "update opm processes"
}

func (i ExecuteUpdateOpm) Execute(jobContext *UpdateContext) error {
	targetOpmBinFiles := i.GetTargetBin(jobContext)
	if len(targetOpmBinFiles) == 0 {
		return fmt.Errorf("not found target opm process")
	}

	err := i.stopOpm(jobContext)
	if err != nil {
		return fmt.Errorf("fail to stop opm : %s", err.Error())
	}

	i.WaitUntilOpmDown(jobContext)

	// copy
	for _, file := range targetOpmBinFiles {
		fmt.Printf("opm : %s\n", file)
		artifactBinDir := filepath.Join(jobContext.GetPackingDir(), "app", file)
		currentBinDir := filepath.Join(jobContext.FatimaHomeDir, "app", file)

		src := filepath.Join(artifactBinDir, file)
		dst := filepath.Join(currentBinDir, file)
		err := CopyFile(src, dst)
		if err != nil {
			return fmt.Errorf("copyfile fail : %s", err.Error())
		}
	}
	fmt.Printf("\n")

	return i.startOpm(jobContext)
}

func (i ExecuteUpdateOpm) WaitUntilOpmDown(jobContext *UpdateContext) error {
	fmt.Printf("wait until opm process down...\n")
	time.Sleep(time.Second * 3)
	return nil
}

func (i ExecuteUpdateOpm) stopOpm(jobContext *UpdateContext) error {
	fmt.Printf("- stop opm process\n")

	// lcslack false
	command := fmt.Sprintf("lcslack false")
	err := ExecuteShell(jobContext.WorkingDir, command)
	if err != nil {
		return err
	}

	// sleep 1
	time.Sleep(time.Second)

	// stopro -y
	command = fmt.Sprintf("stopro -y")
	err = ExecuteShell(jobContext.WorkingDir, command)
	if err != nil {
		return err
	}

	return nil
}

func (i ExecuteUpdateOpm) startOpm(jobContext *UpdateContext) error {
	fmt.Printf("- start opm process\n")

	command := fmt.Sprintf("startro -y")
	err := ExecuteShell(jobContext.WorkingDir, command)
	if err != nil {
		return err
	}

	// sleep 1
	time.Sleep(time.Second)

	// lcslack true
	command = fmt.Sprintf("lcslack true")
	err = ExecuteShell(jobContext.WorkingDir, command)
	if err != nil {
		return err
	}

	return nil
}

func (i ExecuteUpdateOpm) GetTargetBin(jobContext *UpdateContext) []string {
	targetBinList := make([]string, 0)
	procConfigYaml := filepath.Join(jobContext.FatimaHomeDir, "conf", FatimaFileProcConfig)
	data, err := os.ReadFile(procConfigYaml)
	if err != nil {
		fmt.Printf("fail to open %s : %s", procConfigYaml, err.Error())
		return targetBinList
	}

	procConfig := YamlFatimaPackageConfig{}
	err = yaml.Unmarshal(data, &procConfig)
	if err != nil {
		fmt.Printf("fail to yaml unmarshal %s : %s", procConfigYaml, err.Error())
		return targetBinList
	}

	opmGid := procConfig.GetOpmGid()
	if opmGid < 0 {
		fmt.Printf("not found opm gid %s : %s", procConfigYaml, err.Error())
		return targetBinList
	}

	return procConfig.GetProcessList(opmGid)
}

const (
	FatimaFileProcConfig = "fatima-package.yaml"
)

type YamlFatimaPackageConfig struct {
	Groups    []GroupItem   `yaml:"group,flow"`
	Processes []ProcessItem `yaml:"process"`
}

func (y YamlFatimaPackageConfig) GetOpmGid() int {
	for _, item := range y.Groups {
		if strings.ToUpper(item.Name) == "OPM" {
			return item.Id
		}
	}

	return -1 // not found
}

var defaultOpmProcessSet = map[string]struct{}{"jupiter": {}, "juno": {}, "saturn": {}}

func (y YamlFatimaPackageConfig) GetProcessList(gid int) []string {
	targetList := make([]string, 0)

	for _, item := range y.Processes {
		if item.Gid == gid {
			_, ok := defaultOpmProcessSet[item.Name]
			if ok {
				targetList = append(targetList, item.Name)
			}
		}
	}

	return targetList
}

type GroupItem struct {
	Id   int    `yaml:"id"`
	Name string `yaml:"name"`
}

type ProcessItem struct {
	Gid       int    `yaml:"gid"`
	Name      string `yaml:"name"`
	Loglevel  string `yaml:"loglevel"`
	Hb        bool   `yaml:"hb,omitempty"`
	Path      string `yaml:"path,omitempty"`
	Grep      string `yaml:"grep,omitempty"`
	Startmode int    `yaml:"startmode,omitempty"`
}
