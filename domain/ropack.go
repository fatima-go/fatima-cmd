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
 * @date 23. 9. 6. 오후 1:08
 */

package domain

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type RopackResp struct {
	Summary SummaryResp `json:"summary"`
}

type SummaryResp struct {
	Deployment   []DeploymentResp `json:"deployment"`
	GroupCount   int              `json:"group_count"`
	HostCount    int              `json:"host_count"`
	PackageCount int              `json:"package_count"`
}

func (s SummaryResp) GetDeploymentByGroup(groupName string) (DeploymentResp, error) {
	targetGroupName := strings.ToLower(groupName)
	for _, deployment := range s.Deployment {
		if strings.Compare(strings.ToLower(deployment.GroupName), targetGroupName) == 0 {
			return deployment, nil
		}
	}

	return DeploymentResp{}, fmt.Errorf("not found deployment by group %s", groupName)
}

func (s SummaryResp) IsEmptyDeployment() bool {
	return len(s.Deployment) == 0
}

func (s SummaryResp) HasMultipleHost() bool {
	if s.IsEmptyDeployment() {
		return false
	}

	if len(s.Deployment) > 1 {
		return true
	}

	return len(s.Deployment[0].Deploy) > 1
}

func (s SummaryResp) GetFirstDeploymentHost() (DeployResp, error) {
	if s.IsEmptyDeployment() {
		return DeployResp{}, fmt.Errorf("empty deployment")
	}

	if len(s.Deployment[0].Deploy) == 0 {
		return DeployResp{}, fmt.Errorf("empty deploy")
	}

	return s.Deployment[0].Deploy[0], nil
}

func (s SummaryResp) FindDeployByLocalIpaddress() (DeployResp, error) {
	localIpaddress := getDefaultIpAddress()
	return s.FindDeployByIpaddress(localIpaddress)
}

func (s SummaryResp) FindDeployByHost(host string) (DeployResp, error) {
	targetHost := strings.ToLower(host)
	for _, deployment := range s.Deployment {
		for _, deploy := range deployment.Deploy {
			if strings.Compare(strings.ToLower(deploy.Host), targetHost) == 0 {
				return deploy, nil
			}
		}
	}

	return DeployResp{}, fmt.Errorf("not found deploy by host %s", host)
}

func (s SummaryResp) FindDeployByIpaddress(ipaddress string) (DeployResp, error) {
	for _, deployment := range s.Deployment {
		for _, deploy := range deployment.Deploy {
			if strings.Compare(ipaddress, deploy.GetEndpointIpaddress()) == 0 {
				return deploy, nil
			}
		}
	}

	return DeployResp{}, fmt.Errorf("not found deploy by ip %s", ipaddress)
}

type DeploymentResp struct {
	Deploy    []DeployResp `json:"deploy"`
	GroupName string       `json:"group_name"`
}

func (d DeploymentResp) GetHeaders() []string {
	return []string{"host", "name", "endpoint", "regist_date", "status", "platform"}
}

func (d DeploymentResp) GetData() [][]string {
	data := make([][]string, 0)
	for _, deploy := range d.Deploy {
		record := make([]string, 0)
		if len(deploy.Host) > 0 {
			record = append(record, deploy.Host)
		} else {
			record = append(record, "-")
		}
		if len(deploy.Name) > 0 {
			record = append(record, deploy.Name)
		} else {
			record = append(record, "-")
		}
		if len(deploy.Endpoint) > 0 {
			record = append(record, deploy.Endpoint)
		} else {
			record = append(record, "-")
		}
		if len(deploy.RegistDate) > 0 {
			record = append(record, deploy.RegistDate)
		} else {
			record = append(record, "-")
		}
		if len(deploy.Status) > 0 {
			record = append(record, deploy.Status)
		} else {
			record = append(record, "-")
		}
		record = append(record, deploy.Platform.String())
		data = append(data, record)
	}
	return data
}

type DeployResp struct {
	Endpoint   string       `json:"endpoint"`
	Host       string       `json:"host"`
	Name       string       `json:"name"`
	RegistDate string       `json:"regist_date"`
	Status     string       `json:"status"`
	Platform   PlatformResp `json:"platform"`
}

func (d DeployResp) GetEndpointIpaddress() string {
	wellformed, err := url.Parse(d.Endpoint)
	if err != nil {
		return ""
	}

	portIndex := strings.Index(wellformed.Host, ":")
	if portIndex < 0 {
		return wellformed.Host
	}
	return wellformed.Host[:portIndex]
}

type PlatformResp struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

func (p PlatformResp) String() string {
	return fmt.Sprintf("%s_%s", p.OS, p.Architecture)
}

// getDefaultIpAddress find local ipv4 address
func getDefaultIpAddress() string {
	// func Interfaces() ([]Interface, error)
	inf, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	var min = 100
	ordered := make(map[int]string)
	for _, v := range inf {
		if !(v.Flags&net.FlagBroadcast == net.FlagBroadcast) {
			continue
		}
		if !strings.HasPrefix(v.Name, "eth") && !strings.HasPrefix(v.Name, "en") {
			continue
		}
		addrs, _ := v.Addrs()
		if len(addrs) < 1 {
			continue
		}
		var order int
		if strings.HasPrefix(v.Name, "eth") {
			order, _ = strconv.Atoi(v.Name[3:])
		} else {
			order, _ = strconv.Atoi(v.Name[2:])
		}

		for _, addr := range addrs {
			// check the address type and if it is not a loopback the display it
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ordered[order] = ipnet.IP.String()
					if order <= min {
						min = order
					}
					break
				}
			}
		}
	}

	if len(ordered) < 1 {
		return "127.0.0.1"
	}

	return ordered[min]
}
