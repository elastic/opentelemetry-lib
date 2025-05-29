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

	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/trace/config"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv25 "go.opentelemetry.io/otel/semconv/v1.25.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// EnrichResource derives and adds Elastic specific resource attributes.
func EnrichResource(resource pcommon.Resource, cfg config.Config) {
	var c resourceEnrichmentContext
	c.Enrich(resource, cfg.Resource)
}

type resourceEnrichmentContext struct {
	hostName    string
	k8sNodeName string

	telemetrySDKName       string
	telemetrySDKLanguage   string
	telemetrySDKVersion    string
	telemetryDistroName    string
	telemetryDistroVersion string

	deploymentEnvironment     string
	deploymentEnvironmentName string
}

func (s *resourceEnrichmentContext) Enrich(resource pcommon.Resource, cfg config.ResourceConfig) {
	resource.Attributes().Range(func(k string, v pcommon.Value) bool {
		switch k {
		case string(semconv.HostNameKey):
			s.hostName = v.Str()
		case string(semconv.K8SNodeNameKey):
			s.k8sNodeName = v.Str()
		case string(semconv.TelemetrySDKNameKey):
			s.telemetrySDKName = v.Str()
		case string(semconv.TelemetrySDKLanguageKey):
			s.telemetrySDKLanguage = v.Str()
		case string(semconv.TelemetrySDKVersionKey):
			s.telemetrySDKVersion = v.Str()
		case string(semconv.TelemetryDistroNameKey):
			s.telemetryDistroName = v.Str()
		case string(semconv.TelemetryDistroVersionKey):
			s.telemetryDistroVersion = v.Str()
		case string(semconv25.DeploymentEnvironmentKey):
			s.deploymentEnvironment = v.Str()
		case string(semconv.DeploymentEnvironmentNameKey):
			s.deploymentEnvironmentName = v.Str()
		}
		return true
	})

	if cfg.AgentName.Enabled {
		s.setAgentName(resource)
	}
	if cfg.AgentVersion.Enabled {
		s.setAgentVersion(resource)
	}
	if cfg.OverrideHostName.Enabled {
		s.overrideHostNameWithK8sNodeName(resource)
	}
	if cfg.DeploymentEnvironment.Enabled {
		s.setDeploymentEnvironment(resource)
	}
}

// SemConv v1.27.0 deprecated `deployment.environment` and added `deployment.environment.name` in favor of it.
// In the `otel-data` ES plugin we alias `service.environment` to `deployment.environment`.
// ES currently doesn't allow aliases with multiple targets, so if the new field name is used (SemConv v1.27+),
// we duplicate the value and also send it with the old field name to make the alias work.
func (s *resourceEnrichmentContext) setDeploymentEnvironment(resource pcommon.Resource) {
	if s.deploymentEnvironmentName != "" && s.deploymentEnvironment == "" {
		resource.Attributes().PutStr(
			string(semconv25.DeploymentEnvironmentKey),
			s.deploymentEnvironmentName,
		)
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
	resource.Attributes().PutStr(elasticattr.AgentName, agentName)
}

func (s *resourceEnrichmentContext) setAgentVersion(resource pcommon.Resource) {
	agentVersion := "unknown"
	switch {
	case s.telemetryDistroName != "":
		// do not fallback to the Otel SDK version if we have a
		// distro name available as this would only cause confusion
		if s.telemetryDistroVersion != "" {
			agentVersion = s.telemetryDistroVersion
		}
	case s.telemetrySDKVersion != "":
		agentVersion = s.telemetrySDKVersion
	}
	resource.Attributes().PutStr(elasticattr.AgentVersion, agentVersion)
}

func (s *resourceEnrichmentContext) overrideHostNameWithK8sNodeName(resource pcommon.Resource) {
	if s.k8sNodeName == "" {
		return
	}
	// Host name is set same as k8s node name. In case, both host name
	// and k8s node name are set then host name is overridden as this is
	// considered an invalid configuration/smell and k8s node name is
	// given higher preference.
	resource.Attributes().PutStr(
		string(semconv.HostNameKey),
		s.k8sNodeName,
	)
}
