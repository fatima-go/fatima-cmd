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
	"github.com/fatima-go/fatima-cmd/share"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var targetProc string

func duplicate() {
	targetProc = os.Args[3]
	if !isAppExist(proc) {
		fmt.Printf("%s process doesn't exist\n", proc)
		return
	}

	if isAppExist(targetProc) {
		fmt.Printf("%s process dir exist\n", targetProc)
		return
	}

	revision, err := createRevisionApp(targetProc)
	if err != nil {
		fmt.Printf("fail to create process for %s : %s\n", targetProc, err.Error())
		return
	}

	fmt.Printf("targetPath : %s\n", revision.dir)
	err = copyToDest(revision.dir)
	if err != nil {
		fmt.Printf("fail to duplicate process for %s : %s\n", proc, err.Error())
		return
	}

	fmt.Printf("successfully duplicated %s to %s\n", proc, targetProc)

	err = linkRevision(targetProc, revision)
	if err != nil {
		fmt.Printf("fail to link revision for %s : %s\n", targetProc, err.Error())
		return
	}

	fmt.Printf("\nyou have to add process in config using roproc command\n")
}

func isAppExist(proc string) bool {
	appLink := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp, proc)
	if _, err := os.Stat(appLink); err == nil {
		return true
	}

	revPath := getRevisionPath(proc)
	if _, err := os.Stat(revPath); err == nil {
		return true
	}

	return false
}

func createRevisionApp(proc string) (Revision, error) {
	tag := createRevisionTag()

	revPath := filepath.Join(getRevisionPath(proc), tag)

	rev := Revision{}
	rev.revision = "R001"
	rev.number = 1
	rev.dir = revPath
	rev.use = true

	err := os.MkdirAll(revPath, 0755)
	return rev, err
}

var suffixList = [...]string{"properties", "xml", "json", "yaml", "sh", "yml", "dat", "p8", "rb", "rbw", "lua"}

func copyToDest(targetPath string) error {
	appLink := filepath.Join(os.Getenv(share.EnvFatimaHome), share.FatimaFolderApp, proc)
	files, err := ioutil.ReadDir(appLink)
	if err != nil {
		return err
	}

	sourceFiles := make([]string, 0)
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if f.Name() == proc {
			sourceFiles = append(sourceFiles, filepath.Join(appLink, f.Name()))
			continue
		}

		if strings.HasPrefix(f.Name(), proc) {
			sourceFiles = append(sourceFiles, filepath.Join(appLink, f.Name()))
		} else {
			for _, s := range suffixList {
				if strings.HasSuffix(f.Name(), s) {
					sourceFiles = append(sourceFiles, filepath.Join(appLink, f.Name()))
					break
				}
			}
		}
	}

	if len(sourceFiles) == 0 {
		return fmt.Errorf("there is no source files...")
	}

	for _, s := range sourceFiles {
		err := transferFile(s, targetPath)
		if err != nil {
			return fmt.Errorf("copy fail [%s -> %s] : %s", s, targetPath, err.Error())
		}
	}

	fmt.Printf("total %d files copied\n", len(sourceFiles))
	return nil
}

func transferFile(srcFile string, targetPath string) error {
	fileName := filepath.Base(srcFile)
	resolved := filepath.Join(targetPath, fileName)
	if strings.HasPrefix(fileName, proc) {
		resolvedName := strings.Replace(fileName, proc, targetProc, 1)
		resolved = filepath.Join(targetPath, resolvedName)
	}

	fmt.Printf("copying to %s\n", resolved)
	return copyFile(srcFile, resolved)
}

func copyFile(srcFile, dstFile string) error {
	from, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer from.Close()
	stat, _ := from.Stat()
	to, err := os.OpenFile(dstFile, os.O_RDWR|os.O_CREATE, stat.Mode())
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}

	return nil
}

const TIME_YYYYMMDDHHMMSS = "2006.01.02-15.04"

// e.g) 2018.08.14-08.35_R006
func createRevisionTag() string {
	return fmt.Sprintf("%s_R001", time.Now().Format(TIME_YYYYMMDDHHMMSS))
}
