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
	"github.com/elastic/opentelemetry-lib/enrichments/common/attribute"
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
	ServiceFrameworkName    attribute.AttributeConfig `mapstructure:"service_framework_name"`
	ServiceFrameworkVersion attribute.AttributeConfig `mapstructure:"service_framework_version"`
}

// ElasticTransactionConfig configures the enrichment attributes for the
// spans which are identified as elastic transaction.
type ElasticTransactionConfig struct {
	// TimestampUs is a temporary attribute to enable higher
	// resolution timestamps in Elasticsearch. For more details see:
	// https://github.com/elastic/opentelemetry-dev/issues/374.
	TimestampUs         attribute.AttributeConfig `mapstructure:"timestamp_us"`
	Sampled             attribute.AttributeConfig `mapstructure:"sampled"`
	ID                  attribute.AttributeConfig `mapstructure:"id"`
	Root                attribute.AttributeConfig `mapstructure:"root"`
	Name                attribute.AttributeConfig `mapstructure:"name"`
	ProcessorEvent      attribute.AttributeConfig `mapstructure:"processor_event"`
	RepresentativeCount attribute.AttributeConfig `mapstructure:"representative_count"`
	DurationUs          attribute.AttributeConfig `mapstructure:"duration_us"`
	Type                attribute.AttributeConfig `mapstructure:"type"`
	Result              attribute.AttributeConfig `mapstructure:"result"`
	EventOutcome        attribute.AttributeConfig `mapstructure:"event_outcome"`
	InferredSpans       attribute.AttributeConfig `mapstructure:"inferred_spans"`
	UserAgent           attribute.AttributeConfig `mapstructure:"user_agent"`
}

// ElasticSpanConfig configures the enrichment attributes for the spans
// which are NOT identified as elastic transaction.
type ElasticSpanConfig struct {
	// TimestampUs is a temporary attribute to enable higher
	// resolution timestamps in Elasticsearch. For more details see:
	// https://github.com/elastic/opentelemetry-dev/issues/374.
	TimestampUs         attribute.AttributeConfig `mapstructure:"timestamp_us"`
	Name                attribute.AttributeConfig `mapstructure:"name"`
	ProcessorEvent      attribute.AttributeConfig `mapstructure:"processor_event"`
	RepresentativeCount attribute.AttributeConfig `mapstructure:"representative_count"`
	TypeSubtype         attribute.AttributeConfig `mapstructure:"type_subtype"`
	DurationUs          attribute.AttributeConfig `mapstructure:"duration_us"`
	EventOutcome        attribute.AttributeConfig `mapstructure:"event_outcome"`
	ServiceTarget       attribute.AttributeConfig `mapstructure:"service_target"`
	DestinationService  attribute.AttributeConfig `mapstructure:"destination_service"`
	InferredSpans       attribute.AttributeConfig `mapstructure:"inferred_spans"`
	UserAgent           attribute.AttributeConfig `mapstructure:"user_agent"`
}

// SpanEventConfig configures enrichment attributes for the span events.
type SpanEventConfig struct {
	// TimestampUs is a temporary attribute to enable higher
	// resolution timestamps in Elasticsearch. For more details see:
	// https://github.com/elastic/opentelemetry-dev/issues/374.
	TimestampUs        attribute.AttributeConfig `mapstructure:"timestamp_us"`
	TransactionSampled attribute.AttributeConfig `mapstructure:"transaction_sampled"`
	TransactionType    attribute.AttributeConfig `mapstructure:"transaction_type"`
	ProcessorEvent     attribute.AttributeConfig `mapstructure:"processor_event"`

	// For exceptions/errors
	ErrorID               attribute.AttributeConfig `mapstructure:"error_id"`
	ErrorExceptionHandled attribute.AttributeConfig `mapstructure:"error_exception_handled"`
	ErrorGroupingKey      attribute.AttributeConfig `mapstructure:"error_grouping_key"`
	ErrorGroupingName     attribute.AttributeConfig `mapstructure:"error_grouping_name"`
}

// Enabled returns a config with all default enrichments enabled.
func Enabled() Config {
	return Config{
		Resource: resource.EnabledConfig(),
		Scope: ScopeConfig{
			ServiceFrameworkName:    attribute.AttributeConfig{Enabled: true},
			ServiceFrameworkVersion: attribute.AttributeConfig{Enabled: true},
		},
		Transaction: ElasticTransactionConfig{
			TimestampUs:         attribute.AttributeConfig{Enabled: true},
			Sampled:             attribute.AttributeConfig{Enabled: true},
			ID:                  attribute.AttributeConfig{Enabled: true},
			Root:                attribute.AttributeConfig{Enabled: true},
			Name:                attribute.AttributeConfig{Enabled: true},
			ProcessorEvent:      attribute.AttributeConfig{Enabled: true},
			DurationUs:          attribute.AttributeConfig{Enabled: true},
			Type:                attribute.AttributeConfig{Enabled: true},
			Result:              attribute.AttributeConfig{Enabled: true},
			EventOutcome:        attribute.AttributeConfig{Enabled: true},
			RepresentativeCount: attribute.AttributeConfig{Enabled: true},
			InferredSpans:       attribute.AttributeConfig{Enabled: true},
			UserAgent:           attribute.AttributeConfig{Enabled: true},
		},
		Span: ElasticSpanConfig{
			TimestampUs:         attribute.AttributeConfig{Enabled: true},
			Name:                attribute.AttributeConfig{Enabled: true},
			ProcessorEvent:      attribute.AttributeConfig{Enabled: true},
			TypeSubtype:         attribute.AttributeConfig{Enabled: true},
			DurationUs:          attribute.AttributeConfig{Enabled: true},
			EventOutcome:        attribute.AttributeConfig{Enabled: true},
			ServiceTarget:       attribute.AttributeConfig{Enabled: true},
			DestinationService:  attribute.AttributeConfig{Enabled: true},
			RepresentativeCount: attribute.AttributeConfig{Enabled: true},
			InferredSpans:       attribute.AttributeConfig{Enabled: true},
			UserAgent:           attribute.AttributeConfig{Enabled: true},
		},
		SpanEvent: SpanEventConfig{
			TimestampUs:           attribute.AttributeConfig{Enabled: true},
			TransactionSampled:    attribute.AttributeConfig{Enabled: true},
			TransactionType:       attribute.AttributeConfig{Enabled: true},
			ProcessorEvent:        attribute.AttributeConfig{Enabled: true},
			ErrorID:               attribute.AttributeConfig{Enabled: true},
			ErrorExceptionHandled: attribute.AttributeConfig{Enabled: true},
			ErrorGroupingKey:      attribute.AttributeConfig{Enabled: true},
			ErrorGroupingName:     attribute.AttributeConfig{Enabled: true},
		},
	}
}
