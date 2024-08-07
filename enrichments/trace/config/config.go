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

package config

// Config configures the enrichment attributes produced.
type Config struct {
	Resource    ResourceConfig           `mapstructure:"resource"`
	Transaction ElasticTransactionConfig `mapstructure:"elastic_transaction"`
	Span        ElasticSpanConfig        `mapstructure:"elastic_span"`
}

// ResourceConfig configures the enrichment of resource attributes.
type ResourceConfig struct {
	AgentName    AttributeConfig `mapstructure:"agent_name"`
	AgentVersion AttributeConfig `mapstructure:"agent_version"`
}

// ElasticTransactionConfig configures the enrichment attributes for the
// spans which are identified as elastic transaction.
type ElasticTransactionConfig struct {
	Root         AttributeConfig `mapstructure:"root"`
	Name         AttributeConfig `mapstructure:"name"`
	Type         AttributeConfig `mapstructure:"type"`
	Result       AttributeConfig `mapstructure:"result"`
	EventOutcome AttributeConfig `mapstructure:"event_outcome"`
}

// ElasticSpanConfig configures the enrichment attributes for the spans
// which are NOT identified as elastic transaction.
type ElasticSpanConfig struct {
	Name               AttributeConfig `mapstructure:"name"`
	EventOutcome       AttributeConfig `mapstructure:"event_outcome"`
	ServiceTarget      AttributeConfig `mapstructure:"service_target"`
	DestinationService AttributeConfig `mapstructure:"destination_service"`
}

// AttributeConfig is the configuration options for each attribute.
type AttributeConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// Enabled returns a config with all default enrichments enabled.
func Enabled() Config {
	return Config{
		Resource: ResourceConfig{
			AgentName:    AttributeConfig{Enabled: true},
			AgentVersion: AttributeConfig{Enabled: true},
		},
		Transaction: ElasticTransactionConfig{
			Root:         AttributeConfig{Enabled: true},
			Name:         AttributeConfig{Enabled: true},
			Type:         AttributeConfig{Enabled: true},
			Result:       AttributeConfig{Enabled: true},
			EventOutcome: AttributeConfig{Enabled: true},
		},
		Span: ElasticSpanConfig{
			Name:               AttributeConfig{Enabled: true},
			EventOutcome:       AttributeConfig{Enabled: true},
			ServiceTarget:      AttributeConfig{Enabled: true},
			DestinationService: AttributeConfig{Enabled: true},
		},
	}
}
