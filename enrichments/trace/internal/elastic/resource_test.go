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

	"github.com/elastic/opentelemetry-lib/common"
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
				common.AttributeAgentName:    "otlp",
				common.AttributeAgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				common.AttributeAgentName:    "customflavor",
				common.AttributeAgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_distro_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetryDistroName, "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				common.AttributeAgentName:    "customflavor/unknown/elastic",
				common.AttributeAgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_distro_lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKLanguage, "cpp")
				res.Attributes().PutStr(semconv.AttributeTelemetryDistroName, "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				common.AttributeAgentName:    "customflavor/cpp/elastic",
				common.AttributeAgentVersion: "unknown",
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
				common.AttributeAgentName:    "otlp/cpp",
				common.AttributeAgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKLanguage, "cpp")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				common.AttributeAgentName:    "customflavor/cpp",
				common.AttributeAgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_sdkver_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKVersion, "9.999.9")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				common.AttributeAgentName:    "customflavor",
				common.AttributeAgentVersion: "9.999.9",
			},
		},
		{
			name: "sdkname_sdkver_distroname_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKVersion, "9.999.9")
				res.Attributes().PutStr(semconv.AttributeTelemetryDistroName, "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				common.AttributeAgentName:    "customflavor/unknown/elastic",
				common.AttributeAgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_sdkver_distroname_distrover_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKName, "customflavor")
				res.Attributes().PutStr(semconv.AttributeTelemetrySDKVersion, "9.999.9")
				res.Attributes().PutStr(semconv.AttributeTelemetryDistroName, "elastic")
				res.Attributes().PutStr(semconv.AttributeTelemetryDistroVersion, "1.2.3")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				common.AttributeAgentName:    "customflavor/unknown/elastic",
				common.AttributeAgentVersion: "1.2.3",
			},
		},
		{
			name: "host_name_override_with_k8s_node_name",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeHostName, "test-host")
				res.Attributes().PutStr(semconv.AttributeK8SNodeName, "k8s-node")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				semconv.AttributeHostName:    "k8s-node",
				semconv.AttributeK8SNodeName: "k8s-node",
				common.AttributeAgentName:    "otlp",
				common.AttributeAgentVersion: "unknown",
			},
		},
		{
			name: "host_name_if_empty_set_from_k8s_node_name",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(semconv.AttributeK8SNodeName, "k8s-node")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				semconv.AttributeHostName:    "k8s-node",
				semconv.AttributeK8SNodeName: "k8s-node",
				common.AttributeAgentName:    "otlp",
				common.AttributeAgentVersion: "unknown",
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
