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

	remappers "github.com/elastic/opentelemetry-lib/remappers/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"golang.org/x/exp/constraints"
)

func remapDiskMetrics(src, out pmetric.MetricSlice, _ pcommon.Resource, dataset string) error {
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.disk.io", "system.disk.operations":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)

				device, ok := dp.Attributes().Get("device")
				if !ok {
					continue
				}

				var multiplier int64 = 1
				if direction, ok := dp.Attributes().Get("direction"); ok {
					name := metric.Name()
					timestamp := dp.Timestamp()
					value := dp.IntValue()
					addDiskMetric(out, dataset, name, device.Str(), direction.Str(), timestamp, value, multiplier)
				}
			}
		case "system.disk.operation_time":
			var multiplier float64
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				timestamp := dp.Timestamp()
				value := dp.DoubleValue()
				multiplier = 1000 // Elastic saves this value in milliseconds

				device, ok := dp.Attributes().Get("device")
				if !ok {
					continue
				}

				if direction, ok := dp.Attributes().Get("direction"); ok {
					addDiskMetric(out, dataset, metric.Name(), device.Str(), direction.Str(), timestamp, value, multiplier)
				}
			}
		case "system.disk.io_time":
			var multiplier float64
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				timestamp := dp.Timestamp()
				value := dp.DoubleValue()
				multiplier = 1000 // Elastic saves this value in milliseconds

				device, ok := dp.Attributes().Get("device")
				if !ok {
					continue
				}
				addDiskMetric(out, dataset, metric.Name(), device.Str(), "", timestamp, value, multiplier)
			}
		case "system.disk.pending_operations":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				timestamp := dp.Timestamp()
				value := dp.IntValue()

				device, ok := dp.Attributes().Get("device")
				if !ok {
					continue
				}

				var multiplier int64 = 1
				addDiskMetric(out, dataset, metric.Name(), device.Str(), "", timestamp, value, multiplier)
			}
		}
	}
	return nil
}

func addDiskMetric[T interface {
	constraints.Integer | constraints.Float
}](out pmetric.MetricSlice,
	dataset, name, device, direction string,
	timestamp pcommon.Timestamp,
	value, multiplier T) {

	metricsToAdd := map[string]string{
		"system.disk.io":                 "system.diskio.%s.bytes",
		"system.disk.operations":         "system.diskio.%s.count",
		"system.disk.pending_operations": "system.diskio.io.%sops",
		"system.disk.operation_time":     "system.diskio.%s.time",
		"system.disk.io_time":            "system.diskio.io.%stime",
	}

	metricNetworkES, ok := metricsToAdd[name]
	if !ok {
		return
	}

	var intValue *int64
	var doubleValue *float64
	scaledValue := value * multiplier
	if i, ok := any(scaledValue).(int64); ok {
		intValue = &i
	} else if d, ok := any(scaledValue).(float64); ok {
		doubleValue = &d
	}

	remappers.AddMetrics(out, dataset, func(dp pmetric.NumberDataPoint) {
		dp.Attributes().PutStr("system.diskio.name", device)
	},
		remappers.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        fmt.Sprintf(metricNetworkES, direction),
			Timestamp:   timestamp,
			IntValue:    intValue,
			DoubleValue: doubleValue,
		})
}
