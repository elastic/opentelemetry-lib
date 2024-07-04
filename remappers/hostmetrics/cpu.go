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

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/elastic/opentelemetry-lib/remappers/internal/remappedmetric"
)

func remapCPUMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var numCores int64
	var totalPercent, idlePercent, systemPercent, userPercent, stealPercent,
		iowaitPercent, nicePercent, irqPercent, softirqPercent float64

	// iterate all metrics in the current scope and generate the additional Elastic
	// system integration metrics.
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.cpu.logical.count":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			numCores = dp.IntValue()
		case "system.cpu.utilization":
			dataPoints := metric.Gauge().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}

				value := dp.DoubleValue()
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "idle":
						idlePercent += value
					case "system":
						systemPercent += value
						totalPercent += value
					case "user":
						userPercent += value
						totalPercent += value
					case "steal":
						stealPercent += value
						totalPercent += value
					case "wait":
						iowaitPercent += value
						totalPercent += value
					case "nice":
						nicePercent += value
						totalPercent += value
					case "interrupt":
						irqPercent += value
						totalPercent += value
					case "softirq":
						softirqPercent += value
						totalPercent += value
					}
				}
			}
		}
	}

	// Add all metrics that are independent of cpu logical count.
	remappedmetric.AddMetrics(out, dataset, remappedmetric.EmptyMutator,
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.total.pct",
			Timestamp:   timestamp,
			DoubleValue: &totalPercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.idle.pct",
			Timestamp:   timestamp,
			DoubleValue: &idlePercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.system.pct",
			Timestamp:   timestamp,
			DoubleValue: &systemPercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.user.pct",
			Timestamp:   timestamp,
			DoubleValue: &userPercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.steal.pct",
			Timestamp:   timestamp,
			DoubleValue: &stealPercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.iowait.pct",
			Timestamp:   timestamp,
			DoubleValue: &iowaitPercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.nice.pct",
			Timestamp:   timestamp,
			DoubleValue: &nicePercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.irq.pct",
			Timestamp:   timestamp,
			DoubleValue: &irqPercent,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.softirq.pct",
			Timestamp:   timestamp,
			DoubleValue: &softirqPercent,
		},
	)

	if numCores == 0 {
		return errors.New("system.cpu.logical.count metric is missing in the hostmetrics")
	}

	totalNorm := totalPercent / float64(numCores)
	idleNorm := idlePercent / float64(numCores)
	systemNorm := systemPercent / float64(numCores)
	userNorm := userPercent / float64(numCores)
	stealNorm := stealPercent / float64(numCores)
	iowaitNorm := iowaitPercent / float64(numCores)
	niceNorm := nicePercent / float64(numCores)
	irqNorm := irqPercent / float64(numCores)
	softirqNorm := softirqPercent / float64(numCores)

	remappedmetric.AddMetrics(out, dataset, remappedmetric.EmptyMutator,
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.cpu.cores",
			Timestamp: timestamp,
			IntValue:  &numCores,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.load.cores",
			Timestamp: timestamp,
			IntValue:  &numCores,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.total.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &totalNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.idle.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &idleNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.system.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &systemNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.user.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &userNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.steal.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &stealNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.iowait.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &iowaitNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.nice.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &niceNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.irq.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &irqNorm,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.cpu.softirq.norm.pct",
			Timestamp:   timestamp,
			DoubleValue: &softirqNorm,
		},
	)

	return nil
}
