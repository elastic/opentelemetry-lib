package hostmetrics

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func addProcessSummaryMetrics(metrics pmetric.MetricSlice, resource pcommon.Resource, dataset string) error {
	var timestamp pcommon.Timestamp
	var idleProcesses, sleepingProcesses, stoppedProcesses, zombieProcesses, totalProcesses int64

	// iterate all metrics in the current scope and generate the additional Elastic system integration metrics
	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)
		if metric.Name() == "system.processes.count" {
			dataPoints := metric.Sum().DataPoints()
			// iterate over the datapoints corresponding to different 'status' attributes
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

	addMetrics(metrics, resource, dataset,
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.idle",
			timestamp: timestamp,
			intValue:  &idleProcesses,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.sleeping",
			timestamp: timestamp,
			intValue:  &sleepingProcesses,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.stopped",
			timestamp: timestamp,
			intValue:  &stoppedProcesses,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.zombie",
			timestamp: timestamp,
			intValue:  &zombieProcesses,
		},
		metric{
			dataType:  pmetric.MetricTypeSum,
			name:      "system.process.summary.total",
			timestamp: timestamp,
			intValue:  &totalProcesses,
		},
	)
	return nil
}
