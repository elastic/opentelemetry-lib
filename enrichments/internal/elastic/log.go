package elastic

import (
	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/config"
	"go.opentelemetry.io/collector/pdata/plog"
)

func EnrichLog(log plog.LogRecord, cfg config.Config) {
	if cfg.Log.ProcessorEvent.Enabled {
		if _, exists := log.Attributes().Get(elasticattr.ProcessorEvent); !exists {
			log.Attributes().PutStr(elasticattr.ProcessorEvent, "log")
		}
	}
}
