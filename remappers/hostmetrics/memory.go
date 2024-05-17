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
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/elastic/opentelemetry-lib/remappers/internal"
)

func remapMemoryMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var total, free, cached, usedBytes, actualFree, actualUsedBytes int64
	var usedPercent, actualUsedPercent float64

	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.memory.usage":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}

				value := dp.IntValue()
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "cached":
						cached = value
						total += value
					case "free":
						free = value
						usedBytes -= value
						total += value
					case "used":
						total += value
						actualUsedBytes += value
					case "buffered":
						total += value
						actualUsedBytes += value
					case "slab_unreclaimable":
						actualUsedBytes += value
					case "slab_reclaimable":
						actualUsedBytes += value
					}
				}
			}
		case "system.memory.utilization":
			dataPoints := metric.Gauge().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}

				value := dp.DoubleValue()
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "free":
						usedPercent = 1 - value
					case "used":
						actualUsedPercent += value
					case "buffered":
						actualUsedPercent += value
					case "slab_unreclaimable":
						actualUsedPercent += value
					case "slab_reclaimable":
						actualUsedPercent += value
					}
				}
			}
		}
	}

	usedBytes += total
	actualFree = total - actualUsedBytes

	internal.AddMetrics(out, dataset, EmptyMutator,
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.memory.total",
			timestamp: timestamp,
			intValue:  &total,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.memory.free",
			timestamp: timestamp,
			intValue:  &free,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.memory.cached",
			timestamp: timestamp,
			intValue:  &cached,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.memory.used.bytes",
			timestamp: timestamp,
			intValue:  &usedBytes,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.memory.actual.used.bytes",
			timestamp: timestamp,
			intValue:  &actualUsedBytes,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.memory.actual.free",
			timestamp: timestamp,
			intValue:  &actualFree,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.memory.used.pct",
			timestamp:   timestamp,
			doubleValue: &usedPercent,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.memory.actual.used.pct",
			timestamp:   timestamp,
			doubleValue: &actualUsedPercent,
		},
	)

	return nil
}
