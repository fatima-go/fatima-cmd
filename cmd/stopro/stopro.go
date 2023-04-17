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
	"bufio"
	"flag"
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var roPrograms = []string{"jupiter", "juno", "saturn"}

func main() {
	fmt.Printf("STOPPING OPM PROGRAMS...\n")

	if len(os.Getenv(share.EnvFatimaHome)) == 0 {
		fmt.Printf("env %s missing\n", share.EnvFatimaHome)
		return
	}

	flags := buildCoommandmdFlags()
	if !flags.Yes {
		reader := bufio.NewReader(os.Stdin)
		for true {
			fmt.Printf("stop all programs? (y/n) ")
			text, _ := reader.ReadString('\n')
			if len(text) == 0 {
				continue
			}
			answer := strings.ToLower(strings.Trim(text, "\r\n\t "))
			if answer == "n" {
				return
			} else if answer == "y" {
				break
			}
		}
	}

	for _, p := range roPrograms {
		stopProgram(p)
	}
}

func stopProgram(procName string) error {
	fmt.Printf("check process %s\n", procName)

	pid := getPidPath(procName)
	if pid == 0 {
		return nil
	}

	fmt.Printf("try to kill %s. pid %d\n", procName, pid)
	return syscall.Kill(pid, syscall.SIGTERM)
}

func getPidPath(procName string) int {
	pidFile := fmt.Sprintf("%s/app/%s/proc/%s.pid", os.Getenv(share.EnvFatimaHome), procName, procName)
	if !share.IsFileExist(pidFile) {
		return 0
	}

	b, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Printf("fail to read pid file : %s\n", err.Error())
		return 0
	}

	content := strings.Trim(string(b), "\r\n\t ")
	pid, err := strconv.Atoi(content)
	if err != nil {
		fmt.Printf("invalid pid content : [%s]\n", string(b))
		return 0
	}

	return pid
}

type commandFlags struct {
	Yes  bool
	Args []string
}

func buildCoommandmdFlags() commandFlags {
	cmdFlags := commandFlags{}

	flag.BoolVar(&cmdFlags.Yes, "y", false, "yes all")

	flag.Parse()

	cmdFlags.Args = flag.Args()
	return cmdFlags
}
