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
	"time"

	"github.com/elastic/opentelemetry-lib/remappers/internal/remappedmetric"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func remapProcessMetrics(
	src, out pmetric.MetricSlice,
	resource pcommon.Resource,
	dataset string,
) error {
	var timestamp, startTimestamp pcommon.Timestamp
	var threads, memUsage, memVirtual, fdOpen, ioReadBytes, ioWriteBytes, ioReadOperations, ioWriteOperations int64
	var memUtil, total, systemCpuTime, userCpuTime float64

	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "process.threads":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTimestamp == 0 {
				startTimestamp = dp.StartTimestamp()
			}
			threads = dp.IntValue()
		case "process.memory.utilization":
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTimestamp == 0 {
				startTimestamp = dp.StartTimestamp()
			}
			memUtil = dp.DoubleValue()
		case "process.memory.usage":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTimestamp == 0 {
				startTimestamp = dp.StartTimestamp()
			}
			memUsage = dp.IntValue()
		case "process.memory.virtual":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTimestamp == 0 {
				startTimestamp = dp.StartTimestamp()
			}
			memVirtual = dp.IntValue()
		case "process.open_file_descriptors":
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTimestamp == 0 {
				startTimestamp = dp.StartTimestamp()
			}
			fdOpen = dp.IntValue()
		case "process.cpu.time":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}
				if startTimestamp == 0 {
					startTimestamp = dp.StartTimestamp()
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
				if startTimestamp == 0 {
					startTimestamp = dp.StartTimestamp()
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
				if startTimestamp == 0 {
					startTimestamp = dp.StartTimestamp()
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

	systemCpuTime = systemCpuTime * 1000
	userCpuTime = userCpuTime * 1000

	startTime := startTimestamp.AsTime()
	startTimeMillis := startTime.UnixMilli()
	memUtilPct := memUtil / 100
	cpuTimeValue := total * 1000
	processRuntime := timestamp.AsTime().UnixMilli() - startTimeMillis
	cpuPct := cpuTimeValue / float64(processRuntime)

	remappedmetric.Add(out, dataset, addProcessResources(resource, startTime.UTC()),
		// The timestamp metrics get converted from Int to Timestamp in Kibana
		// since these are mapped to timestamp datatype
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "process.cpu.start_time",
			Timestamp: timestamp,
			IntValue:  &startTimeMillis,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.num_threads",
			Timestamp: timestamp,
			IntValue:  &threads,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.process.memory.rss.pct",
			Timestamp:   timestamp,
			DoubleValue: &memUtilPct,
		},
		// The process rss bytes have been found to be equal to the memory usage reported by OTEL
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.memory.rss.bytes",
			Timestamp: timestamp,
			IntValue:  &memUsage,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.memory.size",
			Timestamp: timestamp,
			IntValue:  &memVirtual,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.fd.open",
			Timestamp: timestamp,
			IntValue:  &fdOpen,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "process.memory.pct",
			Timestamp:   timestamp,
			DoubleValue: &memUtilPct,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.total.value",
			Timestamp:   timestamp,
			DoubleValue: &cpuTimeValue,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.system.ticks",
			Timestamp:   timestamp,
			DoubleValue: &systemCpuTime,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.user.ticks",
			Timestamp:   timestamp,
			DoubleValue: &userCpuTime,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        "system.process.cpu.total.ticks",
			Timestamp:   timestamp,
			DoubleValue: &cpuTimeValue,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.read_bytes",
			Timestamp: timestamp,
			IntValue:  &ioReadBytes,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.write_bytes",
			Timestamp: timestamp,
			IntValue:  &ioWriteBytes,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.read_ops",
			Timestamp: timestamp,
			IntValue:  &ioReadOperations,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "system.process.io.write_ops",
			Timestamp: timestamp,
			IntValue:  &ioWriteOperations,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "system.process.cpu.total.pct",
			Timestamp:   timestamp,
			DoubleValue: &cpuPct,
		},
	)

	return nil
}

func addProcessResources(resource pcommon.Resource, startTime time.Time) func(pmetric.NumberDataPoint) {
	startTimeStr := startTime.Format(time.RFC3339)
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
		dp.Attributes().PutStr("system.process.cpu.start_time", startTimeStr)
		// Adding dummy value to process.state as "undefined", since this field is not
		// available through hostmetrics receiver currently and Process tab in curated
		// UI's need this field as a prerequisite.
		dp.Attributes().PutStr("system.process.state", "undefined")
	}
}
