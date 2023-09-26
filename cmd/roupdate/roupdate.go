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
	"os"
)

var usage = `usage: %s [option] command

golang fatima package update tool

version : 2023-09-19.v1

command :
  all    update tool binaries and opm processes
  bin    update only tool binaries
  opm    update only opm processes

optional arguments:
  -u string
        fatima packaging file url
`

var artifactUrl string

func main() {
	flag.Usage = func() {
		fmt.Printf(usage, os.Args[0])
	}

	flag.StringVar(&artifactUrl, "u", "", "fatima packaging file url")

	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
		return
	}

	command := flag.Args()[0]

	ctx, err := NewUpdateContext(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "packaging error : %s", err.Error())
		return
	}

	defer ctx.Close()

	for _, executor := range ctx.executorList {
		fmt.Printf("\n>>> %s...\n", executor.Name())
		err = executor.Execute(ctx)
		if err != nil {
			fmt.Printf("[%s] %s\n", executor.Name(), err.Error())
			return
		}
	}

}
