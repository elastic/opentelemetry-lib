package hostmetrics

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

/*type dataType int

const (
	Gauge dataType = iota
	Sum
)*/

type metric struct {
	dataType       pmetric.MetricType
	name           string
	timestamp      pcommon.Timestamp
	startTimestamp pcommon.Timestamp
	intValue       *int64
	doubleValue    *float64
	attributes     *pcommon.Map
}

func addMetrics(ms pmetric.MetricSlice, resource pcommon.Resource, dataset string, metrics ...metric) {
	ms.EnsureCapacity(ms.Len() + len(metrics))

	for _, metric := range metrics {
		m := ms.AppendEmpty()
		m.SetName(metric.name)

		var dp pmetric.NumberDataPoint
		switch metric.dataType {
		case pmetric.MetricTypeGauge:
			dp = m.SetEmptyGauge().DataPoints().AppendEmpty()
		case pmetric.MetricTypeSum:
			dp = m.SetEmptySum().DataPoints().AppendEmpty()
		}

		if metric.intValue != nil {
			dp.SetIntValue(*metric.intValue)
		} else if metric.doubleValue != nil {
			dp.SetDoubleValue(*metric.doubleValue)
		}

		dp.SetTimestamp(metric.timestamp)
		if metric.startTimestamp != 0 {
			dp.SetStartTimestamp(metric.startTimestamp)
		}

		if metric.attributes != nil {
			metric.attributes.CopyTo(dp.Attributes())
		}
		if dataset == "system.process" {
			// Add resource attribute as an attribute to each datapoint.
			//TODO: Combine this the way we are setting attributes in network scraper
			addProcessAttributes(resource, dp)
		}
		dp.Attributes().PutStr("event.provider", "hostmetrics") // This attribute is added to all the OTELtranslated metrics.
		dp.Attributes().PutStr("data_stream.dataset", dataset)
	}
}
