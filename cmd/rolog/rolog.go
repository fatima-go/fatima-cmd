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
	"fmt"
	"throosea.com/fatima-cmd/juno"
	"throosea.com/fatima-cmd/share"
)

func main() {
	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		fmt.Printf("fail to build argument for execution : %s", err.Error())
		return
	}

	if len(fatimaFlags.Args) != 0 && len(fatimaFlags.Args) != 2 {
		fmt.Printf("You must provide complete process name and log level\n")
		return
	}

	err = share.GetJunoEndpoint(&fatimaFlags)
	if err != nil {
		fmt.Printf("endpoint retrieve fail : %s\n", err.Error())
		return
	}

	if len(fatimaFlags.Args) == 0 {
		err = juno.PrintJunoLogLevels(fatimaFlags)
		if err != nil {
			fmt.Printf("fail to get juno package : %s\n", err.Error())
			return
		}
		return
	}

	err = juno.ChangeLogLevel(fatimaFlags)
	if err != nil {
		fmt.Printf("fail to change loglevel from juno : %s\n", err.Error())
		return
	}
}
