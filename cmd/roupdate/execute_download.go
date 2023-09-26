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

type ExecuteDownload struct {
}

func (i ExecuteDownload) Name() string {
	return "downloading artifact...."
}

func (i ExecuteDownload) Execute(jobContext *UpdateContext) error {
	artifactUrl := jobContext.GetDownloadUrl()
	command := fmt.Sprintf("wget %s", artifactUrl)
	err := ExecuteShell(jobContext.WorkingDir, command)
	if err != nil {
		return err
	}

	// check download file
	filename := filepath.Base(artifactUrl)
	downloadedArtifact := filepath.Join(jobContext.WorkingDir, filename)
	err = CheckFileExist(downloadedArtifact)
	if err != nil {
		return fmt.Errorf("artifact downloading fail : %s", artifactUrl)
	}

	// unzip
	command = fmt.Sprintf("gzip -cd %s | tar xvf -", filename)
	err = ExecuteShell(jobContext.WorkingDir, command)
	if err != nil {
		return err
	}

	// remove unknown extends files
	removeUnknownExtendsFiles(jobContext.GetPackingDir())

	// check some file
	checkingFile := filepath.Join(jobContext.GetPackingDir(), "bin", "rodis")
	err = CheckFileExist(checkingFile)
	if err != nil {
		return fmt.Errorf("artifact unzip fail : %s", artifactUrl)
	}

	return nil
}

// removeUnknownExtendsFiles 맥 운영체제는 BSD tar 명령어를 사용하고 있고 리눅스에서는 GNU tar 명령을 사용하고 있는데
// 동작은 대부분 호환되지만 BSD tar 에서 타르볼에 추가하는 몇 가지 추가 정보를 GNU tar에서 인식할 수 없기 때문에
// "._" 로 시작하는 불필요한 파일들을 생성할 수 있다.
// 함수 내에서는 "."로 시작하는 디렉토리나 파일들을 모두 삭제하도록 한다
func removeUnknownExtendsFiles(packingDir string) {
	removeTargetFiles := make([]string, 0)
	filepath.Walk(packingDir, func(path string, f os.FileInfo, err error) error {
		baseName := filepath.Base(path)
		if len(baseName) > 1 && strings.HasPrefix(baseName, ".") {
			removeTargetFiles = append(removeTargetFiles, path)
		}
		return nil
	})

	for _, file := range removeTargetFiles {
		_ = os.Remove(file)
	}
}
