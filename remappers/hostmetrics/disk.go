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
	"errors"
	"fmt"

	remappers "github.com/elastic/opentelemetry-lib/remappers/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var metricsToAdd = map[string]string{
	"system.disk.io":                 "system.diskio.%s.bytes",
	"system.disk.operations":         "system.diskio.%s.count",
	"system.disk.pending_operations": "system.diskio.io.%sops",
	"system.disk.operation_time":     "system.diskio.%s.time",
	"system.disk.io_time":            "system.diskio.io.%stime",
}

// remapDiskMetrics remaps disk-related metrics from the source to the output metric slice.
func remapDiskMetrics(src, out pmetric.MetricSlice, _ pcommon.Resource, dataset string) error {
	var errs []error
	for i := 0; i < src.Len(); i++ {
		var err error
		metric := src.At(i)
		switch metric.Name() {
		case "system.disk.io", "system.disk.operations", "system.disk.pending_operations":
			err = addDiskMetric(metric, out, dataset, 1)
		case "system.disk.operation_time", "system.disk.io_time":
			err = addDiskMetric(metric, out, dataset, 1000)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func addDiskMetric(metric pmetric.Metric, out pmetric.MetricSlice, dataset string, multiplier int64) error {
	metricNetworkES, ok := metricsToAdd[metric.Name()]
	if !ok {
		return fmt.Errorf("unexpected metric name: %s", metric.Name())
	}

	dps := metric.Sum().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if device, ok := dp.Attributes().Get("device"); ok {
			direction, _ := dp.Attributes().Get("direction")
			remappedMetric := remappers.Metric{
				DataType:  pmetric.MetricTypeSum,
				Name:      fmt.Sprintf(metricNetworkES, direction.Str()),
				Timestamp: dp.Timestamp(),
			}
			switch dp.ValueType() {
			case pmetric.NumberDataPointValueTypeInt:
				v := dp.IntValue() * multiplier
				remappedMetric.IntValue = &v
			case pmetric.NumberDataPointValueTypeDouble:
				v := dp.DoubleValue() * float64(multiplier)
				remappedMetric.DoubleValue = &v
			}
			remappers.AddMetrics(out, dataset, func(dp pmetric.NumberDataPoint) {
				dp.Attributes().PutStr("system.diskio.name", device.Str())
			}, remappedMetric)
		}
	}
	return nil
}
