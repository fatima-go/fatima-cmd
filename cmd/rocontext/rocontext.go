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
	"bufio"
	"flag"
	"fmt"
	"github.com/fatima-go/fatima-cmd/cipher"
	"github.com/fatima-go/fatima-cmd/config"
	"golang.org/x/term"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

/*
rocontext help
rocontext add
rocontext remove dev
rocontext use dev
rocontext set dev
*/

var usage = `usage: %s [options] command context_name

Basic Commands :
 [options] add context_name		add new jupiter context
 remove context_name			remove jupiter context
 use context_name			use jupiter context
 set context_name           set jupiter context (user,passwd,timezone)
 setall           set jupiter to all context (user,passwd,timezone)

options:
 -h 			print usage
 -l uri			jupiter uri. e.g) http://localhost:9190
 -u username	jupiter user name. e.g) admin
 -p password	jupiter user password. e.g) admin
 -t timezone	local timezone. e.g) Asia/Seoul

example:
 $ rocontext -l http://localhost:9190 add local
 $ rocontext remove dev
 $ rocontext use prod
`

var (
	jupiterUri   = flag.String("l", "", "jupiter uri")
	userName     = flag.String("u", "admin", "user name")
	userPassword = flag.String("p", "admin", "user password")
	timezone     = flag.String("t", "Asia/Seoul", "timezone")
)

var jupiterConfig config.JupiterConfig

func main() {
	ProgramName := filepath.Base(os.Args[0])

	flag.Usage = func() {
		fmt.Printf(usage, ProgramName)
	}

	var err error
	jupiterConfig, err = config.NewJupiterConfigList()
	if err != nil {
		fmt.Printf("fatima jupiter context loading error : %s\n", err.Error())
		return
	}

	flag.Parse()
	command := strings.ToLower(flag.Arg(0))
	contextName := ""

	if len(flag.Args()) == 0 ||
		(command != "setall" && len(flag.Args()) < 2) {
		fmt.Printf("More usage : %s -h\n\n", ProgramName)
		fmt.Printf("%s\n", jupiterConfig)
		return
	}

	if len(flag.Args()) == 2 {
		contextName = flag.Arg(1)
	}

	switch command {
	case "add":
		doAddContext(contextName)
	case "remove":
		doRemoveContext(contextName)
	case "use":
		doUseContext(contextName)
	case "set":
		doSetContext(contextName)
	case "setall":
		doSetAllContext()
	default:
		flag.Usage()
		return
	}
}

func doSetAllContext() {
	newRecord, err := interactSetStage("")
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	fmt.Printf("\n----------------------------------------\n")
	fmt.Printf("username : %s\npassword : ......\ntimezone : %s\n\n", newRecord.User, newRecord.Timezone)

	for _, contextName := range jupiterConfig.GetAllContextNames() {
		err = jupiterConfig.SetContext(contextName, newRecord)
		if err != nil {
			fmt.Printf("fail to set %s context info : %s", contextName, err.Error())
			return
		}

		fmt.Printf("context %s set successfully\n", contextName)
	}
}

func buildDefaultJupiterContextRecord(name string) config.JupiterContextRecord {
	defaultRecord := config.JupiterContextRecord{}
	defaultRecord.User = ""
	defaultRecord.Password = ""
	defaultRecord.Timezone = ""
	if len(name) == 0 {
		allNames := jupiterConfig.GetAllContextNames()
		if len(allNames) == 0 {
			return defaultRecord
		}
		name = allNames[0]
	}

	ctx, err := jupiterConfig.GetContext(name)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return defaultRecord
	}

	defaultRecord.User = ctx.Context.User
	defaultRecord.Password = ""
	defaultRecord.Timezone = ctx.Context.Timezone
	return defaultRecord
}

func interactSetStage(name string) (config.JupiterContextRecord, error) {
	newRecord := config.JupiterContextRecord{}
	defaultRecord := buildDefaultJupiterContextRecord(name)

	// get username
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter Username (default %s): ", defaultRecord.User)
	newRecord.User, _ = reader.ReadString('\n')
	newRecord.User = strings.TrimSpace(newRecord.User)
	if len(newRecord.User) == 0 {
		newRecord.User = defaultRecord.User
	}

	// get password
	fmt.Print("Enter Password: ")
	bytePassword, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return newRecord, fmt.Errorf("error : %s\n", err.Error())
	}

	if len(bytePassword) == 0 {
		return newRecord, fmt.Errorf("\nempty password!!!\n")
	}

	newRecord.Password, err = cipher.Aes256Encode(string(bytePassword))
	if err != nil {
		return newRecord, fmt.Errorf("password encoding error : %s\n", err.Error())
	}

	// get timezone
	fmt.Printf("\nEnter Timezone (default : %s): ", defaultRecord.Timezone)
	newRecord.Timezone, _ = reader.ReadString('\n')
	newRecord.Timezone = strings.TrimSpace(newRecord.Timezone)
	if len(newRecord.Timezone) == 0 {
		newRecord.Timezone = defaultRecord.Timezone
	}

	return newRecord, nil
}

func doSetContext(name string) {
	newRecord, err := interactSetStage(name)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	err = jupiterConfig.SetContext(name, newRecord)
	if err != nil {
		fmt.Printf("fail to set %s context info : %s", name, err.Error())
		return
	}

	fmt.Printf("\n----------------------------------------\n")
	fmt.Printf("username : %s\npassword : ......\ntimezone : %s\n", newRecord.User, newRecord.Timezone)
	fmt.Printf("context %s set successfully\n", name)
}

func doAddContext(name string) {
	if jupiterUri == nil || len(*jupiterUri) == 0 {
		fmt.Printf("need jupiter uri\n")
		flag.Usage()
		return
	}
	// NewJupiterContext(name, jupiter, user, password, timezone string
	newContext := config.NewJupiterContext(name, *jupiterUri, *userName, *userPassword, *timezone)
	err := jupiterConfig.AddContext(newContext)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	fmt.Printf("new context %s added\n", name)
}

func doRemoveContext(name string) {
	err := jupiterConfig.RemoveContext(name)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	fmt.Printf("context %s removed\n", name)
}

func doUseContext(name string) {
	err := jupiterConfig.SetActive(name)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	fmt.Printf("context %s actived\n", name)
}
