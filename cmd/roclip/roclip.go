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
	"os"

	"github.com/fatima-go/fatima-cmd/juno"
	"github.com/fatima-go/fatima-cmd/share"
)

func main() {
	flag.StringVar(&binaryFilename, "b", "", "download binary file from juno data folder")
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run() error {
	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		return fmt.Errorf("fail to build argument for execution: %w", err)
	}

	err = share.GetJunoEndpoint(&fatimaFlags)
	if err != nil {
		return fmt.Errorf("endpoint retrieve fail: %w", err)
	}

	if binaryFilename != "" {
		err = juno.DownloadJunoClipBinary(fatimaFlags, binaryFilename)
	} else {
		err = juno.PrintJunoClipboard(fatimaFlags)
	}
	if err != nil {
		return fmt.Errorf("fail to contact juno: %w", err)
	}
	return nil
}

var binaryFilename string
