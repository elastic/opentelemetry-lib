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

package mobile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurateStacktrace(t *testing.T) {
	for _, tc := range []struct {
		name        string
		stacktraces []string
		curated     string
	}{
		{
			name:        "standalone_stacktrace",
			stacktraces: []string{readTestFile(t, "stacktrace1_a.txt"), readTestFile(t, "stacktrace1_b.txt"), readTestFile(t, "stacktrace1_c.txt")},
			curated:     readTestFile(t, "curated_stacktrace1.txt"),
		},
		{
			name:        "stacktrace_with_cause",
			stacktraces: []string{readTestFile(t, "stacktrace2_a.txt"), readTestFile(t, "stacktrace2_b.txt"), readTestFile(t, "stacktrace2_c.txt")},
			curated:     readTestFile(t, "curated_stacktrace2.txt"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, stacktrace := range tc.stacktraces {
				assert.Equal(t, tc.curated, curateStacktrace(stacktrace))
			}
		})
	}
}

func TestCreateGroupingKey(t *testing.T) {
	for _, tc := range []struct {
		name        string
		stacktraces []string
		expectedId  string
	}{
		{
			name:        "standalone_stacktrace",
			stacktraces: []string{readTestFile(t, "stacktrace1_a.txt"), readTestFile(t, "stacktrace1_b.txt"), readTestFile(t, "stacktrace1_c.txt")},
			expectedId:  "d03a030670a234802514cc1e8a7ff74846d5890f5f41499109421bf1d58ab310",
		},
		{
			name:        "stacktrace_with_cause",
			stacktraces: []string{readTestFile(t, "stacktrace2_a.txt"), readTestFile(t, "stacktrace2_b.txt"), readTestFile(t, "stacktrace2_c.txt")},
			expectedId:  "b2f6079aa1ef7044f5f2d8a7fb33a71fc60b0067171f92164b87623bda26646b",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, stacktrace := range tc.stacktraces {
				assert.Equal(t, tc.expectedId, CreateGroupingKey(stacktrace))
			}
		})
	}
}

func readTestFile(t *testing.T, fileName string) string {
	bytes, err := os.ReadFile(filepath.Join("testdata", fileName))
	require.NoError(t, err)
	return string(bytes)
}
