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

package trace

import (
	"github.com/elastic/opentelemetry-lib/enrichments/trace/config"
	"github.com/elastic/opentelemetry-lib/enrichments/trace/internal/elastic"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Enricher enriches the OTel traces with attributes required to power
// functionalities in the Elastic UI.
type Enricher struct {
	Config config.Config
}

// NewEnricher creates a new instance of Enricher.
func NewEnricher(cfg config.Config) Enricher {
	return Enricher{
		Config: cfg,
	}
}

// Enrich enriches the OTel traces with attributes required to power
// functionalities in the Elastic UI. The traces are processed as per the
// Elastic's definition of transactions and spans. The traces passed to
// this function are mutated.
func (e *Enricher) Enrich(pt ptrace.Traces) {
	resSpans := pt.ResourceSpans()
	for i := 0; i < resSpans.Len(); i++ {
		resSpan := resSpans.At(i)
		scopeSpans := resSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				elastic.EnrichSpan(spans.At(k), e.Config)
			}
		}
	}
}
