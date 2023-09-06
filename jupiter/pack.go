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

package jupiter

import (
	"encoding/json"
	"fmt"
	. "github.com/fatima-go/fatima-cmd/domain"
	"github.com/fatima-go/fatima-cmd/share"
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

	// ----------
	ropackResp := RopackResp{}
	err = json.Unmarshal(resp, &ropackResp)
	if err != nil {
		return fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	share.PrintPreface(headers, nil)

	for _, deployment := range ropackResp.Summary.Deployment {
		fmt.Printf("Group : %s\n", deployment.GroupName)
		share.PrintTable(deployment.GetHeaders(), deployment.GetData())
	}

	fmt.Printf("Total group:%d, host:%d, package:%d\n",
		ropackResp.Summary.GroupCount,
		ropackResp.Summary.HostCount,
		ropackResp.Summary.PackageCount)

	return nil
}

/*
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
*/
func GetPackages(flags share.FatimaCmdFlags) (RopackResp, error) {
	resp := RopackResp{}
	url := flags.BuildJupiterServiceUrl(v1PackagesUrl)

	_, respData, err := share.CallFatimaApi(url, flags, nil)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return resp, fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	return resp, nil
}
