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

func remapProcessMetrics(
	src, out pmetric.MetricSlice,
	resource pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var startTime, processRuntime, threads, memUsage, memVirtual, fdOpen, ioReadBytes, ioWriteBytes, ioReadOperations, ioWriteOperations int64
	var memUtil, memUtilPct, total, cpuTimeValue, systemCpuTime, userCpuTime, cpuPct float64

	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "process.threads":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			threads = dp.IntValue()
		case "process.memory.utilization":
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			memUtil = dp.DoubleValue()
		case "process.memory.usage":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			memUsage = dp.IntValue()
		case "process.memory.virtual":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			memVirtual = dp.IntValue()
		case "process.open_file_descriptors":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			fdOpen = dp.IntValue()
		case "process.cpu.time":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}
				if startTime == 0 {
					startTime = dp.StartTimestamp().AsTime().UnixMilli()
				}
				value := dp.DoubleValue()
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "system":
						systemCpuTime = value
						total += value
					case "user":
						userCpuTime = value
						total += value
					case "wait":
						total += value

					}
				}
			}
		case "process.disk.io":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}
				if startTime == 0 {
					startTime = dp.StartTimestamp().AsTime().UnixMilli()
				}
				value := dp.IntValue()
				if direction, ok := dp.Attributes().Get("direction"); ok {
					switch direction.Str() {
					case "read":
						ioReadBytes = value
					case "write":
						ioWriteBytes = value
					}
				}
			}
		case "process.disk.operations":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}
				if startTime == 0 {
					startTime = dp.StartTimestamp().AsTime().UnixMilli()
				}
				value := dp.IntValue()
				if direction, ok := dp.Attributes().Get("direction"); ok {
					switch direction.Str() {
					case "read":
						ioReadOperations = value
					case "write":
						ioWriteOperations = value
					}
				}
			}
		}
	}

	memUtilPct = memUtil / 100
	cpuTimeValue = total * 1000
	systemCpuTime = systemCpuTime * 1000
	userCpuTime = userCpuTime * 1000
	processRuntime = timestamp.AsTime().UnixMilli() - startTime
	cpuPct = cpuTimeValue / float64(processRuntime)

	internal.AddMetrics(out, dataset, addProcessResources(resource),
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "process.cpu.start_time",
			timestamp: timestamp,
			intValue:  &startTime,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.num_threads",
			timestamp: timestamp,
			intValue:  &threads,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.process.memory.rss.pct",
			timestamp:   timestamp,
			doubleValue: &memUtilPct,
		},
		// The process rss bytes have been found to be equal to the memory usage reported by OTEL
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.memory.rss.bytes",
			timestamp: timestamp,
			intValue:  &memUsage,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.memory.size",
			timestamp: timestamp,
			intValue:  &memVirtual,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.fd.open",
			timestamp: timestamp,
			intValue:  &fdOpen,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "process.memory.pct",
			timestamp:   timestamp,
			doubleValue: &memUtilPct,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.total.value",
			timestamp:   timestamp,
			doubleValue: &cpuTimeValue,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.system.ticks",
			timestamp:   timestamp,
			doubleValue: &systemCpuTime,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.user.ticks",
			timestamp:   timestamp,
			doubleValue: &userCpuTime,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.total.ticks",
			timestamp:   timestamp,
			doubleValue: &cpuTimeValue,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.read_bytes",
			timestamp: timestamp,
			intValue:  &ioReadBytes,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.write_bytes",
			timestamp: timestamp,
			intValue:  &ioWriteBytes,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.read_ops",
			timestamp: timestamp,
			intValue:  &ioReadOperations,
		},
		internal.Metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.write_ops",
			timestamp: timestamp,
			intValue:  &ioWriteOperations,
		},
		internal.Metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.process.cpu.total.pct",
			timestamp:   timestamp,
			doubleValue: &cpuPct,
		},
	)

	return nil
}

func addProcessResources(resource pcommon.Resource) func(pmetric.NumberDataPoint) {
	return func(dp pmetric.NumberDataPoint) {
		ppid, _ := resource.Attributes().Get("process.parent_pid")
		if ppid.Int() != 0 {
			dp.Attributes().PutInt("process.parent.pid", ppid.Int())
		}
		owner, _ := resource.Attributes().Get("process.owner")
		if owner.Str() != "" {
			dp.Attributes().PutStr("user.name", owner.Str())
		}
		exec, _ := resource.Attributes().Get("process.executable.path")
		if exec.Str() != "" {
			dp.Attributes().PutStr("process.executable", exec.Str())
		}
		name, _ := resource.Attributes().Get("process.executable.name")
		if name.Str() != "" {
			dp.Attributes().PutStr("process.name", name.Str())
		}
	}
}
