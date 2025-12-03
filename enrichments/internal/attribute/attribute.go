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
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// IsEmpty returns true if the attribute does not exist or is empty.
// For string attributes, returns true if the attribute does not exist or is empty.
// For slice attributes, returns true if the attribute does not exist or has length 0.
// For other types, returns true if the attribute does not exist.
func IsEmpty(attrs pcommon.Map, key string) bool {
	value, exists := attrs.Get(key)
	if !exists {
		return true
	}

	switch value.Type() {
	case pcommon.ValueTypeStr:
		return value.Str() == ""
	case pcommon.ValueTypeSlice:
		return value.Slice().Len() == 0
	default:
		return false
	}
}
