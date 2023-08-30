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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type UpdateExecutor interface {
	Name() string
	Execute(ctx *UpdateContext) error
}

type UpdateContext struct {
	Platform      PlatformInfo
	WorkingDir    string
	FatimaHomeDir string
	executorList  []UpdateExecutor
	Packing       PackingInfo
}

type PackingInfo struct {
	User      string `json:"user"`
	BuildTime string `json:"build_time"`
}

const (
	FatimaRepoUrl         = "https://github.com/fatima-go/fatima-download/raw/main"
	EnvFatimaHome         = "FATIMA_HOME"
	FatimaBasePackingName = "fatima-package"
	PackingFileName       = "packing-info.json"
)

func (u *UpdateContext) GetDownloadUrl() string {
	if len(artifactUrl) > 0 {
		return artifactUrl
	}

	return fmt.Sprintf("%s/fatima-package.%s-%s.tar.gz",
		FatimaRepoUrl, u.Platform.Os, u.Platform.Architecture)
}

func (u *UpdateContext) GetPackingDir() string {
	return filepath.Join(u.WorkingDir, FatimaBasePackingName)
}

func (u *UpdateContext) LoadPackingInfo() bool {
	packingFilePath := filepath.Join(u.WorkingDir, FatimaBasePackingName, PackingFileName)

	b, err := os.ReadFile(packingFilePath)
	if err != nil {
		fmt.Printf("not found %s : %s\n", PackingFileName, err.Error())
		return false
	}

	err = json.Unmarshal(b, &u.Packing)
	if err != nil {
		fmt.Printf("fail to unmarshal packing-info : %s\n", err.Error())
		return false
	}

	return true
}

func (u *UpdateContext) Close() {
	os.RemoveAll(u.WorkingDir)
}

type PlatformInfo struct {
	Os           string
	Architecture string
}

func NewUpdateContext(command string) (*UpdateContext, error) {
	ctx := &UpdateContext{}
	ctx.executorList = make([]UpdateExecutor, 0)

	ctx.executorList = append(ctx.executorList, ExecuteDownload{})

	switch command {
	case "bin":
		ctx.executorList = append(ctx.executorList, ExecuteUpdateBin{})
	case "opm":
		ctx.executorList = append(ctx.executorList, ExecuteUpdateOpm{})
	case "all":
		ctx.executorList = append(ctx.executorList, ExecuteUpdateBin{})
		ctx.executorList = append(ctx.executorList, ExecuteUpdateOpm{})
	default:
		return ctx, fmt.Errorf("undefined command %s\n", command)
	}

	ctx.executorList = append(ctx.executorList, ExecuteReportPacking{})

	ctx.FatimaHomeDir = os.Getenv(EnvFatimaHome)
	if len(ctx.FatimaHomeDir) == 0 {
		return ctx, fmt.Errorf("env %s missing", EnvFatimaHome)
	}

	ctx.Platform.Os = runtime.GOOS
	ctx.Platform.Architecture = runtime.GOARCH
	tmpdir, err := os.MkdirTemp("", "fatima-package")
	if err != nil {
		return ctx, fmt.Errorf("fail to prepare temp dir : %s", err.Error())
	}
	ctx.WorkingDir = tmpdir

	return ctx, nil
}
