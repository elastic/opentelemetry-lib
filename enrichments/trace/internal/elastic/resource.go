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
	"fmt"

	"github.com/elastic/opentelemetry-lib/enrichments/trace/config"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/collector/semconv/v1.22.0"
)

// EnrichResource derives and adds Elastic specific resource attributes.
func EnrichResource(resource pcommon.Resource, cfg config.Config) {
	var c resourceEnrichmentContext
	c.Enrich(resource, cfg.Resource)
}

type resourceEnrichmentContext struct {
	telemetrySDKName       string
	telemetrySDKLanguage   string
	telemetrySDKVersion    string
	telemetryDistroName    string
	telemetryDistroVersion string
}

func (s *resourceEnrichmentContext) Enrich(resource pcommon.Resource, cfg config.ResourceConfig) {
	resource.Attributes().Range(func(k string, v pcommon.Value) bool {
		switch k {
		case semconv.AttributeTelemetrySDKName:
			s.telemetrySDKName = v.Str()
		case semconv.AttributeTelemetrySDKLanguage:
			s.telemetrySDKLanguage = v.Str()
		case semconv.AttributeTelemetrySDKVersion:
			s.telemetrySDKVersion = v.Str()
		case semconv.AttributeTelemetryDistroName:
			s.telemetryDistroName = v.Str()
		case semconv.AttributeTelemetryDistroVersion:
			s.telemetryDistroVersion = v.Str()
		}
		return true
	})

	if cfg.AgentName.Enabled {
		s.setAgentName(resource)
	}
	if cfg.AgentVersion.Enabled {
		s.setAgentVersion(resource)
	}
}

func (s *resourceEnrichmentContext) setAgentName(resource pcommon.Resource) {
	agentName := "otlp"
	if s.telemetrySDKName != "" {
		agentName = s.telemetrySDKName
	}
	switch {
	case s.telemetryDistroName != "":
		agentLang := "unknown"
		if s.telemetrySDKLanguage != "" {
			agentLang = s.telemetrySDKLanguage
		}
		agentName = fmt.Sprintf(
			"%s/%s/%s",
			agentName,
			agentLang,
			s.telemetryDistroName,
		)
	case s.telemetrySDKLanguage != "":
		agentName = fmt.Sprintf(
			"%s/%s",
			agentName,
			s.telemetrySDKLanguage,
		)
	}
	resource.Attributes().PutStr(AttributeAgentName, agentName)
}

func (s *resourceEnrichmentContext) setAgentVersion(resource pcommon.Resource) {
	agentVersion := "unknown"
	switch {
	case s.telemetryDistroName != "" && s.telemetryDistroVersion != "":
		// do not want to fallback to the Otel SDK version if we have a
		// distro name available as this would only cause confusion
		agentVersion = s.telemetryDistroVersion
	case s.telemetrySDKVersion != "":
		agentVersion = s.telemetrySDKVersion
	}
	resource.Attributes().PutStr(AttributeAgentVersion, agentVersion)
}
