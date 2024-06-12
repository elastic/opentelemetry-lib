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
	"fmt"
	"testing"
	"time"

	"github.com/elastic/opentelemetry-lib/remappers/common"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zaptest"
)

var (
	Sum         = pmetric.MetricTypeSum
	Gauge       = pmetric.MetricTypeGauge
	scopePrefix = "otelcol/kubeletstatsreceiver"
	// Test values to make assertion easier
	POD              = "pod0"
	NAMESPACE        = "kube-system"
	DEVICE    string = "eth0"
)

func TestRemap(t *testing.T) {
	doTestRemap(t, "without_k8s_integration", WithKubernetesIntegrationDataset(false))
	doTestRemap(t, "with_k8s_integration", WithKubernetesIntegrationDataset(true))
}

func doTestRemap(t *testing.T, id string, remapOpts ...Option) {
	t.Helper()

	outAttr := func() map[string]any {
		m := map[string]any{"event.module": "elastic/opentelemetry-lib"}
		m["service.type"] = "kubernetes"
		m[common.DatastreamDatasetLabel] = "kubernetes.pod"
		return m
	}
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, tc := range []struct {
		name          string
		scraper       string
		resourceAttrs map[string]any
		input         []testMetric
		expected      []testMetric
	}{
		{
			name:    "kubeletstats",
			scraper: "kubeletstatsreceiver",
			resourceAttrs: map[string]any{
				"k8s.pod.name":       POD,
				"k8s.namespace.name": NAMESPACE,
				"k8s.device":         DEVICE,
			},
			input: []testMetric{
				{Type: Gauge, Name: "k8s.pod.cpu_limit_utilization", DP: testDP{Ts: now, Dbl: ptr(0.26), Attrs: map[string]any{"k8s.pod.name": POD, "k8s.namespace.name": NAMESPACE}}},
				{Type: Gauge, Name: "k8s.pod.memory_limit_utilization", DP: testDP{Ts: now, Dbl: ptr(0.18), Attrs: map[string]any{"k8s.pod.name": "pod0", "k8s.pod.namespace": NAMESPACE}}},
				{Type: Sum, Name: "k8s.pod.network.io", DP: testDP{Ts: now, Int: ptr(int64(1024)), Attrs: map[string]any{"device": DEVICE, "direction": "receive"}}},
				{Type: Sum, Name: "k8s.pod.network.io", DP: testDP{Ts: now, Int: ptr(int64(2048)), Attrs: map[string]any{"device": DEVICE, "direction": "transmit"}}},
			},
			expected: []testMetric{
				{Type: Gauge, Name: "kubernetes.pod.cpu.usage.limit.pct", DP: testDP{Ts: now, Dbl: ptr(0.26), Attrs: outAttr()}},
				{Type: Gauge, Name: "kubernetes.pod.cpu.usage.node.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr()}},
				{Type: Gauge, Name: "kubernetes.pod.memory.usage.node.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr()}},
				{Type: Gauge, Name: "kubernetes.pod.memory.usage.limit.pct", DP: testDP{Ts: now, Dbl: ptr(0.18), Attrs: outAttr()}},
				{Type: Sum, Name: "kubernetes.pod.network.tx.bytes", DP: testDP{Ts: now, Int: ptr(int64(2048)), Attrs: outAttr()}},
				{Type: Sum, Name: "kubernetes.pod.network.rx.bytes", DP: testDP{Ts: now, Int: ptr(int64(1024)), Attrs: outAttr()}},
				{Type: Gauge, Name: "kubernetes.node.cpu.usage.nanocores", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr()}},
				{Type: Gauge, Name: "kubernetes.node.memory.usage.bytes", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr()}},
				{Type: Gauge, Name: "kubernetes.node.fs.capacity.bytes", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr()}},
				{Type: Gauge, Name: "kubernetes.node.fs.used.bytes", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr()}},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.name, id), func(t *testing.T) {
			sm := pmetric.NewScopeMetrics()
			sm.Scope().SetName(fmt.Sprintf("%s/%s", scopePrefix, tc.scraper))
			testMetricToMetricSlice(t, tc.input, sm.Metrics())

			resource := pcommon.NewResource()
			resource.Attributes().FromRaw(tc.resourceAttrs)

			actual := pmetric.NewMetricSlice()
			r := NewRemapper(zaptest.NewLogger(t), remapOpts...)
			r.Remap(sm, actual, resource)
			assert.Equal(t, tc.expected, metricSliceToTestMetric(t, actual))
		})
	}
}

func BenchmarkRemap(b *testing.B) {
	now := pcommon.NewTimestampFromTime(time.Now())
	in := map[string][]testMetric{
		"kubeletstats": []testMetric{
			{Type: Gauge, Name: "k8s.pod.cpu_limit_utilization", DP: testDP{Ts: now, Dbl: ptr(0.26), Attrs: map[string]any{"k8s.pod.name": POD, "k8s.namespace.name": NAMESPACE}}},
			{Type: Gauge, Name: "k8s.pod.memory_limit_utilization", DP: testDP{Ts: now, Dbl: ptr(0.18), Attrs: map[string]any{"k8s.pod.name": "pod0", "k8s.pod.namespace": NAMESPACE}}},
			{Type: Sum, Name: "k8s.pod.network.io", DP: testDP{Ts: now, Int: ptr(int64(1024)), Attrs: map[string]any{"device": DEVICE, "direction": "receive"}}},
			{Type: Sum, Name: "k8s.pod.network.io", DP: testDP{Ts: now, Int: ptr(int64(2048)), Attrs: map[string]any{"device": DEVICE, "direction": "transmit"}}},
		},
	}

	scopeMetrics := make([]pmetric.ScopeMetrics, 0, len(in))
	for scraper, m := range in {
		sm := pmetric.NewScopeMetrics()
		sm.Scope().SetName(fmt.Sprintf("%s/%s", scopePrefix, scraper))
		testMetricToMetricSlice(b, m, sm.Metrics())
		scopeMetrics = append(scopeMetrics, sm)
	}

	r := NewRemapper(zaptest.NewLogger(b))
	resource := pcommon.NewResource()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, sm := range scopeMetrics {
			r.Remap(sm, pmetric.NewMetricSlice(), resource)
		}
	}
}

type testMetric struct {
	Name string
	Type pmetric.MetricType
	DP   testDP
}

type testDP struct {
	Ts    pcommon.Timestamp
	Dbl   *float64
	Int   *int64
	Attrs map[string]any
}

func metricSliceToTestMetric(t *testing.T, ms pmetric.MetricSlice) []testMetric {
	testMetrics := make([]testMetric, ms.Len())
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
		testMetrics[i].DP = testDP{Ts: dp.Timestamp(), Attrs: dp.Attributes().AsRaw()}
		switch dp.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			testMetrics[i].DP.Int = ptr(dp.IntValue())
		case pmetric.NumberDataPointValueTypeDouble:
			testMetrics[i].DP.Dbl = ptr(dp.DoubleValue())
		}
	}

	return testMetrics
}

func testMetricToMetricSlice(t testing.TB, testMetrics []testMetric, out pmetric.MetricSlice) {
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

func ptr[T any](v T) *T {
	return &v
}
