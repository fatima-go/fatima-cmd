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
	"flag"
	"fmt"
	. "github.com/fatima-go/fatima-cmd/domain"
	"github.com/fatima-go/fatima-cmd/jupiter"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
	"path/filepath"
	"time"
)

var usage = `usage: %s [option] file

deploy package to server
version 1.0.0

positional arguments:
  file                  upload 'far' fatima package file

optional arguments:
  -d    Debug mode
  -g    string
        package group name
  -p    string
        Host and Package. e.g) localhost:default
`

const (
	yyyyMMddHHmmss = "2006-01-02 15:04:05"
)

func main() {
	flag.Usage = func() {
		fmt.Printf(usage, os.Args[0])
	}

	var group string

	flag.StringVar(&group, "g", "", "package group name")

	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		fmt.Printf("fail to build argument for execution : %s", err.Error())
		return
	}

	if len(flag.Args()) < 1 {
		flag.Usage()
		return
	}

	farArtifactFile := flag.Args()[0]
	if !share.IsFileExist(farArtifactFile) {
		fmt.Printf("far farArtifactFile doesn't exist : %s\n", farArtifactFile)
		return
	}

	platformSupport := hasPlatformSupport(farArtifactFile)

	err = share.GetToken(&fatimaFlags)
	if err != nil {
		fmt.Printf("auth fail : %s\n", err.Error())
		return
	}

	fmt.Printf("%s login success...\n", time.Now().Format(yyyyMMddHHmmss))
	if platformSupport {
		ropackResp, err := jupiter.GetPackages(fatimaFlags)
		if err != nil {
			fmt.Printf("fail to get juno package : %s\n", err.Error())
			return
		}

		targetPlatform, err := findPlatform(ropackResp, fatimaFlags, group)
		if err != nil {
			fmt.Printf("fail to find platform : %s\n", err.Error())
			return
		}

		if !hasPlatform(farArtifactFile, targetPlatform) {
			fmt.Printf("far(%s) doesn't support platform %s\n", farArtifactFile, targetPlatform)
			return
		}

		farArtifactFile, err = reformArtifact(fatimaFlags, farArtifactFile, targetPlatform)
		if err != nil {
			fmt.Printf("fail to reform artifact for target platform %s : %s", targetPlatform, err.Error())
			return
		}

		defer func() {
			// reform 에 사용된 tmp 폴더는 삭제해 둔다
			_ = os.RemoveAll(filepath.Dir(farArtifactFile))
		}()
	}

	err = jupiter.DeployPackages(fatimaFlags, group, farArtifactFile)
	if err != nil {
		fmt.Printf("fail to deploy package : %s\n", err.Error())
		return
	}
}

func findPlatform(ropackResp RopackResp, flags share.FatimaCmdFlags, group string) (string, error) {
	// flags.UserPackage 가 존재할 경우 해당 HOST 를 찾는다
	if len(flags.UserPackage) > 0 {
		deploy, err := ropackResp.Summary.FindDeployByHost(flags.UserPackage)
		if err != nil {
			return "", err
		}
		fmt.Printf("%s target %s::%s\n", time.Now().Format(yyyyMMddHHmmss), deploy.Host, deploy.Platform)
		return deploy.Platform.String(), nil
	}

	// 디플로이 정보가 아예 없다면 에러 처리한다
	if ropackResp.Summary.IsEmptyDeployment() {
		return "", fmt.Errorf("deployment is empty")
	}

	// 단 한개의 호스트만 존재할 경우 해당 호스트를 넘겨준다
	if !ropackResp.Summary.HasMultipleHost() {
		deploy, err := ropackResp.Summary.GetFirstDeploymentHost()
		if err != nil {
			return "", err
		}

		fmt.Printf("%s target %s:%s\n", time.Now().Format(yyyyMMddHHmmss), deploy.Host, deploy.Platform)
		return deploy.Platform.String(), nil
	}

	// flags.UserPackage 가 비어 있고 group이 존재할 경우 해당 그룹의 첫번째 호스트를 찾는다
	if len(group) > 0 {
		deployment, err := ropackResp.Summary.GetDeploymentByGroup(group)
		if err != nil {
			return "", err
		}
		if len(deployment.Deploy) == 0 {
			return "", fmt.Errorf("empty deploy for group %s", group)
		}

		fmt.Printf("%s target group %s::%s\n", time.Now().Format(yyyyMMddHHmmss), deployment.GroupName, deployment.Deploy[0].Platform)
		return deployment.Deploy[0].Platform.String(), nil
	}

	// flags.UserPackage, group이 모두 비어 있을 경우 같은 IP 를 찾는다
	deployment, err := ropackResp.Summary.FindDeployByLocalIpaddress()
	if err != nil {
		return "", err
	}

	fmt.Printf("%s target %s::%s\n", time.Now().Format(yyyyMMddHHmmss), deployment.Host, deployment.Platform)
	return deployment.Platform.String(), nil
}
