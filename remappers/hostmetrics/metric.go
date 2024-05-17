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

package hostmetrics

import (
	"github.com/elastic/opentelemetry-lib/remappers/common"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var emptyMutator = func(pmetric.NumberDataPoint) {}

type metric struct {
	intValue       *int64
	doubleValue    *float64
	name           string
	timestamp      pcommon.Timestamp
	startTimestamp pcommon.Timestamp
	dataType       pmetric.MetricType
}

func addMetrics(
	ms pmetric.MetricSlice,
	dataset string,
	mutator func(dp pmetric.NumberDataPoint),
	metrics ...metric,
) {
	ms.EnsureCapacity(ms.Len() + len(metrics))

	for _, metric := range metrics {
		m := ms.AppendEmpty()
		m.SetName(metric.name)

		var dp pmetric.NumberDataPoint
		switch metric.dataType {
		case pmetric.MetricTypeGauge:
			dp = m.SetEmptyGauge().DataPoints().AppendEmpty()
		case pmetric.MetricTypeSum:
			dp = m.SetEmptySum().DataPoints().AppendEmpty()
		}

		if metric.intValue != nil {
			dp.SetIntValue(*metric.intValue)
		} else if metric.doubleValue != nil {
			dp.SetDoubleValue(*metric.doubleValue)
		}

		dp.SetTimestamp(metric.timestamp)
		if metric.startTimestamp != 0 {
			dp.SetStartTimestamp(metric.startTimestamp)
		}

		dp.Attributes().PutStr("event.provider", "hostmetrics")
		if dataset != "" {
			dp.Attributes().PutStr(common.DatastreamDatasetLabel, dataset)
		}

		mutator(dp)
	}
}
