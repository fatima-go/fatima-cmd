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
	"os"
	"path/filepath"
	"strings"
)

type ExecuteUpdateBin struct {
}

func (i ExecuteUpdateBin) Name() string {
	return "update fatima tool binaries..."
}

func (i ExecuteUpdateBin) Execute(jobContext *UpdateContext) error {
	artifactBinDir := filepath.Join(jobContext.GetPackingDir(), "bin")
	currentBinDir := filepath.Join(jobContext.FatimaHomeDir, "bin")
	targetBinFiles := GetFilesInDir(artifactBinDir)
	if len(targetBinFiles) == 0 {
		return fmt.Errorf("not found target bin files")
	}

	for _, file := range targetBinFiles {
		src := filepath.Join(artifactBinDir, file)
		dst := filepath.Join(currentBinDir, file)
		err := CopyFile(src, dst)
		if err != nil {
			if !strings.Contains(err.Error(), "text file busy") {
				return fmt.Errorf("copyfile fail : %s", err.Error())
			}
			CopyToTemp(src, dst)
		}
		fmt.Printf("%s ", file)
	}
	fmt.Printf("\n")

	return nil
}

func CopyToTemp(src, dst string) {
	dst = filepath.Join(os.TempDir(), filepath.Base(dst))
	err := CopyFile(src, dst)
	if err != nil {
		fmt.Printf("\ncopy to temp [%s] : %s\n", dst, err.Error())
	}
}
