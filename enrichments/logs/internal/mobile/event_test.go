package mobile

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestEnrichCrashEvents(t *testing.T) {
	now := time.Unix(3600, 0)
	timestamp := pcommon.NewTimestampFromTime(now)
	logRecord := plog.NewLogRecord()
	logRecord.SetTimestamp(timestamp)
	logRecord.Attributes().PutStr("event.name", "device.crash")
	logRecord.Attributes().PutStr("exception.message", "Exception message")
	logRecord.Attributes().PutStr("exception.type", "java.lang.RuntimeException")
	logRecord.Attributes().PutStr("exception.stacktrace", `Exception in thread "main" java.lang.RuntimeException: Test exception\\n at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\\n at com.example.GenerateTrace.main(GenerateTrace.java:5)`)

	EnrichLogEvent(logRecord)

	expectedAttributes := map[string]any{
		"processor.event": "error",
		"timestamp.us":    timestamp.AsTime().UnixMicro(),
	}

	assert.Empty(t, cmp.Diff(logRecord.Attributes().AsRaw(), expectedAttributes))
}
