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

	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/config"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv25 "go.opentelemetry.io/otel/semconv/v1.25.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
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
				elasticattr.AgentName:    "otlp",
				elasticattr.AgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKNameKey), "customflavor")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "customflavor",
				elasticattr.AgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_distro_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKNameKey), "customflavor")
				res.Attributes().PutStr(string(semconv.TelemetryDistroNameKey), "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "customflavor/unknown/elastic",
				elasticattr.AgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_distro_lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKNameKey), "customflavor")
				res.Attributes().PutStr(string(semconv.TelemetrySDKLanguageKey), "cpp")
				res.Attributes().PutStr(string(semconv.TelemetryDistroNameKey), "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "customflavor/cpp/elastic",
				elasticattr.AgentVersion: "unknown",
			},
		},
		{
			name: "lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKLanguageKey), "cpp")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "otlp/cpp",
				elasticattr.AgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_lang_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKNameKey), "customflavor")
				res.Attributes().PutStr(string(semconv.TelemetrySDKLanguageKey), "cpp")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "customflavor/cpp",
				elasticattr.AgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_sdkver_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKNameKey), "customflavor")
				res.Attributes().PutStr(string(semconv.TelemetrySDKVersionKey), "9.999.9")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "customflavor",
				elasticattr.AgentVersion: "9.999.9",
			},
		},
		{
			name: "sdkname_sdkver_distroname_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKNameKey), "customflavor")
				res.Attributes().PutStr(string(semconv.TelemetrySDKVersionKey), "9.999.9")
				res.Attributes().PutStr(string(semconv.TelemetryDistroNameKey), "elastic")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "customflavor/unknown/elastic",
				elasticattr.AgentVersion: "unknown",
			},
		},
		{
			name: "sdkname_sdkver_distroname_distrover_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.TelemetrySDKNameKey), "customflavor")
				res.Attributes().PutStr(string(semconv.TelemetrySDKVersionKey), "9.999.9")
				res.Attributes().PutStr(string(semconv.TelemetryDistroNameKey), "elastic")
				res.Attributes().PutStr(string(semconv.TelemetryDistroVersionKey), "1.2.3")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				elasticattr.AgentName:    "customflavor/unknown/elastic",
				elasticattr.AgentVersion: "1.2.3",
			},
		},
		{
			name: "host_name_override_with_k8s_node_name",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.HostNameKey), "test-host")
				res.Attributes().PutStr(string(semconv.K8SNodeNameKey), "k8s-node")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				string(semconv.HostNameKey):    "k8s-node",
				string(semconv.K8SNodeNameKey): "k8s-node",
				elasticattr.AgentName:          "otlp",
				elasticattr.AgentVersion:       "unknown",
			},
		},
		{
			name: "host_name_if_empty_set_from_k8s_node_name",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.K8SNodeNameKey), "k8s-node")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				string(semconv.HostNameKey):    "k8s-node",
				string(semconv.K8SNodeNameKey): "k8s-node",
				elasticattr.AgentName:          "otlp",
				elasticattr.AgentVersion:       "unknown",
			},
		},
		{
			// Pre SemConv 1.27
			name: "deployment_environment_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv25.DeploymentEnvironmentKey), "prod")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				string(semconv25.DeploymentEnvironmentKey): "prod",
				elasticattr.AgentName:                      "otlp",
				elasticattr.AgentVersion:                   "unknown",
			},
		},
		{
			// SemConv 1.27+ with new `deployment.environment.name` field
			name: "deployment_environment_name_set",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.DeploymentEnvironmentNameKey), "prod")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				// To satisfy aliases defined in ES, we duplicate the value for both fields.
				string(semconv25.DeploymentEnvironmentKey):   "prod",
				string(semconv.DeploymentEnvironmentNameKey): "prod",
				elasticattr.AgentName:                        "otlp",
				elasticattr.AgentVersion:                     "unknown",
			},
		},
		{
			// Mixed pre and post SemConv 1.27 versions (should be an edge case, but some EDOTs might do this).
			name: "deployment_environment_mixed",
			input: func() pcommon.Resource {
				res := pcommon.NewResource()
				res.Attributes().PutStr(string(semconv.DeploymentEnvironmentNameKey), "prod")
				res.Attributes().PutStr(string(semconv25.DeploymentEnvironmentKey), "test")
				return res
			}(),
			config: config.Enabled().Resource,
			enrichedAttrs: map[string]any{
				// If both are set, we don't touch those values and take them as they are.
				string(semconv25.DeploymentEnvironmentKey):   "test",
				string(semconv.DeploymentEnvironmentNameKey): "prod",
				elasticattr.AgentName:                        "otlp",
				elasticattr.AgentVersion:                     "unknown",
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
