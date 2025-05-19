package mobile

import (
	"github.com/elastic/opentelemetry-lib/elasticattr"
	"go.opentelemetry.io/collector/pdata/plog"
)

func EnrichLogEvent(logRecord plog.LogRecord) {
	logRecord.Attributes().PutStr(elasticattr.ProcessorEvent, "error")
	logRecord.Attributes().PutInt(elasticattr.TimestampUs, elasticattr.GetTimestampUs(logRecord.Timestamp()))
}
