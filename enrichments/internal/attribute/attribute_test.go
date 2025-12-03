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

package attribute

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(pcommon.Map)
		key      string
		expected bool
	}{
		{
			name:     "attribute does not exist",
			setup:    func(attrs pcommon.Map) {},
			key:      "nonexistent",
			expected: true,
		},
		{
			name: "string attribute is empty",
			setup: func(attrs pcommon.Map) {
				attrs.PutStr("key", "")
			},
			key:      "key",
			expected: true,
		},
		{
			name: "string attribute is not empty",
			setup: func(attrs pcommon.Map) {
				attrs.PutStr("key", "value")
			},
			key:      "key",
			expected: false,
		},
		{
			name: "int attribute exists",
			setup: func(attrs pcommon.Map) {
				attrs.PutInt("key", 42)
			},
			key:      "key",
			expected: false,
		},
		{
			name: "int attribute with zero value exists",
			setup: func(attrs pcommon.Map) {
				attrs.PutInt("key", 0)
			},
			key:      "key",
			expected: false,
		},
		{
			name: "bool attribute exists",
			setup: func(attrs pcommon.Map) {
				attrs.PutBool("key", true)
			},
			key:      "key",
			expected: false,
		},
		{
			name: "bool attribute with false value exists",
			setup: func(attrs pcommon.Map) {
				attrs.PutBool("key", false)
			},
			key:      "key",
			expected: false,
		},
		{
			name: "double attribute exists",
			setup: func(attrs pcommon.Map) {
				attrs.PutDouble("key", 3.14)
			},
			key:      "key",
			expected: false,
		},
		{
			name: "double attribute with zero value exists",
			setup: func(attrs pcommon.Map) {
				attrs.PutDouble("key", 0.0)
			},
			key:      "key",
			expected: false,
		},
		{
			name:     "slice attribute does not exist",
			setup:    func(attrs pcommon.Map) {},
			key:      "key",
			expected: true,
		},
		{
			name: "slice attribute is empty",
			setup: func(attrs pcommon.Map) {
				attrs.PutEmptySlice("key")
			},
			key:      "key",
			expected: true,
		},
		{
			name: "slice attribute is not empty",
			setup: func(attrs pcommon.Map) {
				slice := attrs.PutEmptySlice("key")
				slice.AppendEmpty().SetStr("value1")
				slice.AppendEmpty().SetStr("value2")
			},
			key:      "key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			tt.setup(attrs)
			result := IsEmpty(attrs, tt.key)
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
