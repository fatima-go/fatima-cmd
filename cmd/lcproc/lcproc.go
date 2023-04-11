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
// @date 2018. 8. 7. PM 4:05
//

package main

import (
	"fmt"
	"os"
	"strings"
)

var roPrograms = []string{"jupiter", "juno", "saturn"}

var usage = `usage: %s process command [parameter]

display/control process version, duplicate process

positional arguments:
  process		process name
  command		version/dup

example :

lcproc mypgm version		: display mypgm revision versions
lcproc mypgm version R017	: change mypgm revision to R017
lcproc mypgm dup mypgm2		: duplicate mypgm to mypgm2
`

var proc string
var cmd string
var mode string
var turnOn bool

func main() {
	if len(os.Args) < 3 {
		fmt.Printf(string(usage), os.Args[0])
		return
	}

	proc = strings.ToLower(os.Args[1])
	if isRoProgram(proc) {
		fmt.Printf("not permitted ro programs (e.g juno,jupiter,saturn)\n")
		return
	}

	cmd = strings.ToLower(strings.ToLower(os.Args[2]))
	if cmd == "version" {
		versioning()
	} else if cmd == "dup" {
		if len(os.Args) < 4 {
			fmt.Printf(string(usage), os.Args[0])
		} else {
			duplicate()
		}
	} else {
		fmt.Printf(string(usage), os.Args[0])
	}
}
