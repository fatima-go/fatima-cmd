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

package share

import (
	"fmt"
	"net/http"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
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
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader(headers),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	_ = table.Bulk(data)
	_ = table.Render()
}
