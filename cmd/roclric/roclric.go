//
// Copyright (c) 2018 SK TECHX.
// All right reserved.
//
// This software is the confidential and proprietary information of SK TECHX.
// You shall not disclose such Confidential Information and
// shall use it only in accordance with the terms of the license agreement
// you entered into with SK TECHX.
//
//
// @project fatima-cmd
// @author 1100282
// @date 2018. 7. 31. PM 2:30
//

package main

import (
	"flag"
	"fmt"
	"os"
	"throosea.com/fatima-cmd/juno"
	"throosea.com/fatima-cmd/share"
)

var usage = `usage: %s [option] process

clear ic(initial count) process

positional arguments:
  process               process name

optional arguments:
  -a    clear ic all process
  -d    Debug mode
  -g string
        process group name. e.g) svc
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
	flag.BoolVar(&optionAll, "a", false, "clear ic all process")

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

	err = juno.ClearIcJunoProc(fatimaFlags, optionGroup, optionAll, procName)
	if err != nil {
		fmt.Printf("fail to get juno package : %s\n", err.Error())
		return
	}
}
