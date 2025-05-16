package log

import "go.opentelemetry.io/collector/pdata/plog"

type Enricher struct {
}

func (e *Enricher) Enrich(logs plog.Logs) {
	resourceLogs := logs.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		scopeLogs := resourceLogs.At(i).ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			logRecords := scopeLogs.At(j).LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				// todo
			}
		}
	}
}
