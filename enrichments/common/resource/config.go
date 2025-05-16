package resource

import "github.com/elastic/opentelemetry-lib/enrichments/common/attribute"

// ResourceConfig configures the enrichment of resource attributes.
type ResourceConfig struct {
	AgentName             attribute.AttributeConfig `mapstructure:"agent_name"`
	AgentVersion          attribute.AttributeConfig `mapstructure:"agent_version"`
	OverrideHostName      attribute.AttributeConfig `mapstructure:"override_host_name"`
	DeploymentEnvironment attribute.AttributeConfig `mapstructure:"deployment_environment"`
}

func EnabledConfig() ResourceConfig {
	return ResourceConfig{
		AgentName:             attribute.AttributeConfig{Enabled: true},
		AgentVersion:          attribute.AttributeConfig{Enabled: true},
		OverrideHostName:      attribute.AttributeConfig{Enabled: true},
		DeploymentEnvironment: attribute.AttributeConfig{Enabled: true},
	}
}
