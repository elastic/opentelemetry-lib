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

import (
	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/common/resource"
)

// Config configures the enrichment attributes produced.
type Config struct {
	Resource    resource.ResourceConfig  `mapstructure:"resource"`
	Scope       ScopeConfig              `mapstructure:"scope"`
	Transaction ElasticTransactionConfig `mapstructure:"elastic_transaction"`
	Span        ElasticSpanConfig        `mapstructure:"elastic_span"`
	SpanEvent   SpanEventConfig          `mapstructure:"span_event"`
}

// ScopeConfig configures the enrichment of scope attributes.
type ScopeConfig struct {
	ServiceFrameworkName    elasticattr.AttributeConfig `mapstructure:"service_framework_name"`
	ServiceFrameworkVersion elasticattr.AttributeConfig `mapstructure:"service_framework_version"`
}

// ElasticTransactionConfig configures the enrichment attributes for the
// spans which are identified as elastic transaction.
type ElasticTransactionConfig struct {
	// TimestampUs is a temporary attribute to enable higher
	// resolution timestamps in Elasticsearch. For more details see:
	// https://github.com/elastic/opentelemetry-dev/issues/374.
	TimestampUs         elasticattr.AttributeConfig `mapstructure:"timestamp_us"`
	Sampled             elasticattr.AttributeConfig `mapstructure:"sampled"`
	ID                  elasticattr.AttributeConfig `mapstructure:"id"`
	Root                elasticattr.AttributeConfig `mapstructure:"root"`
	Name                elasticattr.AttributeConfig `mapstructure:"name"`
	ProcessorEvent      elasticattr.AttributeConfig `mapstructure:"processor_event"`
	RepresentativeCount elasticattr.AttributeConfig `mapstructure:"representative_count"`
	DurationUs          elasticattr.AttributeConfig `mapstructure:"duration_us"`
	Type                elasticattr.AttributeConfig `mapstructure:"type"`
	Result              elasticattr.AttributeConfig `mapstructure:"result"`
	EventOutcome        elasticattr.AttributeConfig `mapstructure:"event_outcome"`
	InferredSpans       elasticattr.AttributeConfig `mapstructure:"inferred_spans"`
	UserAgent           elasticattr.AttributeConfig `mapstructure:"user_agent"`
}

// ElasticSpanConfig configures the enrichment attributes for the spans
// which are NOT identified as elastic transaction.
type ElasticSpanConfig struct {
	// TimestampUs is a temporary attribute to enable higher
	// resolution timestamps in Elasticsearch. For more details see:
	// https://github.com/elastic/opentelemetry-dev/issues/374.
	TimestampUs         elasticattr.AttributeConfig `mapstructure:"timestamp_us"`
	Name                elasticattr.AttributeConfig `mapstructure:"name"`
	ProcessorEvent      elasticattr.AttributeConfig `mapstructure:"processor_event"`
	RepresentativeCount elasticattr.AttributeConfig `mapstructure:"representative_count"`
	TypeSubtype         elasticattr.AttributeConfig `mapstructure:"type_subtype"`
	DurationUs          elasticattr.AttributeConfig `mapstructure:"duration_us"`
	EventOutcome        elasticattr.AttributeConfig `mapstructure:"event_outcome"`
	ServiceTarget       elasticattr.AttributeConfig `mapstructure:"service_target"`
	DestinationService  elasticattr.AttributeConfig `mapstructure:"destination_service"`
	InferredSpans       elasticattr.AttributeConfig `mapstructure:"inferred_spans"`
	UserAgent           elasticattr.AttributeConfig `mapstructure:"user_agent"`
}

// SpanEventConfig configures enrichment attributes for the span events.
type SpanEventConfig struct {
	// TimestampUs is a temporary attribute to enable higher
	// resolution timestamps in Elasticsearch. For more details see:
	// https://github.com/elastic/opentelemetry-dev/issues/374.
	TimestampUs        elasticattr.AttributeConfig `mapstructure:"timestamp_us"`
	TransactionSampled elasticattr.AttributeConfig `mapstructure:"transaction_sampled"`
	TransactionType    elasticattr.AttributeConfig `mapstructure:"transaction_type"`
	ProcessorEvent     elasticattr.AttributeConfig `mapstructure:"processor_event"`

	// For exceptions/errors
	ErrorID               elasticattr.AttributeConfig `mapstructure:"error_id"`
	ErrorExceptionHandled elasticattr.AttributeConfig `mapstructure:"error_exception_handled"`
	ErrorGroupingKey      elasticattr.AttributeConfig `mapstructure:"error_grouping_key"`
	ErrorGroupingName     elasticattr.AttributeConfig `mapstructure:"error_grouping_name"`
}

// Enabled returns a config with all default enrichments enabled.
func Enabled() Config {
	return Config{
		Resource: resource.EnabledConfig(),
		Scope: ScopeConfig{
			ServiceFrameworkName:    elasticattr.AttributeConfig{Enabled: true},
			ServiceFrameworkVersion: elasticattr.AttributeConfig{Enabled: true},
		},
		Transaction: ElasticTransactionConfig{
			TimestampUs:         elasticattr.AttributeConfig{Enabled: true},
			Sampled:             elasticattr.AttributeConfig{Enabled: true},
			ID:                  elasticattr.AttributeConfig{Enabled: true},
			Root:                elasticattr.AttributeConfig{Enabled: true},
			Name:                elasticattr.AttributeConfig{Enabled: true},
			ProcessorEvent:      elasticattr.AttributeConfig{Enabled: true},
			DurationUs:          elasticattr.AttributeConfig{Enabled: true},
			Type:                elasticattr.AttributeConfig{Enabled: true},
			Result:              elasticattr.AttributeConfig{Enabled: true},
			EventOutcome:        elasticattr.AttributeConfig{Enabled: true},
			RepresentativeCount: elasticattr.AttributeConfig{Enabled: true},
			InferredSpans:       elasticattr.AttributeConfig{Enabled: true},
			UserAgent:           elasticattr.AttributeConfig{Enabled: true},
		},
		Span: ElasticSpanConfig{
			TimestampUs:         elasticattr.AttributeConfig{Enabled: true},
			Name:                elasticattr.AttributeConfig{Enabled: true},
			ProcessorEvent:      elasticattr.AttributeConfig{Enabled: true},
			TypeSubtype:         elasticattr.AttributeConfig{Enabled: true},
			DurationUs:          elasticattr.AttributeConfig{Enabled: true},
			EventOutcome:        elasticattr.AttributeConfig{Enabled: true},
			ServiceTarget:       elasticattr.AttributeConfig{Enabled: true},
			DestinationService:  elasticattr.AttributeConfig{Enabled: true},
			RepresentativeCount: elasticattr.AttributeConfig{Enabled: true},
			InferredSpans:       elasticattr.AttributeConfig{Enabled: true},
			UserAgent:           elasticattr.AttributeConfig{Enabled: true},
		},
		SpanEvent: SpanEventConfig{
			TimestampUs:           elasticattr.AttributeConfig{Enabled: true},
			TransactionSampled:    elasticattr.AttributeConfig{Enabled: true},
			TransactionType:       elasticattr.AttributeConfig{Enabled: true},
			ProcessorEvent:        elasticattr.AttributeConfig{Enabled: true},
			ErrorID:               elasticattr.AttributeConfig{Enabled: true},
			ErrorExceptionHandled: elasticattr.AttributeConfig{Enabled: true},
			ErrorGroupingKey:      elasticattr.AttributeConfig{Enabled: true},
			ErrorGroupingName:     elasticattr.AttributeConfig{Enabled: true},
		},
	}
}
