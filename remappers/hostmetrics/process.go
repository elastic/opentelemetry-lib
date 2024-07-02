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

	remappers.AddMetrics(out, dataset, addProcessResources(resource),
		// The timestamp metrics get converted from Int to Timestamp in Kibana
		// since these are mapped to timestamp datatype
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "process.cpu.start_time",
			Timestamp: timestamp,
			IntValue:  &startTime,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.cpu.start_time",
			Timestamp: timestamp,
			IntValue:  &startTime,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.num_threads",
			Timestamp: timestamp,
			IntValue:  &threads,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.process.memory.rss.pct",
			Timestamp:   timestamp,
			DoubleValue: &memUtilPct,
		},
		// The process rss bytes have been found to be equal to the memory usage reported by OTEL
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.memory.rss.bytes",
			Timestamp: timestamp,
			IntValue:  &memUsage,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.memory.size",
			Timestamp: timestamp,
			IntValue:  &memVirtual,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.fd.open",
			Timestamp: timestamp,
			IntValue:  &fdOpen,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "process.memory.pct",
			Timestamp:   timestamp,
			DoubleValue: &memUtilPct,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.total.value",
			Timestamp:   timestamp,
			DoubleValue: &cpuTimeValue,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.system.ticks",
			Timestamp:   timestamp,
			DoubleValue: &systemCpuTime,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.user.ticks",
			Timestamp:   timestamp,
			DoubleValue: &userCpuTime,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.total.ticks",
			Timestamp:   timestamp,
			DoubleValue: &cpuTimeValue,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.read_bytes",
			Timestamp: timestamp,
			IntValue:  &ioReadBytes,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.write_bytes",
			Timestamp: timestamp,
			IntValue:  &ioWriteBytes,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.read_ops",
			Timestamp: timestamp,
			IntValue:  &ioReadOperations,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.write_ops",
			Timestamp: timestamp,
			IntValue:  &ioWriteOperations,
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.process.cpu.total.pct",
			Timestamp:   timestamp,
			DoubleValue: &cpuPct,
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
		cmdline, _ := resource.Attributes().Get("process.command_line")
		if cmdline.Str() != "" {
			dp.Attributes().PutStr("system.process.cmdline", cmdline.Str())
		}
		//Adding dummy value to process.state as "Not Known", since this field is not
		//available through Hostmetrics currently and Process tab in Curated UI's needs this field as a prerequisite to work
		dp.Attributes().PutStr("system.process.state", "Not Known")
	}
}
