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

var usage = `usage: %s [option]

display package report

optional arguments:
  -d    Debug mode
  -s    string
        sorting option. name=byNameAsc, index=byRegisterIndex. default name
  -p    string
        Host and Package. e.g) localhost:default
`

func main() {
	flag.Usage = func() {
		fmt.Printf(usage, os.Args[0])
	}

	var sort string

	flag.StringVar(&sort, "s", "", "sorting option")

	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		fmt.Printf("fail to build argument for execution : %s", err.Error())
		return
	}

	err = share.GetJunoEndpoint(&fatimaFlags)
	if err != nil {
		fmt.Printf("endpoint retrieve fail : %s\n", err.Error())
		return
	}

	err = juno.PrintJunoPackage(fatimaFlags, juno.NewSortingOption(sort))
	if err != nil {
		fmt.Printf("fail to get juno package : %s\n", err.Error())
		return
	}
}
