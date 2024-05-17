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

func remapProcessesMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var idleProcesses, sleepingProcesses, stoppedProcesses, zombieProcesses, totalProcesses int64

	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		if metric.Name() == "system.processes.count" {
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}
				value := dp.IntValue()
				if status, ok := dp.Attributes().Get("status"); ok {
					switch status.Str() {
					case "idle":
						idleProcesses = value
						totalProcesses += value
					case "sleeping":
						sleepingProcesses = value
						totalProcesses += value
					case "stopped":
						stoppedProcesses = value
						totalProcesses += value
					case "zombies":
						zombieProcesses = value
						totalProcesses += value
					}
				}
			}
		}

	}

	internal.AddMetrics(out, dataset, EmptyMutator,
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.idle",
			timestamp: timestamp,
			intValue:  &idleProcesses,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.sleeping",
			timestamp: timestamp,
			intValue:  &sleepingProcesses,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.stopped",
			timestamp: timestamp,
			intValue:  &stoppedProcesses,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.zombie",
			timestamp: timestamp,
			intValue:  &zombieProcesses,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.total",
			timestamp: timestamp,
			intValue:  &totalProcesses,
		},
	)
	return nil
}
