package trace

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func BenchmarkEnrich(b *testing.B) {
	traceFile := filepath.Join("testdata", "trace.yaml")
	traces, err := golden.ReadTraces(traceFile)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Enrich(traces)
	}
}

// BenchmarkBaseline benchmarks the baseline of adding a given number of
// attributes a specific trace data. It can be used as an baseline to
// reason about the performance of BenchmarkEnrich.
func BenchmarkBaseline(b *testing.B) {
	attrsToAdd := 3
	var attrKeys []string
	for i := 0; i < attrsToAdd; i++ {
		attrKeys = append(attrKeys, fmt.Sprintf("teststr%d", i))
	}

	traceFile := filepath.Join("testdata", "trace.yaml")
	traces, err := golden.ReadTraces(traceFile)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rspans := traces.ResourceSpans()
		for j := 0; j < rspans.Len(); j++ {
			rspan := rspans.At(j)
			sspan := rspan.ScopeSpans()
			for k := 0; k < sspan.Len(); k++ {
				scope := sspan.At(k)
				spans := scope.Spans()
				for l := 0; l < spans.Len(); l++ {
					span := spans.At(l)
					span.Attributes().Range(func(k string, v pcommon.Value) bool {
						// no-op range
						return true
					})
					span.Attributes().EnsureCapacity(attrsToAdd)
					for _, key := range attrKeys {
						span.Attributes().PutStr(key, "random")
					}
				}
			}
		}
	}
}
