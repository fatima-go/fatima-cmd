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

package jupiter

import (
	"encoding/json"
	"fmt"
	"throosea.com/fatima-cmd/share"
)

const (
	v1PackagesUrl = "/pack/v1"
)

func PrintPackages(flags share.FatimaCmdFlags) error {
	url := flags.BuildJupiterServiceUrl(v1PackagesUrl)

	headers, resp, err := share.CallFatimaApi(url, flags, nil)
	if err != nil {
		return err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	summaryObj := respMap["summary"]
	summary, ok := summaryObj.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response structure")
	}

	share.PrintPreface(headers, nil)

	deploymentObj := summary["deployment"]
	deploymentList, ok := deploymentObj.([]interface{})
	for _, d := range deploymentList {
		groupDep := d.(map[string]interface{})
		if groupDep == nil {
			continue
		}

		groupName := share.GetString(groupDep, "group_name")
		fmt.Printf("Group : %s\n", groupName)

		data := make([][]string, 0)
		for _, v := range buildDeploymentList(groupDep["deploy"]) {
			data = append(data, v.ToList())
		}
		h := []string{"host", "name", "endpoint", "regist_date", "status", "platform"}
		share.PrintTable(h, data)
	}

	fmt.Printf("Total group:%d, host:%d, package:%d\n",
		share.GetInt(summary, "group_count"),
		share.GetInt(summary, "host_count"),
		share.GetInt(summary, "package_count"))

	return nil
}

type Deployment struct {
	Host       string
	Name       string
	Endpoint   string
	RegistDate string
	Status     string
	Platform   string
}

func (p Deployment) ToList() []string {
	list := make([]string, 0)
	list = append(list, p.Host)
	list = append(list, p.Name)
	list = append(list, p.Endpoint)
	list = append(list, p.RegistDate)
	list = append(list, p.Status)
	list = append(list, p.Platform)
	return list
}

func buildDeployment(m map[string]interface{}) Deployment {
	p := Deployment{}
	p.Host = "-"
	p.Name = "-"
	p.Endpoint = "-"
	p.RegistDate = "-"
	p.Status = "-"
	p.Platform = "-"

	for k, v := range m {
		switch k {
		case "name":
			p.Name = share.AsString(v)
		case "host":
			p.Host = share.AsString(v)
		case "endpoint":
			p.Endpoint = share.AsString(v)
		case "status":
			p.Status = share.AsString(v)
		case "regist_date":
			p.RegistDate = share.AsString(v)
		case "platform":
			if platform, ok := v.(map[string]interface{}); ok {
				os := platform["os"]
				arch := platform["architecture"]
				p.Platform = fmt.Sprintf("%s_%s", os, arch)
			}
		}
	}

	return p
}

func buildDeploymentList(data interface{}) []Deployment {
	list := make([]Deployment, 0)

	if val, ok := data.([]interface{}); ok {
		for _, v := range val {
			if m, ok := v.(map[string]interface{}); ok {
				list = append(list, buildDeployment(m))
			}
		}
	}

	return list
}
