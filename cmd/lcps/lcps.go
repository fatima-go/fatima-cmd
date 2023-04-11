//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with p work for additional information
// regarding copyright ownership.  The ASF licenses p file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use p file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// @project fatima-cmd
// @author DeockJin Chung (jin.freestyle@gmail.com)
// @date 2017. 10. 28. PM 2:52
//

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"throosea.com/fatima-cmd/share"
)

var usage = `usage: %s [-h] [set] [status]

show/control package primary/secondary status

positional arguments:
  set         set package PS status
  status      primary/secondary

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
		case "PRIMARY":
			changePackageStatus(1)
			fmt.Printf("set to PRIMARY\n")
		case "SECONDARY":
			changePackageStatus(2)
			fmt.Printf("set to SECONDARY\n")
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
	hafile := getPackagePsFile()
	if !share.IsFileExist(hafile) {
		fmt.Printf("ps file not found\n")
		return
	}

	b, err := os.ReadFile(hafile)
	if err != nil {
		fmt.Printf("fail to read ps file : %s\n", err.Error())
		return
	}

	ha := strings.Trim(string(b), "\r\n\t ")
	switch ha {
	case "1":
		fmt.Printf("PRIMARY\n")
	case "2":
		fmt.Printf("SECONDARY\n")
	default:
		fmt.Printf("UNKNOWN\n")
	}
}

func getPackagePsFile() string {
	return filepath.Join(os.Getenv(share.EnvFatimaHome), systemPsFile)
}

const (
	systemPsFile = "/package/cfm/ha/system.ps"
)

func changePackageStatus(status int) {
	hafile := getPackagePsFile()
	if !share.IsFileExist(hafile) {
		fmt.Printf("ps file not found\n")
		return
	}

	d1 := []byte(fmt.Sprintf("%d", status))
	err := os.WriteFile(hafile, d1, 0644)
	if err != nil {
		fmt.Printf("fail to write ps file : %s", err.Error())
		os.Exit(1)
	}
}
