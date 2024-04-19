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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ExecuteUpdateBin struct {
}

func (i ExecuteUpdateBin) Name() string {
	return "update fatima tool binaries..."
}

const (
	roupdateBin = "roupdate"
)

func (i ExecuteUpdateBin) Execute(jobContext *UpdateContext) error {
	artifactBinDir := filepath.Join(jobContext.GetPackingDir(), "bin")
	currentBinDir := filepath.Join(jobContext.FatimaHomeDir, "bin")
	targetBinFiles := GetFilesInDir(artifactBinDir)
	if len(targetBinFiles) == 0 {
		return fmt.Errorf("not found target bin files")
	}

	tmpBinList := make([]string, 0)
	for _, file := range targetBinFiles {
		src := filepath.Join(artifactBinDir, file)
		dst := filepath.Join(currentBinDir, file)

		copiedPath, err := copyFatimaBinaries(src, dst)
		if err != nil {
			return fmt.Errorf("copyfile fail : %s", err.Error())
		}
		if copiedPath != dst {
			tmpBinList = append(tmpBinList, copiedPath)
		}

		f, _ := os.Stat(copiedPath)
		if !isExecOwner(f.Mode()) {
			mode := f.Mode() | 0700
			_ = os.Chmod(copiedPath, mode)
		}
		fmt.Printf("%s ", file)
	}
	fmt.Printf("\n")
	for _, tmpBin := range tmpBinList {
		fmt.Printf("\n>>> binary copied to %s. YOU HAVE TO MOVE IT\n", tmpBin)
		fmt.Printf("\n$ cp %s $FATIMA_HOME/bin\n", tmpBin)
	}

	return nil
}

// copyBinary 바이너리 파일을 복사한다.
// 정상적일 경우 파라미터로 요청한 dst 경로를 리턴한다
// 만약 에러가 있거나 에러가 없더라도 dst 와 다른 경로를 리턴할 수 있다 (이 경우 dst는 temp 에 복사된 경로일 것이다)
func copyFatimaBinaries(src, dst string) (copiedPath string, err error) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		copiedPath, err = copyBinaryFile(src, dst)
		cancel()
	}()

	select {
	case <-time.After(time.Second):
		copiedPath = CopyToTemp(src, dst)
	case <-ctx.Done():
	}

	return
}

func copyBinaryFile(src, dst string) (string, error) {
	err := CopyFile(src, dst)
	if err != nil {
		if !strings.Contains(err.Error(), "text file busy") {
			return "", fmt.Errorf("copyfile fail : %s", err.Error())
		}
		tmp := CopyToTemp(src, dst)
		return tmp, nil
	}
	return dst, nil
}

func isExecOwner(mode os.FileMode) bool {
	return mode&0100 != 0
}

func CopyToTemp(src, dst string) string {
	dst = filepath.Join(os.TempDir(), filepath.Base(dst))
	err := CopyFile(src, dst)
	if err != nil {
		fmt.Printf("\ncopy to temp [%s] : %s\n", dst, err.Error())
	}
	return dst
}
