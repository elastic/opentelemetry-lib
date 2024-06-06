package kubernetesmetrics

import (
	remappers "github.com/elastic/opentelemetry-lib/remappers/internal"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func addClusterMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
) error {
	var timestamp pcommon.Timestamp
	var node_allocatable_memory, node_allocatable_cpu int64

	// iterate all metrics in the current scope and generate the additional Elastic kubernetes integration metrics
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		if metric.Name() == "k8s.node.allocatable_cpu" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			node_allocatable_cpu = dp.IntValue()
		} else if metric.Name() == "k8s.node.allocatable_memory" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			node_allocatable_memory = dp.IntValue()
		}
	}

	remappers.Addk8sMetrics(out, remappers.EmptyMutator,
		remappers.Metric{
			DataType:  pmetric.MetricTypeGauge,
			Name:      "kubernetes.node.cpu.allocatable.cores",
			Timestamp: timestamp,
			IntValue:  &node_allocatable_cpu,
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeGauge,
			Name:      "kubernetes.node.memory.allocatable.bytes",
			Timestamp: timestamp,
			IntValue:  &node_allocatable_memory,
		},
	)

	return nil
}
