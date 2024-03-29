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
 * @project fatima-go
 * @author dave_01
 * @date 23. 9. 5. 오후 4:48
 */

package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
	"path/filepath"
	"strings"
)

func hasPlatformSupport(zipfile string) bool {
	platformDirPrefix := fmt.Sprintf("/%s/", PlatformDirName)
	return isIncludeDirectory(zipfile, platformDirPrefix)
}

func hasPlatform(zipfile, platform string) bool {
	platformDirPrefix := fmt.Sprintf("/%s/%s/", PlatformDirName, platform)
	return isIncludeDirectory(zipfile, platformDirPrefix)
}

func isIncludeDirectory(zipfile, dirname string) bool {
	zipListing, err := zip.OpenReader(zipfile)
	if err != nil {
		return false
	}
	defer zipListing.Close()

	for _, file := range zipListing.File {
		if !file.FileInfo().IsDir() {
			continue
		}

		if strings.Compare(file.Name, dirname) == 0 {
			return true
		}
	}

	return false
}

// reformArtifact 원본 far 파일에서 타겟 platform 의 바이너리를 base 디렉토리에 복사하고 platform 폴더를 삭제한 후
// 다시 far 파일로 압축한다. 타겟이 되는 서버의 경우 다른 플랫폼 바이너리는 (사용하지 않을 것이기에) 불필요 하므로 제거해서,
// 즉 타겟이 되는 서버의 플랫폼 바이너리만 전송하기 위해 far 파일을 다시 생성하는 것이다.
// 예를 들어 최초 빌드했을때의 far 는 gofar.yaml 정의에 의해 N 개의 플랫폼 바이너리들이 준비되어 있을테고
// 실제 배포시에는 그 중 1개의 플랫폼 바이너리만 필요하므로 나머지는 제거한다.
func reformArtifact(flags share.FatimaCmdFlags, originFarFile string, platform string) (string, error) {
	exposeName := filepath.Base(originFarFile)

	workingDir, err := os.MkdirTemp("", exposeName)
	if err != nil {
		return "", fmt.Errorf("fail to create tmp dir : %s", err.Error())
	}

	err = unzip(originFarFile, workingDir)
	if err != nil {
		return "", fmt.Errorf("fail to unzip : %s", err.Error())
	}

	// copy platform target bin to base dir
	platformBaseDir := filepath.Join(workingDir, PlatformDirName)
	platformTargetDir := filepath.Join(platformBaseDir, platform)
	files, err := os.ReadDir(platformTargetDir)
	if err != nil {
		return "", fmt.Errorf("fail to read dir %s : %s", platformTargetDir, err.Error())
	}

	executableBinNameList := make([]string, 0)
	for _, file := range files {
		executableBinNameList = append(executableBinNameList, file.Name())
		srcFile := filepath.Join(platformTargetDir, file.Name())
		dstFile := filepath.Join(workingDir, file.Name())
		err = copyFile(srcFile, dstFile)
		if err != nil {
			return "", fmt.Errorf("fail to copy %s : %s", srcFile, err.Error())
		}
		_ = os.Chmod(dstFile, 0755)
	}

	// remove all platform directories
	_ = os.RemoveAll(platformBaseDir)

	// markDeployUser
	markDeployUser(flags, workingDir)

	// zip again
	artifactFile := filepath.Join(workingDir, exposeName)
	return artifactFile, zipArtifact(workingDir, artifactFile, executableBinNameList)
}

func markDeployUser(flags share.FatimaCmdFlags, workingDir string) {
	deploymentJsonFile := filepath.Join(workingDir, DeploymentJson)
	dataBytes, err := os.ReadFile(deploymentJsonFile)
	if err != nil {
		fmt.Printf("not found deployment json\n")
		return
	}

	var m map[string]interface{}
	err = json.Unmarshal(dataBytes, &m)
	if err != nil {
		fmt.Printf("fail to unmarshal deployment json : %s", err.Error())
		return
	}

	buildObj := m["build"]
	buildInfo, ok := buildObj.(map[string]interface{})
	if !ok {
		return
	}
	buildInfo["user"] = flags.Username
	data, err := json.Marshal(m)
	if err != nil {
		return
	}

	_ = os.WriteFile(deploymentJsonFile, data, 0644)
}

const (
	PlatformDirName = "platform"
	DeploymentJson  = "deployment.json"
)
