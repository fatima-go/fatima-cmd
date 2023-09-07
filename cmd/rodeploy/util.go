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
 * @date 23. 9. 6. 오후 4:52
 */

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func unzip(sourceFarFile, destDir string) error {
	archive, err := zip.OpenReader(sourceFarFile)
	if err != nil {
		return fmt.Errorf("fail to open zip reader %s : %s", sourceFarFile, err.Error())
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(destDir, f.Name)

		if f.Name == "/" {
			continue
		}

		if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path")
		}

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("fail to open %s : %s", filePath, err.Error())
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return fmt.Errorf("fail to open %s file : %s", f.Name, err.Error())
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return fmt.Errorf("fail to copy %s : %s", dstFile.Name(), err.Error())
		}

		dstFile.Close()
		fileInArchive.Close()
	}

	return nil
}

func copyFile(src, dst string) error {
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

type fileMeta struct {
	Path  string
	IsDir bool
}

func zipArtifact(baseDir, artifactFile string, executableBinNameList []string) error {
	var files []fileMeta
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		files = append(files, fileMeta{Path: path, IsDir: info.IsDir()})
		return nil
	})
	if err != nil {
		return err
	}

	z, err := os.Create(artifactFile)
	if err != nil {
		return err
	}
	defer z.Close()

	zw := zip.NewWriter(z)
	defer zw.Close()

	for _, f := range files {
		path := f.Path

		if len(baseDir) == len(path) {
			path = ""
		} else if len(baseDir) < len(path) {
			path = fmt.Sprintf("%c%s", os.PathSeparator, path[len(baseDir)+1:])
		}

		if f.IsDir {
			path = fmt.Sprintf("%s%c", path, os.PathSeparator)
		}

		err = copyIntoZip(zw, path, f, executableBinNameList)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyIntoZip(zw *zip.Writer, path string, f fileMeta, executableBinNameList []string) error {
	header := &zip.FileHeader{
		Name:     path,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}

	baseName := filepath.Base(path)
	for _, executableBin := range executableBinNameList {
		if strings.Compare(baseName, executableBin) == 0 {
			header.SetMode(0755)
			break
		}
	}

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	if f.IsDir {
		return nil
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	return err
}
