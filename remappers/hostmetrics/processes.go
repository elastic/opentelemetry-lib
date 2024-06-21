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

func remapProcessesMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var idleProcesses, sleepingProcesses, stoppedProcesses, zombieProcesses, runningProcesses, totalProcesses int64

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
					case "running":
						runningProcesses = value
						totalProcesses += value
					default:
						totalProcesses += value
					}
				}
			}
		}

	}

	remappers.AddMetrics(out, dataset, remappers.EmptyMutator,
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.summary.idle",
			Timestamp: timestamp,
			IntValue:  &idleProcesses,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.summary.sleeping",
			Timestamp: timestamp,
			IntValue:  &sleepingProcesses,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.summary.stopped",
			Timestamp: timestamp,
			IntValue:  &stoppedProcesses,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.summary.zombie",
			Timestamp: timestamp,
			IntValue:  &zombieProcesses,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.summary.running",
			Timestamp: timestamp,
			IntValue:  &runningProcesses,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.summary.total",
			Timestamp: timestamp,
			IntValue:  &totalProcesses,
		},
	)
	return nil
}
