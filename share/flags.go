//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with p work for additional information
// regarding copyright ownership.  The ASF licenses p file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use p file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// @project fatima-cmd
// @author DeockJin Chung (jin.freestyle@gmail.com)
// @date 2017. 10. 28. PM 2:52
//

package share

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"throosea.com/fatima-cmd/config"
)

const (
	EnvFatimaHome          = "FATIMA_HOME"
	EnvFatimaProfile       = "FATIMA_PROFILE"
	EnvFatimaRepositoryUri = "FATIMA_REPOSITORY_URI"
	EnvFatimaJupiterUri    = "FATIMA_JUPITER_URI"
	EnvFatimaUsername      = "FATIMA_USERNAME"
	EnvFatimaPassword      = "FATIMA_PASSWORD"
	EnvFatimaTimezone      = "FATIMA_TIMEZONE"
	FatimaFolderApp        = "app"
	FatimaFolderAppProc    = "proc"
	FatimaFolderRevision   = "revision"
	FatimaShellGoaway      = "goaway.sh"
)

type FatimaCmdFlags struct {
	Username    string
	Password    string
	JupiterUri  string
	Timezone    string
	Debug       bool
	UserPackage string
	Args        []string
	Token       string
	Endpoint    string
}

func (c FatimaCmdFlags) Validate() error {
	if len(c.Username) == 0 {
		return ErrInvalidFatimaUsername
	}
	if len(c.Password) == 0 {
		return ErrInvalidFatimaPassword
	}
	if len(c.JupiterUri) == 0 {
		return ErrInvalidFatimaJupiterUri
	}

	return nil
}

func (c FatimaCmdFlags) BuildHeader() map[string]string {
	m := make(map[string]string)
	m["User-Agent"] = "go-fatimaclient"
	if len(c.Token) > 0 {
		m["fatima-auth-token"] = c.Token
		m["fatima-timezone"] = c.Timezone
	}

	return m
}

func (c FatimaCmdFlags) BuildJupiterServiceUrl(url string) string {
	if c.JupiterUri[len(c.JupiterUri)-1] == '/' {
		if url[0] == '/' {
			return c.JupiterUri + url[1:]
		}
		return c.JupiterUri + url
	}

	if url[0] == '/' {
		return c.JupiterUri + url
	}

	return c.JupiterUri + "/" + url
}

func (c FatimaCmdFlags) BuildJunoServiceUrl(url string) string {
	if c.Endpoint[len(c.Endpoint)-1] == '/' {
		if url[0] == '/' {
			return c.Endpoint + url[1:]
		}
		return c.Endpoint + url
	}

	if url[0] == '/' {
		return c.Endpoint + url
	}

	return c.Endpoint + "/" + url
}

var (
	ErrInvalidFatimaUsername   = errors.New("you must provide a username via either -fu or env[FATIMA_USERNAME]")
	ErrInvalidFatimaPassword   = errors.New("you must provide a password via either -fp or env[FATIMA_PASSWORD]")
	ErrInvalidFatimaJupiterUri = errors.New("you must provide a uri to fatima jupiter via either -fj or env[FATIMA_JUPITER_URI]")
)

func BuildFatimaCmdFlags() (FatimaCmdFlags, error) {
	cmdFlags := FatimaCmdFlags{}

	if len(os.Getenv(EnvFatimaHome)) == 0 {
		return cmdFlags, fmt.Errorf("env %s missing", EnvFatimaHome)
	}

	activeContext, err := config.GetActiveContext()
	if err != nil {
		return cmdFlags, err
	}

	flag.BoolVar(&cmdFlags.Debug, "d", false, "Debug mode")
	flag.StringVar(&cmdFlags.UserPackage, "p", "", "Host and Package. e.g) localhost:default")

	flag.Parse()

	cmdFlags.Username = activeContext.User
	cmdFlags.Password = activeContext.GetPassword()
	cmdFlags.JupiterUri = activeContext.Jupiter
	cmdFlags.Timezone = activeContext.Timezone

	cmdFlags.Args = flag.Args()
	return cmdFlags, cmdFlags.Validate()
}
