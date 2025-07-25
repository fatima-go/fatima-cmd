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

package juno

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"sort"
	"strings"
)

func PrintJunoPackage(flags share.FatimaCmdFlags, sortingOpt SortingOption) error {
	url := flags.BuildJunoServiceUrl(v1PackResourceUrl)

	headers, resp, err := callJuno(url, flags, nil)
	if err != nil {
		return err
	}

	summaryObj := resp["summary"]
	summary, ok := summaryObj.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response structure")
	}

	share.PrintPreface(headers, resp)

	data := make([][]string, 0)
	for _, v := range buildProcessInfoList(resp["process_list"], sortingOpt) {
		data = append(data, v.ToList())
	}

	h := []string{"name", "pid", "status", "cpu", "mem", "fd", "thr", "start_time", "ic", "group"}
	share.PrintTable(h, data)

	fmt.Printf("Total:%d (Alive:%d, Dead:%d), system is %s/%s\n",
		share.GetInt(summary, "total"),
		share.GetInt(summary, "alive"),
		share.GetInt(summary, "dead"),
		share.AsHaString(share.GetInt(resp, "system_status")),
		share.AsPsString(share.GetInt(resp, "system_ps_status")))

	return nil
}

const (
	v1PackResourceUrl = "package/dis/v1"
)

type ProcessInfo struct {
	Index     int
	Cpu       string
	Fd        string
	Thread    string
	Group     string
	Ic        string
	Mem       string
	Name      string
	Pid       string
	Qcount    string
	Qkey      string
	StartTime string
	Status    string
}

func (p ProcessInfo) ToList() []string {
	list := make([]string, 0)
	list = append(list, p.Name)
	list = append(list, p.Pid)
	list = append(list, p.Status)
	list = append(list, p.Cpu)
	list = append(list, p.Mem)
	list = append(list, p.Fd)
	list = append(list, p.Thread)
	list = append(list, p.StartTime)
	list = append(list, p.Ic)
	list = append(list, p.Group)
	return list
}

func buildProcessInfo(m map[string]interface{}) ProcessInfo {
	p := ProcessInfo{}
	p.Index = 0
	p.Cpu = "-"
	p.Fd = "-"
	p.Thread = "-"
	p.Group = "-"
	p.Ic = "-"
	p.Mem = "-"
	p.Name = "-"
	p.Pid = "-"
	p.Qcount = "-"
	p.Qkey = "-"
	p.StartTime = "-"
	p.StartTime = "-"

	for k, v := range m {
		switch k {
		case "cpu":
			p.Cpu = share.AsString(v)
		case "fd":
			p.Fd = share.AsString(v)
		case "thread":
			p.Thread = share.AsString(v)
		case "group":
			p.Group = share.AsString(v)
		case "ic":
			p.Ic = share.AsString(v)
		case "index":
			p.Index = share.AsInt(v)
		case "mem":
			p.Mem = share.AsString(v)
		case "name":
			p.Name = share.AsString(v)
		case "pid":
			p.Pid = share.AsString(v)
		case "qcount":
			p.Qcount = share.AsString(v)
		case "qkey":
			p.Qkey = share.AsString(v)
		case "start_time":
			p.StartTime = share.AsString(v)
		case "status":
			p.Status = share.AsString(v)
		}
	}

	return p
}

func buildProcessInfoList(data interface{}, sortingOpt SortingOption) []ProcessInfo {
	list := make([]ProcessInfo, 0)

	if val, ok := data.([]interface{}); ok {
		for _, v := range val {
			if m, ok := v.(map[string]interface{}); ok {
				list = append(list, buildProcessInfo(m))
			}
		}
	}

	switch sortingOpt.sortType {
	case SortTypeName:
		list = sortWithName(list, sortingOpt.order)
	case SortTypeIndex:
		sort.Sort(ByIndex(list))
	}

	return list
}

func sortWithName(list []ProcessInfo, order Order) []ProcessInfo {
	if order == OrderNone {
		return list
	}

	orderedProcInfo := make([]ProcessInfo, 0)
	for _, groupProcInfo := range splitProcessWithGroup(list) {
		if !IsOpmGroup(groupProcInfo.GroupName) {
			if order == OrderAsc {
				sort.Sort(ByProcessNameAsc(groupProcInfo.Entries))
			} else if order == OrderDesc {
				sort.Sort(ByProcessNameDesc(groupProcInfo.Entries))
			}
		}
		orderedProcInfo = append(orderedProcInfo, groupProcInfo.Entries...)
	}
	return orderedProcInfo
}

type ByIndex []ProcessInfo

func (a ByIndex) Len() int           { return len(a) }
func (a ByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByIndex) Less(i, j int) bool { return a[i].Index < a[j].Index }

type ByProcessNameAsc []ProcessInfo

func (n ByProcessNameAsc) Len() int      { return len(n) }
func (n ByProcessNameAsc) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n ByProcessNameAsc) Less(i, j int) bool {
	return n[i].Name < n[j].Name
}

type ByProcessNameDesc []ProcessInfo

func (n ByProcessNameDesc) Len() int      { return len(n) }
func (n ByProcessNameDesc) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n ByProcessNameDesc) Less(i, j int) bool {
	return n[i].Name > n[j].Name
}

func splitProcessWithGroup(list []ProcessInfo) []*GroupProcessInfo {
	groupList := make([]*GroupProcessInfo, 0)

	for _, proc := range list {
		info := findGroupInList(groupList, proc.Group)
		if info == nil {
			info = &GroupProcessInfo{}
			info.GroupName = proc.Group
			groupList = append(groupList, info)
		}
		info.addEntry(proc)
	}

	return groupList
}

func findGroupInList(list []*GroupProcessInfo, groupName string) *GroupProcessInfo {
	for _, record := range list {
		if record.GroupName == groupName {
			return record
		}
	}
	return nil
}

type GroupProcessInfo struct {
	GroupName string
	Entries   []ProcessInfo
}

func (g *GroupProcessInfo) addEntry(entry ProcessInfo) {
	if g.Entries == nil {
		g.Entries = make([]ProcessInfo, 0)
	}
	g.Entries = append(g.Entries, entry)
}

const (
	ProcessGroupOpm = "opm"
)

func IsOpmGroup(opm string) bool {
	return strings.ToLower(opm) == ProcessGroupOpm
}
