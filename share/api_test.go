/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with p work for additional information
 * regarding copyright ownership.  The ASF licenses p file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use p file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 *  @project fatima-cmd
 *  @author DeockJin Chung (jin.freestyle@gmail.com)
 *  @date 23. 3. 29. 오후 1:58
 *
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
