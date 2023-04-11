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

package juno

import (
	"encoding/json"
	"fmt"
	"net/http"
	"throosea.com/fatima-cmd/share"
)

func callJuno(url string, flags share.FatimaCmdFlags, b []byte) (http.Header, map[string]interface{}, error) {
	headers, resp, err := share.CallFatimaApi(url, flags, b)
	if err != nil {
		return nil, nil, err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	return headers, respMap, nil
}
