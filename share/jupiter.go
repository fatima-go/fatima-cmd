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

func GetToken(flags *FatimaCmdFlags) error {
	authUrl := flags.BuildJupiterServiceUrl(v1LoginResourceUrl)

	param := make(map[string]interface{})
	param["id"] = flags.Username
	param["passwd"] = flags.GetEncryptedPassword()

	b, err := json.Marshal(param)
	if err != nil {
		return fmt.Errorf("fail to marshal to json : %s\n", err.Error())
	}

	_, resp, err := CallFatimaApi(authUrl, *flags, b)
	if err != nil {
		return err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return fmt.Errorf("invalid repsonse message sturcture : %s", err.Error())
	}

	token := respMap["token"]
	if token == nil {
		return fmt.Errorf("there is not token")
	}

	if val, ok := token.(string); ok {
		flags.Token = val
		return nil
	}

	return fmt.Errorf("invalid token type. real type=%v", reflect.ValueOf(token).Type())
}

const (
	v1LoginResourceUrl = "/auth/login/v1"
)
