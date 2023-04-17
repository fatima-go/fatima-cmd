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
