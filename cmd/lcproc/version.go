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
	"bytes"
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func versioning() {
	if isRoProgram(proc) {
		fmt.Printf("not permitted ro programs (e.g juno,jupiter,saturn)\n")
		return
	}

	revisions, current, err := localRevisions(os.Getenv(share.EnvFatimaHome), proc)
	if err != nil {
		fmt.Printf("error : %s\n", err.Error())
		return
	}

	if len(os.Args) == 3 {
		fmt.Printf("%s revisions...\n", proc)
		for _, r := range revisions {
			mark := "   "
			if same, _ := sameRevision(current, r.dir); same {
				mark = "[O]"
			}
			fmt.Printf("%-21s %s %-12s %s\n", r.revision, mark, r.kind, r.GetBuildSummary())
		}
		return
	}

	newRevision, err := findRevision(revisions, os.Args[3])
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	same, err := sameRevision(current, newRevision.dir)
	if err != nil {
		fmt.Printf("리비전 확인 실패: %s\n", err)
		return
	}
	if same {
		fmt.Printf("%s: 이미 %s 리비전을 가리키고 있습니다. 변경이 필요하지 않습니다.\n", proc, newRevision.revision)
		return
	}
	reader := bufio.NewReader(os.Stdin)
	for true {
		fmt.Printf("%s :: reset to revision %s? (y/n) ", proc, newRevision.revision)
		text, err := reader.ReadString('\n')
		if len(text) == 0 {
			if err != nil {
				return
			}
			continue
		}
		answer := strings.ToLower(strings.Trim(text, "\r\n\t "))
		if answer == "n" {
			return
		} else if answer == "y" {
			break
		}
	}

	err = switchLocalRevision(os.Getenv(share.EnvFatimaHome), proc, newRevision.dir)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	fmt.Printf("process %s tagged to %s revision. start process with rostart\n", proc, newRevision.revision)
}

func isRoProgram(proc string) bool {
	comp := strings.ToLower(proc)
	for _, p := range roPrograms {
		if p == comp {
			return true
		}
	}

	return false
}

func getRevisionPath(proc string) string {
	return filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp, share.FatimaFolderRevision, proc)
}

type Revision struct {
	dir         string
	name        string // folder name
	revision    string // label shown and accepted as argument: R017, o_35e6d8ad1a
	kind        string
	number      int // legacy revision number
	use         bool
	time        time.Time
	createDtime string          // 배포 날짜
	deployment  DeploymentBuild // 배포 브랜치 이름, 마지막 commit message, ...
	operationID string          // v2 manifest
	sha256      string          // v2 manifest
}

func (r Revision) GetBuildSummary() string {
	var buff bytes.Buffer
	buff.WriteString(r.createDtime)
	buff.WriteString(" | ")
	buff.WriteString(fmt.Sprintf("%10s", r.deployment.BuildUser))
	if !r.deployment.HasGit() {
		return buff.String()
	}
	buff.WriteString(" | ")
	buff.WriteString(fmt.Sprintf("%10s", r.deployment.Git.Branch))
	buff.WriteString(" | ")
	buff.WriteString(r.deployment.Git.Commit)
	if r.deployment.Git.HasMessage() {
		buff.WriteString(" | ")
		buff.WriteString(GetTrimmedMessage(r.deployment.Git.Message))
	}

	return buff.String()
}

const (
	yyyyMMddHHmmss     = "2006-01-02 15:04:05"
	deploymentJsonFile = "deployment.json"
)

func linkRevision(proc string, revision Revision) error {
	// unlink $FATIMA_HOME/app/example
	appDir := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp)
	appLink := filepath.Join(appDir, proc)
	fmt.Printf("remove applink : %s\n", appLink)
	err := os.Remove(appLink)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("fail to remove applink : %s", err.Error())
		}
	}

	// link revdir to $FATIMA_HOME/app/example
	relPath, e := filepath.Rel(appDir, revision.dir)
	if e != nil {
		return fmt.Errorf("fail to create relative link : %s", e.Error())
	}

	command := fmt.Sprintf("ln -s %s %s", relPath, proc)
	fmt.Printf("create applink : %s\n", command)

	var cmd *exec.Cmd
	s := regexp.MustCompile("\\s+").Split(command, -1)
	cmd = exec.Command(s[0], s[1:]...)
	cmd.Dir = appDir
	e = cmd.Run()
	if e != nil {
		return fmt.Errorf("fail to process link. command=[%s], err=[%s]", command, e.Error())
	}

	return nil
}
