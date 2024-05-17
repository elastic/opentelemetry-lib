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
	addMetrics(out, dataset, emptyMutator,
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.total.pct",
			timestamp:   timestamp,
			doubleValue: &totalPercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.idle.pct",
			timestamp:   timestamp,
			doubleValue: &idlePercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.system.pct",
			timestamp:   timestamp,
			doubleValue: &systemPercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.user.pct",
			timestamp:   timestamp,
			doubleValue: &userPercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.steal.pct",
			timestamp:   timestamp,
			doubleValue: &stealPercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.wait.pct",
			timestamp:   timestamp,
			doubleValue: &iowaitPercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.nice.pct",
			timestamp:   timestamp,
			doubleValue: &nicePercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.irq.pct",
			timestamp:   timestamp,
			doubleValue: &irqPercent,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.softirq.pct",
			timestamp:   timestamp,
			doubleValue: &softirqPercent,
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

	addMetrics(out, dataset, emptyMutator,
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.cpu.cores",
			timestamp: timestamp,
			intValue:  &numCores,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.load.cores",
			timestamp: timestamp,
			intValue:  &numCores,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.total.norm.pct",
			timestamp:   timestamp,
			doubleValue: &totalNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.idle.norm.pct",
			timestamp:   timestamp,
			doubleValue: &idleNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.system.norm.pct",
			timestamp:   timestamp,
			doubleValue: &systemNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.user.norm.pct",
			timestamp:   timestamp,
			doubleValue: &userNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.steal.norm.pct",
			timestamp:   timestamp,
			doubleValue: &stealNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.wait.norm.pct",
			timestamp:   timestamp,
			doubleValue: &iowaitNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.nice.norm.pct",
			timestamp:   timestamp,
			doubleValue: &niceNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.irq.norm.pct",
			timestamp:   timestamp,
			doubleValue: &irqNorm,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.cpu.softirq.norm.pct",
			timestamp:   timestamp,
			doubleValue: &softirqNorm,
		},
	)

	return nil
}
