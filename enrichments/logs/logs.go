package logs

import (
	"github.com/elastic/opentelemetry-lib/enrichments/common/attribute"
	"github.com/elastic/opentelemetry-lib/enrichments/common/resource"
	"github.com/elastic/opentelemetry-lib/enrichments/logs/internal/mobile"
	"go.opentelemetry.io/collector/pdata/plog"
)

type Enricher struct {
}

func NewEnricher() *Enricher {
	return &Enricher{}
}

func (e *Enricher) Enrich(logs plog.Logs) {
	resourceLogs := logs.ResourceLogs()
	resourceConfig := resource.ResourceConfig{
		AgentName: attribute.AttributeConfig{
			Enabled: true,
		},
	}

	for i := 0; i < resourceLogs.Len(); i++ {
		resourceLog := resourceLogs.At(i)
		resource.EnrichResource(resourceLog.Resource(), resourceConfig)
		scopeLogs := resourceLog.ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			logRecords := scopeLogs.At(j).LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				mobile.EnrichLogEvent(logRecords.At(k))
			}
		}
	}
}
