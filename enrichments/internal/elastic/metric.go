package elastic

import (
	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/config"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func EnrichMetric(metric pmetric.ResourceMetrics, cfg config.Config) {
	if cfg.Metric.ProcessorEvent.Enabled {
		if _, exists := metric.Resource().Attributes().Get(elasticattr.ProcessorEvent); !exists {
			metric.Resource().Attributes().PutStr(elasticattr.ProcessorEvent, "metric")
		}
	}
}
