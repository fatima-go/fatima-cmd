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
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"throosea.com/fatima-cmd/share"
)

var usage = `usage: %s [-h] 

clear fatima app directories

positional arguments:

optional arguments:
  -h, --help  show this help message and exit
`

func main() {
	flag.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}
	setCommand := flag.NewFlagSet("set", flag.ExitOnError)
	setCommand.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}

	flag.Parse()
	clearAppLink()
	clearBackupFiles()
}

func clearBackupFiles() {
	appDir := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp)

	files := make([]string, 0)
	filepath.Walk(appDir, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".backup") {
			files = append(files, path)
		}
		if strings.HasSuffix(path, ".old") {
			files = append(files, path)
		}
		return nil
	})

	for _, file := range files {
		os.Remove(file)
		fmt.Printf("removed : %s\n", file)
	}
}

func clearAppLink() {
	appDir := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp)
	//revisionDir := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderRevision)

	files, err := ioutil.ReadDir(appDir)
	if err != nil {
		fmt.Printf("fail to read dir %s : %s\n", appDir, err.Error())
		return
	}

	for _, f := range files {
		checkLink(filepath.Join(appDir, f.Name()))
	}
}

func checkLink(path string) {
	info, err := os.Lstat(path)
	if err != nil {
		fmt.Printf("fail Lstat (%s) : %s", path, err.Error())
		return
	}
	switch mode := info.Mode(); {
	case mode&os.ModeSymlink != 0:
		inspectAppVersion(path)
	}
}

func inspectAppVersion(path string) {
	eval, err := filepath.EvalSymlinks(path)
	if err != nil {
		fmt.Printf("fail to read linke (%s) : %s\n", path, err.Error())
		return
	}
	//fmt.Printf("eval(%s) : %s\n", path, eval)

	appRevBaseDir := filepath.Dir(eval)
	files, err := ioutil.ReadDir(appRevBaseDir)
	if err != nil {
		fmt.Printf("fail to read dir %s : %s\n", appRevBaseDir, err.Error())
		return
	}

	originName := filepath.Base(eval)
	for _, f := range files {
		if f.Name() == originName {
			continue
		}
		removingDir := filepath.Join(appRevBaseDir, f.Name())
		removeDir(removingDir)
	}
}

func removeDir(path string) {
	command := fmt.Sprintf("rm -rf %s", path)
	_, err := ExecuteShell(command)
	if err != nil {
		fmt.Printf("[%s] fail to remove dir : %s", path, err.Error())
		return
	}
	fmt.Printf("removed dir : %s\n", path)
}

func ExecuteShell(command string) (string, error) {
	if len(command) == 0 {
		return "", errors.New("empty command")
	}

	var cmd *exec.Cmd
	cmd = exec.Command("/bin/sh", "-c", command)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
