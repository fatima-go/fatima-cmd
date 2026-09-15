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
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
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
	args := os.Args[1:]
	plain := false
	if len(args) > 0 && args[0] == "--plain" {
		plain = true
		args = args[1:]
		os.Args = append([]string{os.Args[0]}, args...)
	}
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Printf(usage, os.Args[0])
		fmt.Println("\n인자 없이 실행하면 TUI로 안내합니다. --plain: 기존 텍스트 방식")
		return
	}
	if !plain && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		if _, err := tea.NewProgram(newLocalModel(args), tea.WithAltScreen()).Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	legacyMain()
}

func legacyMain() {
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
