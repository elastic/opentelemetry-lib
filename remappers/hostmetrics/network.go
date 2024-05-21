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
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var metricsMapping = map[string]string{
	"system.network.io":      "system.network.%s.bytes",
	"system.network.packets": "system.network.%s.packets",
	"system.network.dropped": "system.network.%s.dropped",
	"system.network.errors":  "system.network.%s.errors",
	"host.network.io":        "host.network.%s.bytes",
	"host.network.packets":   "host.network.%s.packets",
}

func remapNetworkMetrics(
	src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var (
		metricsetPeriod    int64
		metricsetTimestamp pcommon.Timestamp
	)
	for i := 0; i < src.Len(); i++ {
		m := src.At(i)

		elasticMetric, ok := metricsMapping[m.Name()]
		if !ok {
			continue
		}

		// Special case handling for host network metrics produced by aggregating
		// system network metrics and converting to delta temporality. These host
		// metrics have been aggregated to drop all labels other than direction.
		transformedHostMetrics := strings.HasPrefix(m.Name(), "host.")

		dataPoints := m.Sum().DataPoints()
		for j := 0; j < dataPoints.Len(); j++ {
			dp := dataPoints.At(j)

			device, ok := dp.Attributes().Get("device")
			if !ok && !transformedHostMetrics {
				continue
			}

			direction, ok := dp.Attributes().Get("direction")
			if !ok {
				continue
			}

			ts := dp.Timestamp()
			value := dp.IntValue()
			metricDataType := pmetric.MetricTypeSum
			if transformedHostMetrics {
				// transformed metrics are gauges as they are trasformed from cumulative to delta
				metricDataType = pmetric.MetricTypeGauge
				startTs := dp.StartTimestamp()
				if metricsetPeriod == 0 && startTs != 0 && startTs < ts {
					metricsetPeriod = int64(ts-startTs) / 1e6
					metricsetTimestamp = ts
				}
			}

			addMetrics(out, dataset,
				func(dp pmetric.NumberDataPoint) {
					if deviceStr := device.Str(); deviceStr != "" {
						dp.Attributes().PutStr("system.network.name", deviceStr)
					}
				},
				metric{
					dataType: metricDataType,
					name: fmt.Sprintf(
						elasticMetric,
						normalizeDirection(direction, transformedHostMetrics),
					),
					timestamp: ts,
					intValue:  &value,
				},
			)
		}
	}

	if metricsetPeriod > 0 {
		addMetrics(out, dataset, emptyMutator, metric{
			dataType:  pmetric.MetricTypeGauge,
			name:      "metricset.period",
			timestamp: metricsetTimestamp,
			intValue:  &metricsetPeriod,
		})
	}

	return nil
}

// normalize direction normalizes the OTel direction attribute to Elastic direction.
func normalizeDirection(dir pcommon.Value, hostMetrics bool) string {
	var in, out = "in", "out"
	if hostMetrics {
		in, out = "ingress", "egress"
	}
	switch dir.Str() {
	case "receive":
		return in
	case "transmit":
		return out
	}
	return ""
}
