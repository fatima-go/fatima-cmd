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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScreenPasswordString(t *testing.T) {
	var body = []byte(`{"id":"djin.chung","passwd":"qwer12"}`)
	var expect = []byte(`{"id":"djin.chung","passwd":"########"}`)
	assert.Equal(t, screenPasswordString(body), string(expect))

	body = []byte(`{"system":{"code":200,"message":"success"}}`)
	expect = []byte(`{"system":{"code":200,"message":"success"}}`)
	assert.Equal(t, screenPasswordString(body), string(expect))

	body = []byte(`{"system":{"code":200,"password":"qwer12"}}`)
	expect = []byte(`{"system":{"code":200,"password":"########"}}`)
	assert.Equal(t, screenPasswordString(body), string(expect))
}
