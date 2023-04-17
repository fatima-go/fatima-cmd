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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var roPrograms = []string{"jupiter", "juno", "saturn"}

func main() {
	fmt.Printf("STARTING OPM PROGRAMS...\n")

	if len(os.Getenv(share.EnvFatimaHome)) == 0 {
		fmt.Printf("env %s missing\n", share.EnvFatimaHome)
		return
	}

	flags := buildCoommandmdFlags()
	if !flags.Yes {
		reader := bufio.NewReader(os.Stdin)
		for true {
			fmt.Printf("start all programs? (y/n) ")
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
		err := startProgram(p)
		if err != nil {
			fmt.Printf("fail to execute %s : %s\n", p, err.Error())
		}
	}
}

func startProgram(procName string) error {
	fmt.Printf("check process %s\n", procName)

	pgm := buildShellPath(procName)
	if !share.IsFileExist(pgm) {
		pgm = buildShellPath(procName + ".sh")
		if !share.IsFileExist(pgm) {
			return nil
		}
	}

	workingDir := getWorkingDir(procName)
	pid, err := execProgram(workingDir, pgm)
	if err != nil {
		return err
	}
	fmt.Printf("process %s STARTED. pid=%d\n", procName, pid)
	time.Sleep(time.Second * 1)
	return nil
}

func buildShellPath(procName string) string {
	return fmt.Sprintf("%s/app/%s/%s", os.Getenv(share.EnvFatimaHome), procName, procName)
}

func getWorkingDir(procName string) string {
	return fmt.Sprintf("%s/app/%s", os.Getenv(share.EnvFatimaHome), procName)
}

func execProgram(workingDir string, path string) (int, error) {
	var cmd *exec.Cmd
	//cmd = exec.Command(filepath.Base(path))
	cmd = exec.Command("bash", "-c", filepath.Base(path))
	cmd.Dir = workingDir

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		return 0, err
	}

	slurp, _ := ioutil.ReadAll(stderr)
	fmt.Printf("%s\n", slurp)

	return cmd.Process.Pid, nil
}

// cmd := exec.Command("bash", "-c", "pidof tor | xargs kill -HUP")

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
