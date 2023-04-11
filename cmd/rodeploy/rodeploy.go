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
	"throosea.com/fatima-cmd/jupiter"
	"throosea.com/fatima-cmd/share"
)

var usage = `usage: %s [option] file

deploy package to server

positional arguments:
  file                  upload 'far' fatima package file

optional arguments:
  -d    Debug mode
  -g string
        package group name
  -p string
        Host and Package. e.g) localhost:default
`

func main() {
	flag.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}

	var group string

	flag.StringVar(&group, "g", "", "package group name")

	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		fmt.Printf("fail to build argument for execution : %s", err.Error())
		return
	}

	if len(flag.Args()) < 1 {
		flag.Usage()
		return
	}

	file := flag.Args()[0]
	if !share.IsFileExist(file) {
		fmt.Printf("far file doesn't exist : %s\n", file)
		return
	}

	err = share.GetToken(&fatimaFlags)
	if err != nil {
		fmt.Printf("auth fail : %s\n", err.Error())
		return
	}

	err = jupiter.DeployPackages(fatimaFlags, group, file)
	if err != nil {
		fmt.Printf("fail to deploy package : %s\n", err.Error())
		return
	}

}
