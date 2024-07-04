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
	"github.com/elastic/opentelemetry-lib/remappers/internal/remappedmetric"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func addClusterMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
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

	remappedmetric.AddMetrics(out, dataset, remappedmetric.EmptyMutator,
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeGauge,
			Name:      "kubernetes.node.cpu.allocatable.cores",
			Timestamp: timestamp,
			IntValue:  &node_allocatable_cpu,
		},
		remappedmetric.Metric{
			DataType:  pmetric.MetricTypeGauge,
			Name:      "kubernetes.node.memory.allocatable.bytes",
			Timestamp: timestamp,
			IntValue:  &node_allocatable_memory,
		},
	)

	return nil
}
