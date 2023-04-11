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
	"fmt"
	"github.com/olekukonko/tablewriter"
	"net/http"
	"os"
)

func PrintPreface(respHeader http.Header, body map[string]interface{}) {
	pinfo := NewPackageInfo(body)
	fatimaResponsetime := GetStringFromHeader(respHeader, "Fatima-Response-Time")
	fatimaTimezone := GetStringFromHeader(respHeader, "Fatima-Timezone")
	fmt.Printf("%s (%s)\n", fatimaResponsetime, fatimaTimezone)
	if pinfo.Valid() {
		fmt.Printf("%s\n", pinfo)
	}
}

func PrintTable(headers []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render() // Send output
}
