package mobile

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestEnrichCrashEvents(t *testing.T) {
	logRecord := plog.NewLogRecord()
	logRecord.Attributes().PutStr("event.name", "device.crash")
	logRecord.Attributes().PutStr("exception.message", "Exception message")
	logRecord.Attributes().PutStr("exception.type", "java.lang.RuntimeException")
	logRecord.Attributes().PutStr("exception.stacktrace", `Exception in thread "main" java.lang.RuntimeException: Test exception\\n at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\\n at com.example.GenerateTrace.main(GenerateTrace.java:5)`)

	EnrichLogEvent(logRecord)

	expectedLogRecord := plog.NewLogRecord()
	logRecord.CopyTo(expectedLogRecord)

	expectedLogRecord.Attributes().PutStr("processor.event", "error")

	assert.Empty(t, cmp.Diff(logRecord.Attributes().AsRaw(), expectedLogRecord.Attributes().AsRaw()))
}
