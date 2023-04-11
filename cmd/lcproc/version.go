//
// Copyright (c) 2018 SK TECHX.
// All right reserved.
//
// This software is the confidential and proprietary information of SK TECHX.
// You shall not disclose such Confidential Information and
// shall use it only in accordance with the terms of the license agreement
// you entered into with SK TECHX.
//
//
// @project fatima-cmd
// @author 1100282
// @date 2018. 8. 14. AM 8:45
//

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"throosea.com/fatima-cmd/share"
)

func versioning() {
	if isRoProgram(proc) {
		fmt.Printf("not permitted ro programs (e.g juno,jupiter,saturn)\n")
		return
	}

	revFolder := getRevisionPath(proc)
	if !isExistRevision(revFolder) {
		fmt.Printf("%s revision folder doesn't exist\n", proc)
		return
	}

	curRev, err := getCurrentRevision(proc)
	if err != nil {
		fmt.Printf("error : %s\n", err.Error())
		return
	}

	revisions := getRevisions(revFolder)

	if len(os.Args) == 3 {
		fmt.Printf("%s revisions...\n", proc)
		for _, r := range revisions {
			if r.number == curRev {
				fmt.Printf("%s <=== Current\n", r.revision)
			} else {
				fmt.Printf("%s\n", r.revision)
			}
		}
		return
	}

	newVersion := strings.ToUpper(strings.ToLower(os.Args[3]))
	if !strings.HasPrefix(newVersion, "R") {
		fmt.Printf("Invalid new revision : %s\n", newVersion)
		return
	}

	newRevision, ok := getVersion(revisions, newVersion)
	if !ok {
		fmt.Printf("Not found revision %s\n", newVersion)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	for true {
		fmt.Printf("%s :: reset to revision %s? (y/n) ", proc, newVersion)
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

	pid := readPidFromFile(proc)
	if pid > 0 {
		if isPidExist(pid) {
			fmt.Printf("pid %d exist. firstly, you have to stop process\n", pid)
			return
		}
	}

	// link again to new version
	err = linkRevision(proc, newRevision)
	if err != nil {
		fmt.Printf("fail to link revision to %s : %s\n", newVersion, err.Error())
		return
	}

	fmt.Printf("process %s tagged to %s revision. start process\n", proc, newRevision.revision)
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

func isExistRevision(revFolder string) bool {
	if stat, err := os.Stat(revFolder); err != nil {
		if os.IsNotExist(err) {
			return false
		} else if !stat.IsDir() {
			return false
		}
		return false
	}
	return true
}

type Revision struct {
	dir      string
	revision string
	number   int
	use      bool
}

func (r Revision) getRelativePath() string {
	idx := strings.LastIndex(r.dir, "revision")
	return r.dir[idx:]
}

type RevisionNumbers []Revision

func (a RevisionNumbers) Len() int           { return len(a) }
func (a RevisionNumbers) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RevisionNumbers) Less(i, j int) bool { return a[j].number < a[i].number }

func getRevisions(revFolder string) []Revision {
	revisions := make([]Revision, 0)
	dirs, err := filepath.Glob(revFolder + "/*_R[0-9]*")
	if err != nil {
		return revisions
	}

	for _, v := range dirs {
		idx := strings.LastIndex(v, "R")
		m, err := strconv.Atoi(v[idx+1:])
		if err != nil {
			continue
		}
		d := Revision{dir: v, revision: v[idx:], number: m}
		revisions = append(revisions, d)
	}

	sort.Sort(RevisionNumbers(revisions))
	return revisions
}

func getCurrentRevision(proc string) (int, error) {
	appProc := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp, proc)
	fi, err := os.Lstat(appProc)
	if err != nil {
		return 0, err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return 0, fmt.Errorf("not symbolic link path")
	}

	linkPath, err := os.Readlink(appProc)
	if err != nil {
		return 0, err
	}

	idx := strings.LastIndex(linkPath, "R")
	if idx < 1 {
		return 0, fmt.Errorf("invalid link name")
	}

	m, err := strconv.Atoi(linkPath[idx+1:])
	if err != nil {
		return 0, fmt.Errorf("invalid revision number format")
	}

	return m, nil
}

func getVersion(revisions []Revision, newVer string) (Revision, bool) {
	for _, r := range revisions {
		if r.revision == newVer {
			return r, true
		}
	}

	return Revision{}, false
}

func readPidFromFile(procName string) int {
	pidFile := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp, procName, share.FatimaFolderAppProc, procName+".pid")

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	var pid = 0
	pid, err = strconv.Atoi(strings.Trim(string(data), "\r\n"))
	if err != nil {
		fmt.Errorf("fail to parse proc[%s] pid value to int : %s\n", procName, err.Error())
		return 0
	}

	return pid
}

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

func executeShell(command string) (string, error) {
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

func isPidExist(pid int) bool {
	command := fmt.Sprintf("ps")
	out, err := executeShell(command)
	if err != nil {
		fmt.Printf("fail to execute command : %s\n", err.Error())
		return true
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimLeft(line, "\r\t\n ")
		items := strings.Split(line, " ")
		if len(items) < 1 {
			continue
		}
		procId, err := strconv.Atoi(items[0])
		if err != nil {
			continue
		}
		if procId == pid {
			return true
		}
	}

	return false
}
