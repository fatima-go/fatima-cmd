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

package config

import (
	"bytes"
	"fmt"
	"github.com/fatima-go/fatima-cmd/cipher"
	"gopkg.in/yaml.v2"
	"os"
	"os/user"
	"path/filepath"
)

const (
	FatimaJupiterFolderName     = ".fatima"
	FatimaJupiterConfigfileName = "config"
)

func NewJupiterConfigList() (JupiterConfig, error) {
	var jupiterConfig JupiterConfig

	user, err := user.Current()
	if err != nil {
		return jupiterConfig, fmt.Errorf("cannot find os current user : %s", err.Error())
	}

	fatimaConfigDir := filepath.Join(user.HomeDir, FatimaJupiterFolderName)
	err = ensureDirectory(fatimaConfigDir, true)
	if err != nil {
		return jupiterConfig, fmt.Errorf("config directory error : %s", err.Error())
	}

	fatimaConfigFile := filepath.Join(fatimaConfigDir, FatimaJupiterConfigfileName)
	err = checkFileExist(fatimaConfigFile)
	if err != nil {
		// create sample
		createLocalConfigFile(fatimaConfigFile)
		//return jupiterConfig, fmt.Errorf("config file error : %s", err.Error())
	}

	d, err := os.ReadFile(fatimaConfigFile)
	if err != nil {
		return jupiterConfig, fmt.Errorf("reading config error : %s", err.Error())
	}

	err = yaml.Unmarshal(d, &jupiterConfig)
	if err != nil {
		return jupiterConfig, fmt.Errorf("reading config as yaml error : %s", err.Error())
	}

	return jupiterConfig, nil
}

func GetActiveContext() (JupiterContextRecord, error) {
	currentConfig, err := NewJupiterConfigList()
	if err != nil {
		return JupiterContextRecord{}, err
	}

	for _, v := range currentConfig {
		if v.Active {
			return v.Context, nil
		}
	}

	return JupiterContextRecord{}, fmt.Errorf("not found active context")
}

// createLocalConfigFile
func createLocalConfigFile(configFilePath string) error {
	var jupiterConfig JupiterConfig
	jupiterConfig = make([]JupiterContext, 1)
	jupiterConfig[0] = JupiterContext{}
	jupiterConfig[0].Name = localContextName
	jupiterConfig[0].Active = true
	jupiterConfig[0].Context.Jupiter = localJupiterUri
	jupiterConfig[0].Context.User = localUser
	jupiterConfig[0].Context.Password = localPassword
	jupiterConfig[0].Context.Timezone = getDefaultTimezone()

	d, err := yaml.Marshal(jupiterConfig)
	if err != nil {
		return fmt.Errorf("fail to create yaml data : %s", err.Error())
	}

	err = os.WriteFile(configFilePath, d, 0644)
	if err != nil {
		return fmt.Errorf("fail to save yaml configuration file : %s", err.Error())
	}

	return nil
}

const (
	localContextName = "local"
	localJupiterUri  = "http://127.0.0.1:9190"
	localUser        = "admin"
	localPassword    = "oBE87gjbotORkFKy+qEjbQ=="
	localTimezone    = "Asia/Seoul"
)

// getDefaultTimezone
// TODO : load local timezone
func getDefaultTimezone() string {
	return localTimezone
}

func syncJupiterConfigList(configContext JupiterConfig) error {
	user, err := user.Current()
	if err != nil {
		return fmt.Errorf("cannot find os current user : %s", err.Error())
	}

	fatimaConfigDir := filepath.Join(user.HomeDir, FatimaJupiterFolderName)
	err = ensureDirectory(fatimaConfigDir, true)
	if err != nil {
		return fmt.Errorf("config directory error : %s", err.Error())
	}

	fatimaConfigFile := filepath.Join(fatimaConfigDir, FatimaJupiterConfigfileName)
	d, err := yaml.Marshal(configContext)
	if err != nil {
		return fmt.Errorf("fail to marshal to yaml : %s", err.Error())
	}

	err = os.WriteFile(fatimaConfigFile, d, 0644)
	if err != nil {
		return fmt.Errorf("fail to save yaml configuration file : %s", err.Error())
	}

	return nil
}

