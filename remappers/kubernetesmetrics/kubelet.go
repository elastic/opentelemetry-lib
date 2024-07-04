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

package kubernetesmetrics

import (
	"math"

	"github.com/elastic/opentelemetry-lib/remappers/internal/remappedmetric"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func addKubeletMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var total_transmited, total_received, node_memory_usage, filesystem_capacity, filesystem_usage int64
	var cpu_limit_utilization, memory_limit_utilization, node_cpu_usage, pod_cpu_usage_node, pod_memory_usage_node float64

	// iterate all metrics in the current scope and generate the additional Elastic kubernetes integration metrics
	//pod
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		// kubernetes.pod.memory.usage.node.pct still needs to be implemented
		if metric.Name() == "k8s.pod.cpu_limit_utilization" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			cpu_limit_utilization = dp.DoubleValue()
		} else if metric.Name() == "k8s.pod.cpu.node.utilization" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			pod_cpu_usage_node = dp.DoubleValue()
		} else if metric.Name() == "k8s.pod.memory_limit_utilization" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			memory_limit_utilization = dp.DoubleValue()
		} else if metric.Name() == "k8s.pod.network.io" {
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				if timestamp == 0 {
					timestamp = dp.Timestamp()
				}

				value := dp.IntValue()
				if direction, ok := dp.Attributes().Get("direction"); ok {
					switch direction.Str() {
					case "receive":
						total_received += value
					case "transmit":
						total_transmited += value
					}
				}
			}
			//node
		} else if metric.Name() == "k8s.node.cpu.usage" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			node_cpu_usage = dp.DoubleValue() * math.Pow10(9)
		} else if metric.Name() == "k8s.node.memory.usage" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			node_memory_usage = dp.IntValue()
		} else if metric.Name() == "k8s.node.filesystem.capacity" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			filesystem_capacity = dp.IntValue()
		} else if metric.Name() == "k8s.node.filesystem.usage" {
			dp := metric.Gauge().DataPoints().At(0)
			if timestamp == 0 {
				timestamp = dp.Timestamp()
			}
			filesystem_usage = dp.IntValue()
		}

	}

	remappedmetric.Add(out, dataset, func(dp pmetric.NumberDataPoint) {
		dp.Attributes().PutStr("service.type", "kubernetes")
	},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "kubernetes.pod.cpu.usage.limit.pct",
			Timestamp:   timestamp,
			DoubleValue: &cpu_limit_utilization,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "kubernetes.pod.cpu.usage.node.pct",
			Timestamp:   timestamp,
			DoubleValue: &pod_cpu_usage_node,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "kubernetes.pod.memory.usage.node.pct",
			Timestamp:   timestamp,
			DoubleValue: &pod_memory_usage_node,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "kubernetes.pod.memory.usage.limit.pct",
			Timestamp:   timestamp,
			DoubleValue: &memory_limit_utilization,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "kubernetes.pod.network.tx.bytes",
			Timestamp: timestamp,
			IntValue:  &total_transmited,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      "kubernetes.pod.network.rx.bytes",
			Timestamp: timestamp,
			IntValue:  &total_received,
		},
		remappedmetric.Metric{
			DataType:    pmetric.MetricTypeGauge,
			Name:        "kubernetes.node.cpu.usage.nanocores",
			Timestamp:   timestamp,
			DoubleValue: &node_cpu_usage,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeGauge,
			Name:      "kubernetes.node.memory.usage.bytes",
			Timestamp: timestamp,
			IntValue:  &node_memory_usage,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeGauge,
			Name:      "kubernetes.node.fs.capacity.bytes",
			Timestamp: timestamp,
			IntValue:  &filesystem_capacity,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeGauge,
			Name:      "kubernetes.node.fs.used.bytes",
			Timestamp: timestamp,
			IntValue:  &filesystem_usage,
		},
	)

	return nil
}
