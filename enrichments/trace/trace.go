package trace

import (
	"github.com/elastic/opentelemetry-lib/enrichments/trace/internal/elastic"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Enrich enriches the OTel traces with attributes required to power
// functionalities in the Elastic UI. The traces are processed as per the
// Elastic's definition of transactions and spans. The traces passed to
// this function are mutated.
func Enrich(pt ptrace.Traces) {
	resSpans := pt.ResourceSpans()
	for i := 0; i < resSpans.Len(); i++ {
		resSpan := resSpans.At(i)
		scopeSpans := resSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				var span elastic.Span
				span.Enrich(spans.At(k))
			}
		}
	}
}
