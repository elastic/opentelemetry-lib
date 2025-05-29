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

package logs

import (
	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/common/resource"
	"github.com/elastic/opentelemetry-lib/enrichments/logs/internal/mobile"
	"go.opentelemetry.io/collector/pdata/plog"
)

type Enricher struct {
}

func NewEnricher() *Enricher {
	return &Enricher{}
}

func (e *Enricher) Enrich(logs plog.Logs) {
	resourceLogs := logs.ResourceLogs()
	resourceConfig := resource.ResourceConfig{
		AgentName: elasticattr.AttributeConfig{
			Enabled: true,
		},
	}

	for i := 0; i < resourceLogs.Len(); i++ {
		resourceLog := resourceLogs.At(i)
		res := resourceLog.Resource()
		resource.EnrichResource(res, resourceConfig)
		scopeLogs := resourceLog.ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			logRecords := scopeLogs.At(j).LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				logRecord := logRecords.At(k)
				eventName, ok := getEventName(logRecord)
				if ok {
					ctx := mobile.EventContext{
						EventName:          eventName,
						ResourceAttributes: res.Attributes().AsRaw(),
					}
					mobile.EnrichLogEvent(ctx, logRecord)
				}
			}
		}
	}
}

func getEventName(logRecord plog.LogRecord) (string, bool) {
	if logRecord.EventName() != "" {
		return logRecord.EventName(), true
	}
	attributeValue, ok := logRecord.Attributes().Get("event.name")
	if ok {
		return attributeValue.AsString(), true
	}
	return "", false
}
