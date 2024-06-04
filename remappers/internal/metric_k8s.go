package internal

import (
	"strings"

	"github.com/elastic/opentelemetry-lib/remappers/common"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func Addk8sMetrics(ms pmetric.MetricSlice, mutator func(dp pmetric.NumberDataPoint), metrics ...Metric) {
	ms.EnsureCapacity(ms.Len() + len(metrics))

	for _, metric := range metrics {
		m := ms.AppendEmpty()
		m.SetName(metric.Name)

		var dp pmetric.NumberDataPoint
		switch metric.DataType {
		case pmetric.MetricTypeGauge:
			dp = m.SetEmptyGauge().DataPoints().AppendEmpty()
		case pmetric.MetricTypeSum:
			dp = m.SetEmptySum().DataPoints().AppendEmpty()
		}

		if metric.IntValue != nil {
			dp.SetIntValue(*metric.IntValue)
		} else if metric.DoubleValue != nil {
			dp.SetDoubleValue(*metric.DoubleValue)
		}

		dp.SetTimestamp(metric.Timestamp)
		if metric.StartTimestamp != 0 {
			dp.SetStartTimestamp(metric.StartTimestamp)
		}

		// Calculate datastream attribute as an attribute to each datapoint
		dataset := AddDatastream(metric.Name)
		if dataset == "kubernetes.node" {
			dataset = "kubernetes.pod"
		}
		dp.Attributes().PutStr("event.module", common.RemapperEventModule)
		dp.Attributes().PutStr("service.type", "kubernetes")
		dp.Attributes().PutStr(common.DatastreamDatasetLabel, dataset)

		mutator(dp)
	}
}

// AddDatastream calculates the datastream_dataset name of the metric based on the name of the metric provided
func AddDatastream(name string) string {
	splitted_metric := strings.Split(name, ".")
	datastream := ""
	prefix := "kubernetes"

	if splitted_metric[0] == "kubernetes" {
		datastream = splitted_metric[1]
	}

	return prefix + "." + datastream
}
