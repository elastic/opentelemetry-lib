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
	logRecord.Attributes().PutStr("exception.stacktrace", "Exception in thread \"main\" java.lang.RuntimeException: Test exception\n at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\n at com.example.GenerateTrace.main(GenerateTrace.java:5)")
	expectedLogRecord := plog.NewLogRecord()
	logRecord.CopyTo(expectedLogRecord)

	EnrichLogEvent(logRecord)

	expectedLogRecord.Attributes().PutStr("processor.event", "error")
	expectedLogRecord.Attributes().PutInt("timestamp.us", timestamp.AsTime().UnixMicro())
	expectedLogRecord.Attributes().PutStr("error.grouping_key", "96b957020e07ac5c1ed7f86e7df9e3e393ede1284c2c52cb4e8e64f902d37833")
	expectedLogRecord.Attributes().PutStr("error.type", "crash")

	assert.Empty(t, cmp.Diff(logRecord.Attributes().AsRaw(), expectedLogRecord.Attributes().AsRaw(), ignoreMapKey("error.id")))
	errorId, ok := logRecord.Attributes().Get("error.id")
	if !ok {
		assert.Fail(t, "error.id not found")
	}
	assert.Equal(t, 32, len(errorId.AsString()))
}

func ignoreMapKey(k string) cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		mapIndex, ok := p.Last().(cmp.MapIndex)
		if !ok {
			return false
		}
		return mapIndex.Key().String() == k
	}, cmp.Ignore())
}
