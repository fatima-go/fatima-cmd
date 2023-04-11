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
	"encoding/json"
	"fmt"
	"reflect"
)

func GetJunoEndpoint(flags *FatimaCmdFlags) error {
	err := GetToken(flags)
	if err != nil {
		return fmt.Errorf("auth fail : %s\n", err.Error())
	}

	url := flags.JupiterUri + v1EndpointResourceUrl

	var b []byte
	if len(flags.UserPackage) > 0 {
		param := make(map[string]interface{})
		param["package"] = flags.UserPackage

		b, err = json.Marshal(param)
		if err != nil {
			return fmt.Errorf("fail to marshal to json : %s\n", err.Error())
		}
	}

	_, resp, err := CallFatimaApi(url, *flags, b)
	if err != nil {
		return err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return fmt.Errorf("invalid response message structure : %s", err.Error())
	}

	if !isSuccess(respMap) {
		message := GetSystemMessage(respMap)
		return fmt.Errorf("%s", message)
	}

	endpoint := respMap["endpoint"]
	if endpoint == nil {
		return fmt.Errorf("there is no endpoint")
	}

	if val, ok := endpoint.(string); ok {
		flags.Endpoint = val
		return nil
	}

	return fmt.Errorf("invalid endpoint type. real type=%v", reflect.ValueOf(endpoint).Type())
}

const (
	v1EndpointResourceUrl = "/juno/retrieve/v1"
)
