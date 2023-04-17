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
	"path/filepath"
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

	// check some file
	checkingFile := filepath.Join(jobContext.GetPackingDir(), "bin", "rodis")
	err = CheckFileExist(checkingFile)
	if err != nil {
		return fmt.Errorf("artifact unzip fail : %s", artifactUrl)
	}

	return nil
}

//func removeDir(path string) {
//	command := fmt.Sprintf("rm -rf %s", path)
//	_, err := ExecuteShell(command)
//	if err != nil {
//		fmt.Printf("[%s] fail to remove dir : %s", path, err.Error())
//		return
//	}
//	fmt.Printf("removed dir : %s\n", path)
//}
