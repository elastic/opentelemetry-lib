package hostmetrics

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func addLoadMetrics(metrics pmetric.MetricSlice, resource pcommon.Resource, dataset string) error {
	var timestamp pcommon.Timestamp
	var l1, l5, l15 float64

	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)
		if metric.Name() == "system.cpu.load_average.1m" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			l1 = dp.DoubleValue()
		} else if metric.Name() == "system.cpu.load_average.5m" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			l5 = dp.DoubleValue()
		} else if metric.Name() == "system.cpu.load_average.15m" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			l15 = dp.DoubleValue()
		}
	}

	addMetrics(metrics, resource, dataset,
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.load.1",
			timestamp:   timestamp,
			doubleValue: &l1,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.load.5",
			timestamp:   timestamp,
			doubleValue: &l5,
		},
		metric{
			dataType:    pmetric.MetricTypeGauge,
			name:        "system.load.15",
			timestamp:   timestamp,
			doubleValue: &l15,
		},
	)

	return nil
}
