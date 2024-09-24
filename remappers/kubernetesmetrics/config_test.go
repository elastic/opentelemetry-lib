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

package kubernetesmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		opts     []Option
		expected config
	}{
		{
			name: "default",
			opts: nil,
			expected: config{
				KubernetesIntegrationDataset: false,
				Override:                     false,
			},
		},
		{
			name: "k8s_integration_dataset",
			opts: []Option{WithKubernetesIntegrationDataset(true, true)},
			expected: config{
				KubernetesIntegrationDataset: true,
				Override:                     true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, newConfig(tc.opts...))
		})
	}
}
