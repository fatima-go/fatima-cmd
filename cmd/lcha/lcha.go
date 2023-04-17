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
	"flag"
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
	"path/filepath"
	"strings"
)

var usage = `usage: %s [-h] [set] [status]

show/control package status

positional arguments:
  set         set package status
  status      active/standby

optional arguments:
  -h, --help  show this help message and exit
`

func main() {
	flag.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}
	setCommand := flag.NewFlagSet("set", flag.ExitOnError)
	setCommand.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}

	flag.Parse()

	if len(os.Args) == 1 {
		printPackageHaStatus()
		return
	}

	if len(flag.Args()) < 2 {
		flag.Usage()
		return
	}

	switch os.Args[1] {
	case "set":
		setCommand.Parse(os.Args[2:])
		status := os.Args[2]
		switch strings.ToUpper(status) {
		case "ACTIVE":
			changePackageStatus(1)
			fmt.Printf("set to ACTIVE\n")
		case "STANDBY":
			changePackageStatus(2)
			fmt.Printf("set to STANDBY\n")
		default:
			flag.Usage()
			return
		}
	default:
		fmt.Printf(string(usage), os.Args[0])
		return
	}
}

func printPackageHaStatus() {
	hafile := getPackageHaFile()
	if !share.IsFileExist(hafile) {
		fmt.Printf("ha file not found\n")
		return
	}

	b, err := os.ReadFile(hafile)
	if err != nil {
		fmt.Printf("fail to read ha file : %s\n", err.Error())
		return
	}

	ha := strings.Trim(string(b), "\r\n\t ")
	switch ha {
	case "1":
		fmt.Printf("ACTIVE\n")
	case "2":
		fmt.Printf("STANDBY\n")
	default:
		fmt.Printf("UNKNOWN\n")
	}
}

func getPackageHaFile() string {
	return filepath.Join(os.Getenv(share.EnvFatimaHome), systemHaFile)
}

const (
	systemHaFile = "/package/cfm/ha/system.ha"
)

func changePackageStatus(status int) {
	hafile := getPackageHaFile()
	if !share.IsFileExist(hafile) {
		fmt.Printf("ha file not found\n")
		return
	}

	d1 := []byte(fmt.Sprintf("%d", status))
	err := os.WriteFile(hafile, d1, 0644)
	if err != nil {
		fmt.Printf("fail to write ha file : %s", err.Error())
		os.Exit(1)
	}
}
