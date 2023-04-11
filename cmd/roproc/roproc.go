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
	"throosea.com/fatima-cmd/juno"
	"throosea.com/fatima-cmd/share"
)

var usage = `usage: %s [option] command process [grp]

add/remove process in package

positional arguments:
  command               add or remove
  process               process name
  grp                   process group id

optional arguments:
  -d    Debug mode
  -p string
        Host and Package. e.g) localhost:default
`

const (
	// default process group if user not specify
	defaultGroupValue = "4"
)

func main() {
	flag.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}
	addCommand := flag.NewFlagSet("add", flag.ExitOnError)
	removeCommand := flag.NewFlagSet("remove", flag.ExitOnError)

	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		fmt.Printf("fail to parse : %s", err.Error())
		return
	}

	flag.Parse()

	if len(flag.Args()) < 2 {
		flag.Usage()
		return
	}

	switch flag.Args()[0] {
	case "add":
		addCommand.Parse(flag.Args()[1:])
	case "remove":
		removeCommand.Parse(flag.Args()[1:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if addCommand.Parsed() {
		if len(addCommand.Args()) < 1 {
			flag.Usage()
			return
		}
		err = share.GetJunoEndpoint(&fatimaFlags)
		if err != nil {
			fmt.Printf("endpoint retrieve fail : %s\n", err.Error())
			return
		}

		group := defaultGroupValue
		if len(addCommand.Args()) == 2 {
			group = addCommand.Args()[1]
		}
		err = juno.AddJunoProc(fatimaFlags, addCommand.Args()[0], group)
		if err != nil {
			fmt.Printf("fail to get juno package : %s\n", err.Error())
		}
		return
	} else if removeCommand.Parsed() {
		if len(removeCommand.Args()) < 1 {
			flag.Usage()
			return
		}
		err = share.GetJunoEndpoint(&fatimaFlags)
		if err != nil {
			fmt.Printf("endpoint retrieve fail : %s\n", err.Error())
			return
		}
		err = juno.RemoveJunoProc(fatimaFlags, removeCommand.Args()[0])
		if err != nil {
			fmt.Printf("fail to get juno package : %s\n", err.Error())
		}
		return
	}
}
