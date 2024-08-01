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

package elastic

import (
	"testing"

	"github.com/elastic/opentelemetry-lib/enrichments/trace/config"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/collector/semconv/v1.25.0"
)

func TestResourceEnrich(t *testing.T) {
	for _, tc := range []struct {
		name          string
		input         pcommon.Resource
		config        config.ResourceConfig
		enrichedAttrs map[string]any
	}{
		{
			name:          "all_disabled",
			input:         pcommon.NewResource(),
			enrichedAttrs: map[string]any{},
		},
		{
			name:   "empty",
			input:  pcommon.NewResource(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				AttributeAgentName: "otlp",
			},
		},
		{
			name: "sdk_name_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				AttributeAgentName: "customflavor",
			},
		},
		{
			name: "sdk_name_distro_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetryDistroName, "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				AttributeAgentName: "customflavor/unknown/elastic",
			},
		},
		{
			name: "sdk_name_distro_lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKLanguage, "cpp")
				res.Attributes().PutStr(semconv.AttributeTelemetryDistroName, "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				AttributeAgentName: "customflavor/cpp/elastic",
			},
		},
		{
			name: "lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKLanguage, "cpp")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				AttributeAgentName: "otlp/cpp",
			},
		},
		{
			name: "sdk_name_lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKLanguage, "cpp")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				AttributeAgentName: "customflavor/cpp",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Merge existing resource attrs with the attrs added
			// by enrichment to get the expected attributes.
			expectedAttrs := tc.input.Attributes().AsRaw()
			for k, v := range tc.enrichedAttrs {
				expectedAttrs[k] = v
			}

			EnrichResource(tc.input, config.Config{
				Resource: tc.config,
			})

			assert.Empty(t, cmp.Diff(expectedAttrs, tc.input.Attributes().AsRaw()))
		})
	}
}
