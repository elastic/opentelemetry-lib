package resource

import "github.com/elastic/opentelemetry-lib/elasticattr"

// ResourceConfig configures the enrichment of resource attributes.
type ResourceConfig struct {
	AgentName             elasticattr.AttributeConfig `mapstructure:"agent_name"`
	AgentVersion          elasticattr.AttributeConfig `mapstructure:"agent_version"`
	OverrideHostName      elasticattr.AttributeConfig `mapstructure:"override_host_name"`
	DeploymentEnvironment elasticattr.AttributeConfig `mapstructure:"deployment_environment"`
}

func EnabledConfig() ResourceConfig {
	return ResourceConfig{
		AgentName:             elasticattr.AttributeConfig{Enabled: true},
		AgentVersion:          elasticattr.AttributeConfig{Enabled: true},
		OverrideHostName:      elasticattr.AttributeConfig{Enabled: true},
		DeploymentEnvironment: elasticattr.AttributeConfig{Enabled: true},
	}
}
