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

	remappers "github.com/elastic/opentelemetry-lib/remappers/internal"
)

// system.load.cores is calculated using the cpu remapper and if dataset
// based on system integration is enabled then the metric would end up in
// system.cpu dataset.
func remapLoadMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var l1, l5, l15 float64

	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		if metric.Name() == "system.cpu.load_average.1m" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			l1 = dp.DoubleValue()
		} else if metric.Name() == "system.cpu.load_average.5m" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			l5 = dp.DoubleValue()
		} else if metric.Name() == "system.cpu.load_average.15m" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			l15 = dp.DoubleValue()
		}
	}

	remappers.AddMetrics(out, dataset, remappers.EmptyMutator,
		remappers.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.load.1",
			Timestamp:   timestamp,
			DoubleValue: &l1,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.load.5",
			Timestamp:   timestamp,
			DoubleValue: &l5,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.load.15",
			Timestamp:   timestamp,
			DoubleValue: &l15,
		},
	)

	return nil
}