type JupiterConfig []JupiterContext

func (j JupiterConfig) String() string {
	var buff bytes.Buffer
	header := fmt.Sprintf("%-8s%-16s%-48s%-12s%-12s\n", "CURRENT", "CONTEXT_NAME", "JUPITER", "USER", "TZ")
	buff.WriteString(header)
	for _, v := range j {
		context := fmt.Sprintf("%-8s%-16s%-48s%-12s%-12s\n", "",
			v.Name, v.Context.Jupiter, v.Context.User, v.Context.Timezone)
		if v.Active {
			context = fmt.Sprintf("%-8s%-16s%-48s%-12s%-12s\n", "*",
				v.Name, v.Context.Jupiter, v.Context.User, v.Context.Timezone)
		}
		buff.WriteString(context)
	}
	return buff.String()
}

func (j JupiterConfig) SetActive(name string) error {
	found := false
	for i := 0; i < len(j); i++ {
		if j[i].Name == name {
			j[i].Active = true
			found = true
			continue
		}
		j[i].Active = false
	}

	if !found {
		return fmt.Errorf("not found %s context", name)
	}

	return syncJupiterConfigList(j)
}

func (j JupiterConfig) SetContext(name string, ctx JupiterContextRecord) error {
	for i := 0; i < len(j); i++ {
		if j[i].Name == name {
			j[i].Context.User = ctx.User
			j[i].Context.Password = ctx.Password
			j[i].Context.Timezone = ctx.Timezone
			return syncJupiterConfigList(j)
		}
	}

	return fmt.Errorf("not found jupiter context for name %s", name)
}

func (j JupiterConfig) GetContext(name string) (JupiterContext, error) {
	for i := 0; i < len(j); i++ {
		if j[i].Name == name {
			return j[i], nil
		}
		j[i].Active = false
	}

	return JupiterContext{}, fmt.Errorf("not found jupiter context for name %s", name)
}

func (j JupiterConfig) GetAllContextNames() []string {
	contextNameList := make([]string, 0)

	for i := 0; i < len(j); i++ {
		contextNameList = append(contextNameList, j[i].Name)
	}

	return contextNameList
}

func (j JupiterConfig) AddContext(configContext JupiterContext) error {
	for _, v := range j {
		if v.Name == configContext.Name {
			return fmt.Errorf("%s context exist", configContext.Name)
		}
	}

	newList := make([]JupiterContext, 0)
	newList = append(newList, j...)
	newList = append(newList, configContext)

	if len(newList) == 1 {
		newList[0].Active = true
	}

	return syncJupiterConfigList(newList)
}

func (j JupiterConfig) RemoveContext(name string) error {
	activeRemoved := false
	newList := make([]JupiterContext, 0)
	for i := 0; i < len(j); i++ {
		if j[i].Name == name {
			if j[i].Active {
				activeRemoved = true
			}
			continue
		}
		newList = append(newList, j[i])
	}

	if activeRemoved && len(newList) > 0 {
		newList[0].Active = true
	}

	return syncJupiterConfigList(newList)
}

func NewJupiterContext(name, jupiter, user, password, timezone string) JupiterContext {
	c := JupiterContext{}
	c.Name = name
	c.Active = false
	c.Context.Jupiter = RemoveLastSlash(jupiter)
	c.Context.User = user
	c.Context.Password, _ = cipher.Aes256Encode(password)
	c.Context.Timezone = timezone
	return c
}

type JupiterContext struct {
	Name    string               `yaml:"name"`
	Active  bool                 `yaml:"active"`
	Context JupiterContextRecord `yaml:"context"`
}

type JupiterContextRecord struct {
	Jupiter  string `yaml:"jupiter"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Timezone string `yaml:"timezone"`
}

func (r JupiterContextRecord) GetPassword() string {
	plainText, _ := cipher.Aes256Decode(r.Password)
	return plainText
}
