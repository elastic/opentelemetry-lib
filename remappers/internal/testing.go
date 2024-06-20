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

package internal

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type TestMetric struct {
	DP   TestDP
	Name string
	Type pmetric.MetricType
}

type TestDP struct {
	Dbl   *float64
	Int   *int64
	Attrs map[string]any
	Ts    pcommon.Timestamp
}

func MetricSliceToTestMetric(t *testing.T, ms pmetric.MetricSlice) []TestMetric {
	testMetrics := make([]TestMetric, ms.Len())
	for i := 0; i < ms.Len(); i++ {
		m := ms.At(i)
		testMetrics[i].Name = m.Name()
		testMetrics[i].Type = m.Type()

		var dps pmetric.NumberDataPointSlice
		switch m.Type() {
		case pmetric.MetricTypeGauge:
			dps = m.Gauge().DataPoints()
		case pmetric.MetricTypeSum:
			dps = m.Sum().DataPoints()
		}

		if dps.Len() != 1 {
			t.Fatalf("unexpected metric, test is written assuming each metric with a single datapoint")
		}

		dp := dps.At(0)
		testMetrics[i].DP = TestDP{Ts: dp.Timestamp(), Attrs: dp.Attributes().AsRaw()}
		switch dp.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			testMetrics[i].DP.Int = Ptr(dp.IntValue())
		case pmetric.NumberDataPointValueTypeDouble:
			testMetrics[i].DP.Dbl = Ptr(dp.DoubleValue())
		}
	}

	return testMetrics
}

func TestMetricToMetricSlice(t testing.TB, testMetrics []TestMetric, out pmetric.MetricSlice) {
	out.EnsureCapacity(len(testMetrics))

	for _, testm := range testMetrics {
		m := out.AppendEmpty()
		m.SetName(testm.Name)

		var dps pmetric.NumberDataPointSlice
		switch typ := testm.Type; typ {
		case pmetric.MetricTypeGauge:
			dps = m.SetEmptyGauge().DataPoints()
		case pmetric.MetricTypeSum:
			dps = m.SetEmptySum().DataPoints()
		default:
			t.Fatalf("unhandled metric type %s", typ)
		}

		dp := dps.AppendEmpty()
		dp.SetTimestamp(testm.DP.Ts)
		if testm.DP.Int != nil {
			dp.SetIntValue(*testm.DP.Int)
		} else if testm.DP.Dbl != nil {
			dp.SetDoubleValue(*testm.DP.Dbl)
		}
		if err := dp.Attributes().FromRaw(testm.DP.Attrs); err != nil {
			t.Fatalf("failed to copy attributes from test data: %v", err)
		}
	}
}

func Ptr[T any](v T) *T {
	return &v
}
