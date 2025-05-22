package mobile

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/elastic/opentelemetry-lib/elasticattr"
	"go.opentelemetry.io/collector/pdata/plog"
)

func EnrichLogEvent(logRecord plog.LogRecord) {
	logRecord.Attributes().PutStr(elasticattr.ProcessorEvent, "error")
	logRecord.Attributes().PutInt(elasticattr.TimestampUs, elasticattr.GetTimestampUs(logRecord.Timestamp()))
	if id, err := newUniqueID(); err == nil {
		logRecord.Attributes().PutStr("error.id", id)
	}
	stacktrace, ok := logRecord.Attributes().Get("exception.stacktrace")
	if ok {
		logRecord.Attributes().PutStr("error.grouping_key", CreateGroupingKey(stacktrace.AsString()))
	}
	logRecord.Attributes().PutStr("error.type", "crash")
}

func newUniqueID() (string, error) {
	var u [16]byte
	if _, err := io.ReadFull(rand.Reader, u[:]); err != nil {
		return "", err
	}

	// convert to string
	buf := make([]byte, 32)
	hex.Encode(buf, u[:])

	return string(buf), nil
}
