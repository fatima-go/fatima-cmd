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
	"github.com/fatima-go/fatima-cmd/juno"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
)

var usage = `usage: %s [option] process

show deployment history of process

positional arguments:
  process               process name

optional arguments:
  -d    Debug mode
  -p string
        Host and Package. e.g) localhost:default
`

var optionGroup string
var optionAll bool

func main() {
	flag.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}

	flag.StringVar(&optionGroup, "g", "", "process group name")
	flag.BoolVar(&optionAll, "a", false, "all process")

	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		fmt.Printf("fail to build argument for execution : %s", err.Error())
		return
	}

	if !optionAll && len(optionGroup) == 0 && len(flag.Args()) < 1 {
		flag.Usage()
		return
	}

	err = share.GetJunoEndpoint(&fatimaFlags)
	if err != nil {
		fmt.Printf("endpoint retrieve fail : %s\n", err.Error())
		return
	}

	procName := ""
	if len(flag.Args()) > 0 {
		procName = flag.Args()[0]
	}

	err = juno.DeploymentHistoryJunoProc(fatimaFlags, optionGroup, optionAll, procName)
	if err != nil {
		fmt.Printf("fail to get juno package : %s\n", err.Error())
		return
	}
}
