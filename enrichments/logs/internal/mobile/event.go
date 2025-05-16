package mobile

import "go.opentelemetry.io/collector/pdata/plog"

func EnrichLogEvent(logRecord plog.LogRecord) {
	logRecord.Attributes().PutStr("processor.event", "error")
}
