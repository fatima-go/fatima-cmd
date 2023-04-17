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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func ExecuteShell(wd, command string) error {
	if len(command) == 0 {
		return errors.New("empty command")
	}

	var cmd *exec.Cmd
	cmd = exec.Command("/bin/sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = wd
	return cmd.Run()
}

func CheckFileExist(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not exist file : %s", path)
		}
		return fmt.Errorf("error checking : %s (%s)", path, err.Error())
	}

	if stat.IsDir() {
		return fmt.Errorf("exist but it is directory")
	}

	return nil
}

func GetFilesInDir(path string) []string {
	targetList := make([]string, 0)
	files, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("fail to read dir %s : %s", path, err.Error())
		return targetList
	}

	for _, file := range files {
		if !file.IsDir() {
			targetList = append(targetList, file.Name())
		}
	}

	return targetList
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s error : %s", src, err.Error())
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s error : %s", dst, err.Error())
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("copy error : %s", err.Error())
	}
	err = out.Sync()
	if err != nil {
		return fmt.Errorf("sync error : %s", err.Error())
	}

	return nil
}
