package hostmetrics

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func addNetworkMetrics(metrics pmetric.MetricSlice, resource pcommon.Resource, dataset string) error {
	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)
		dataPoints := metric.Sum().DataPoints()
		for j := 0; j < dataPoints.Len(); j++ {
			dp := dataPoints.At(j)

			var device string
			if d, ok := dp.Attributes().Get("device"); ok {
				device = d.Str()
			} else {
				continue
			}

			if direction, ok := dp.Attributes().Get("direction"); ok {
				name := metric.Name()
				timestamp := dp.Timestamp()
				value := dp.IntValue()

				switch direction.Str() {
				case "receive":
					addDeviceMetric(metrics, resource, dataset, name, device, "in", timestamp, value)
				case "transmit":
					addDeviceMetric(metrics, resource, dataset, name, device, "out", timestamp, value)
				}
			}
		}
	}

	return nil
}

func addDeviceMetric(metrics pmetric.MetricSlice, resource pcommon.Resource,
	dataset, name, device, direction string, timestamp pcommon.Timestamp, value int64) {

	metricsToAdd := map[string]string{
		"system.network.io":      "system.network.%s.bytes",
		"system.network.packets": "system.network.%s.packets",
		"system.network.dropped": "system.network.%s.dropped",
		"system.network.errors":  "system.network.%s.errors",
	}

	if metricNetworkES, ok := metricsToAdd[name]; ok {
		attributes := pcommon.NewMap()
		attributes.PutStr("system.network.name", device)

		addMetrics(metrics, resource, dataset,
			metric{
				dataType:   pmetric.MetricTypeSum,
				name:       fmt.Sprintf(metricNetworkES, direction),
				timestamp:  timestamp,
				intValue:   &value,
				attributes: &attributes,
			})
	}
}
