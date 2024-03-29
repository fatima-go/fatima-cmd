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

package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSample(t *testing.T) {
	jupiterConfig, err := config.NewJupiterConfigList()
	if err != nil {
		t.Fatalf("NewJupiterConfigList error : %s", err.Error())
		return
	}

	fmt.Printf("%s\n", jupiterConfig)
}

func TestCleanPath(t *testing.T) {
	assert.Equal(t, "http://127.0.0.1:9190", config.RemoveLastSlash("http://127.0.0.1:9190"))
	assert.Equal(t, "http://127.0.0.1:9190", config.RemoveLastSlash("http://127.0.0.1:9190/"))
	assert.Equal(t, "http://127.0.0.1:9190", config.RemoveLastSlash("http://127.0.0.1:9190//"))
}
