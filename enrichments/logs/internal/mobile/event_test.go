package mobile

import (
	"testing"
	"time"

	"maps"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestEnrichEvents(t *testing.T) {
	now := time.Unix(3600, 0)
	timestamp := pcommon.NewTimestampFromTime(now)
	stacktrace := "Exception in thread \"main\" java.lang.RuntimeException: Test exception\n at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\n at com.example.GenerateTrace.main(GenerateTrace.java:5)"
	stacktraceHash := "96b957020e07ac5c1ed7f86e7df9e3e393ede1284c2c52cb4e8e64f902d37833"

	for _, tc := range []struct {
		name               string
		eventName          string
		input              func() plog.LogRecord
		expectedAttributes map[string]any
	}{
		{
			name:      "crash_event",
			eventName: "device.crash",
			input: func() plog.LogRecord {
				logRecord := plog.NewLogRecord()
				logRecord.SetTimestamp(timestamp)
				logRecord.Attributes().PutStr("event.name", "device.crash")
				logRecord.Attributes().PutStr("exception.message", "Exception message")
				logRecord.Attributes().PutStr("exception.type", "java.lang.RuntimeException")
				logRecord.Attributes().PutStr("exception.stacktrace", stacktrace)
				return logRecord
			},
			expectedAttributes: map[string]any{
				"processor.event":    "error",
				"timestamp.us":       timestamp.AsTime().UnixMicro(),
				"error.grouping_key": stacktraceHash,
				"error.type":         "crash",
				"event.kind":         "event",
			},
		},
		{
			name:      "crash_event_without_timestamp",
			eventName: "device.crash",
			input: func() plog.LogRecord {
				logRecord := plog.NewLogRecord()
				logRecord.SetObservedTimestamp(timestamp)
				logRecord.Attributes().PutStr("event.name", "device.crash")
				logRecord.Attributes().PutStr("exception.message", "Exception message")
				logRecord.Attributes().PutStr("exception.type", "java.lang.RuntimeException")
				logRecord.Attributes().PutStr("exception.stacktrace", stacktrace)
				return logRecord
			},
			expectedAttributes: map[string]any{
				"processor.event":    "error",
				"timestamp.us":       timestamp.AsTime().UnixMicro(),
				"error.grouping_key": stacktraceHash,
				"error.type":         "crash",
				"event.kind":         "event",
			},
		},
		{
			name:      "non_crash_event",
			eventName: "othername",
			input: func() plog.LogRecord {
				logRecord := plog.NewLogRecord()
				logRecord.Attributes().PutStr("event.name", "othername")
				return logRecord
			},
			expectedAttributes: map[string]any{
				"event.kind": "event",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inputLogRecord := tc.input()

			maps.Copy(tc.expectedAttributes, inputLogRecord.Attributes().AsRaw())

			EnrichLogEvent(tc.eventName, inputLogRecord)

			assert.Empty(t, cmp.Diff(inputLogRecord.Attributes().AsRaw(), tc.expectedAttributes, ignoreMapKey("error.id")))
			errorId, ok := inputLogRecord.Attributes().Get("error.id")
			if ok {
				assert.Equal(t, "device.crash", tc.eventName)
				assert.Equal(t, 32, len(errorId.AsString()))
			} else {
				assert.NotEqual(t, "device.crash", tc.eventName)
			}
		})
	}
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
