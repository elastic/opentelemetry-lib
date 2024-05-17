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
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func remapNetworkMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		dataPoints := metric.Sum().DataPoints()
		for j := 0; j < dataPoints.Len(); j++ {
			dp := dataPoints.At(j)

			var device string
			if d, ok := dp.Attributes().Get("device"); ok {
				device = d.Str()
			} else {
				continue
			}

			if direction, ok := dp.Attributes().Get("direction"); ok {
				name := metric.Name()
				timestamp := dp.Timestamp()
				value := dp.IntValue()

				switch direction.Str() {
				case "receive":
					addDeviceMetric(out, timestamp, dataset, name, device, "in", value)
				case "transmit":
					addDeviceMetric(out, timestamp, dataset, name, device, "out", value)
				}
			}
		}
	}

	return nil
}

func addDeviceMetric(
	out pmetric.MetricSlice,
	timestamp pcommon.Timestamp,
	dataset, name, device, direction string,
	value int64,
) {
	metricsToAdd := map[string]string{
		"system.network.io":      "system.network.%s.bytes",
		"system.network.packets": "system.network.%s.packets",
		"system.network.dropped": "system.network.%s.dropped",
		"system.network.errors":  "system.network.%s.errors",
	}

	if metricNetworkES, ok := metricsToAdd[name]; ok {
		addMetrics(out, dataset,
			func(dp pmetric.NumberDataPoint) {
				dp.Attributes().PutStr("system.network.name", device)
			},
			metric{
				dataType:  pmetric.MetricTypeSum,
				name:      fmt.Sprintf(metricNetworkES, direction),
				timestamp: timestamp,
				intValue:  &value,
			})
	}
}
