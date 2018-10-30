/*
 *    Copyright 2018 Insolar
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package main

import (
	"testing"

	"github.com/insolar/insolar/configuration"
	"github.com/stretchr/testify/assert"
)

func TestInitComponents(t *testing.T) {
	cfg := configuration.NewConfiguration()
	cfg.Bootstrap.RootKeys = "../../testdata/functional/root_member_keys.json"
	cfg.KeysPath = "../../testdata/functional/bootstrap_keys.json"
	cfg.CertificatePath = "../../testdata/functional/certificate.json"

	cm, _, repl, err := InitComponents(cfg, false)
	assert.NoError(t, err)
	assert.NotNil(t, cm)
	assert.NotNil(t, repl)
}
