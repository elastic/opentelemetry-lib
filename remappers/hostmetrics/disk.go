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

// remapDiskMetrics remaps disk-related metrics from the source to the output metric slice.
func remapDiskMetrics(src, out pmetric.MetricSlice, _ pcommon.Resource, dataset string) error {
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.disk.io", "system.disk.operations", "system.disk.pending_operations":
			remapDiskIntMetrics(metric, out, dataset, 1)
		case "system.disk.operation_time", "system.disk.io_time":
			remapDiskFloatMetrics(metric, out, dataset, 1000)
		}
	}
	return nil
}

// remapDiskIntMetrics processes integer-based disk metrics.
func remapDiskIntMetrics(metric pmetric.Metric, out pmetric.MetricSlice, dataset string, multiplier int64) {
	dataPoints := metric.Sum().DataPoints()
	for j := 0; j < dataPoints.Len(); j++ {
		dp := dataPoints.At(j)
		if device, ok := dp.Attributes().Get("device"); ok {
			timestamp := dp.Timestamp()
			value := dp.IntValue()
			direction, _ := dp.Attributes().Get("direction")
			addDiskMetric(out, dataset, metric.Name(), device.Str(), direction.Str(), timestamp, value, multiplier)
		} else {
			fmt.Printf("Missing 'device' attribute for metric: %s\n", metric.Name())
		}
	}
}

// processFloatMetrics processes float-based disk metrics.
func remapDiskFloatMetrics(metric pmetric.Metric, out pmetric.MetricSlice, dataset string, multiplier float64) {
	dataPoints := metric.Sum().DataPoints()
	for j := 0; j < dataPoints.Len(); j++ {
		dp := dataPoints.At(j)
		if device, ok := dp.Attributes().Get("device"); ok {
			timestamp := dp.Timestamp()
			value := dp.DoubleValue()
			direction, _ := dp.Attributes().Get("direction")
			addDiskMetric(out, dataset, metric.Name(), device.Str(), direction.Str(), timestamp, value, multiplier)
		} else {
			fmt.Printf("Missing 'device' attribute for metric: %s\n", metric.Name())
		}
	}
}

// addDiskMetric adds a remapped disk metric to the output slice.
func addDiskMetric[T constraints.Integer | constraints.Float](
	out pmetric.MetricSlice,
	dataset, name, device, direction string,
	timestamp pcommon.Timestamp,
	value T, multiplier T,
) {
	metricsToAdd := map[string]string{
		"system.disk.io":                 "system.diskio.%s.bytes",
		"system.disk.operations":         "system.diskio.%s.count",
		"system.disk.pending_operations": "system.diskio.io.%sops",
		"system.disk.operation_time":     "system.diskio.%s.time",
		"system.disk.io_time":            "system.diskio.io.%stime",
	}

	metricNetworkES, ok := metricsToAdd[name]
	if !ok {
		fmt.Printf("Unknown metric name: %s\n", name)
		return
	}

	scaledValue := value * multiplier
	var intValue *int64
	var doubleValue *float64
	switch v := any(scaledValue).(type) {
	case int64:
		intValue = &v
	case float64:
		doubleValue = &v
	default:
		fmt.Printf("Unsupported value type for metric: %s\n", name)
		return
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
