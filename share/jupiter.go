//
// Copyright (c) 2017 SK TECHX.
// All right reserved.
//
// This software is the confidential and proprietary information of SK TECHX.
// You shall not disclose such Confidential Information and
// shall use it only in accordance with the terms of the license agreement
// you entered into with SK TECHX.
//
//
// @project fatima-cmd
// @author 1100282
// @date 2017. 10. 31. AM 8:53
//

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
	param["passwd"] = flags.Password

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
