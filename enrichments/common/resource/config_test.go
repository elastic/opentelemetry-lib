// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package resource

import (
	"reflect"
	"testing"

	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/stretchr/testify/require"
)

func TestEnabled(t *testing.T) {
	config := EnabledConfig()
	assertAllEnabled(t, reflect.ValueOf(config))
}

func assertAllEnabled(t *testing.T, cfg reflect.Value) {
	t.Helper()

	for i := 0; i < cfg.NumField(); i++ {
		rAttrCfg := cfg.Field(i).Interface()
		attrCfg, ok := rAttrCfg.(elasticattr.AttributeConfig)
		require.True(t, ok, "must be a type of AttributeConfig")
		require.True(t, attrCfg.Enabled, "must be enabled")
	}
}
