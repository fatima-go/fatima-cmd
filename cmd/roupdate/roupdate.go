/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with p work for additional information
 * regarding copyright ownership.  The ASF licenses p file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use p file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 * @project fatima
 * @author DeockJin Chung (jin.freestyle@gmail.com)
 * @date 22. 10. 7. 오후 6:02
 */

package main

import (
	"flag"
	"fmt"
	"os"
)

var usage = `usage: %s command

golang fatima package update tool

command :
  all    update tool binaries and opm processes
  bin    update only tool binaries
  opm    update only opm processes
`

func main() {
	flag.Usage = func() {
		fmt.Printf(usage, os.Args[0])
	}

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
