// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mobile

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/elastic/opentelemetry-lib/elasticattr"
	"go.opentelemetry.io/collector/pdata/plog"
)

// EventContext contains contextual information for log event enrichment
type EventContext struct {
	ResourceAttributes map[string]any
	EventName          string
}

func EnrichLogEvent(ctx EventContext, logRecord plog.LogRecord) {
	logRecord.Attributes().PutStr(elasticattr.EventKind, "event")

	if ctx.EventName == "device.crash" {
		enrichCrashEvent(logRecord, ctx.ResourceAttributes)
	}
}

func enrichCrashEvent(logRecord plog.LogRecord, resourceAttrs map[string]any) {
	timestamp := logRecord.Timestamp()
	if timestamp == 0 {
		timestamp = logRecord.ObservedTimestamp()
	}
	logRecord.Attributes().PutStr(elasticattr.ProcessorEvent, "error")
	logRecord.Attributes().PutInt(elasticattr.TimestampUs, elasticattr.GetTimestampUs(timestamp))
	if id, err := newUniqueID(); err == nil {
		logRecord.Attributes().PutStr(elasticattr.ErrorID, id)
	}
	stacktrace, ok := logRecord.Attributes().Get("exception.stacktrace")
	if ok {
		language, hasLanguage := resourceAttrs["telemetry.sdk.language"]
		if hasLanguage && language == "java" {
			logRecord.Attributes().PutStr(elasticattr.ErrorGroupingKey, CreateJavaStacktraceGroupingKey(stacktrace.AsString()))
		}
	}
	logRecord.Attributes().PutStr(elasticattr.ErrorType, "crash")
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
