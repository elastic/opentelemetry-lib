package hostmetrics

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func addProcessMetrics(metrics pmetric.MetricSlice, resource pcommon.Resource, dataset string) error {
	var timestamp pcommon.Timestamp
	var startTime, processRuntime, threads, memUsage, memVirtual, fdOpen, ioReadBytes, ioWriteBytes, ioReadOperations, ioWriteOperations int64
	var memUtil, memUtilPct, total, cpuTimeValue, systemCpuTime, userCpuTime, cpuPct float64

	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)
		if metric.Name() == "process.threads" {
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			threads = dp.IntValue()
		} else if metric.Name() == "process.memory.utilization" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			memUtil = dp.DoubleValue()
		} else if metric.Name() == "process.memory.usage" {
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			memUsage = dp.IntValue()
		} else if metric.Name() == "process.memory.virtual" {
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			memVirtual = dp.IntValue()
		} else if metric.Name() == "process.open_file_descriptors" {
			dp := metric.Sum().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			if startTime == 0 {
				startTime = dp.StartTimestamp().AsTime().UnixMilli()
			}
			fdOpen = dp.IntValue()
		} else if metric.Name() == "process.cpu.time" {
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
		} else if metric.Name() == "process.disk.io" {
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
		} else if metric.Name() == "process.disk.operations" {
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

	addMetrics(metrics, resource, dataset,
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "process.cpu.start_time",
			timestamp: timestamp,
			intValue:  &startTime,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.num_threads",
			timestamp: timestamp,
			intValue:  &threads,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.process.memory.rss.pct",
			timestamp:   timestamp,
			doubleValue: &memUtilPct,
		},
		// The process rss bytes have been found to be equal to the memory usage reported by OTEL
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.memory.rss.bytes",
			timestamp: timestamp,
			intValue:  &memUsage,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.memory.size",
			timestamp: timestamp,
			intValue:  &memVirtual,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.fd.open",
			timestamp: timestamp,
			intValue:  &fdOpen,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "process.memory.pct",
			timestamp:   timestamp,
			doubleValue: &memUtilPct,
		},
		metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.total.value",
			timestamp:   timestamp,
			doubleValue: &cpuTimeValue,
		},
		metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.system.ticks",
			timestamp:   timestamp,
			doubleValue: &systemCpuTime,
		},
		metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.user.ticks",
			timestamp:   timestamp,
			doubleValue: &userCpuTime,
		},
		metric{
			dataType:    pmetric.MetricTypeSum,
			name:        "system.process.cpu.total.ticks",
			timestamp:   timestamp,
			doubleValue: &cpuTimeValue,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.read_bytes",
			timestamp: timestamp,
			intValue:  &ioReadBytes,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.write_bytes",
			timestamp: timestamp,
			intValue:  &ioWriteBytes,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.read_ops",
			timestamp: timestamp,
			intValue:  &ioReadOperations,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.io.write_ops",
			timestamp: timestamp,
			intValue:  &ioWriteOperations,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.process.cpu.total.pct",
			timestamp:   timestamp,
			doubleValue: &cpuPct,
		},
	)

	return nil
}

func addProcessAttributes(resource pcommon.Resource, dp pmetric.NumberDataPoint) {
	process_ppid, _ := resource.Attributes().Get("process.parent_pid")
	if process_ppid.Int() != 0 {
		dp.Attributes().PutInt("process.parent.pid", process_ppid.Int())
	}
	process_owner, _ := resource.Attributes().Get("process.owner")
	if process_owner.Str() != "" {
		dp.Attributes().PutStr("user.name", process_owner.Str())
	}
	process_executable, _ := resource.Attributes().Get("process.executable.path")
	if process_executable.Str() != "" {
		dp.Attributes().PutStr("process.executable", process_executable.Str())
	}
	process_name, _ := resource.Attributes().Get("process.executable.name")
	if process_name.Str() != "" {
		dp.Attributes().PutStr("process.name", process_name.Str())
	}
}
