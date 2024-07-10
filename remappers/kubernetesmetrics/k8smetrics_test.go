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
	"github.com/elastic/opentelemetry-lib/remappers/internal/testutils"
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

	kubernetesIntegration := newConfig(remapOpts...).KubernetesIntegrationDataset
	outAttr := func(scraper string) map[string]any {
		dataset := scraperToElasticDataset[scraper]
		m := map[string]any{
			common.OTelRemappedLabel: true,
			common.EventDatasetLabel: dataset,
			"service.type":           "kubernetes",
		}
		if kubernetesIntegration {
			m[common.DatastreamDatasetLabel] = dataset
		}
		return m
	}
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, tc := range []struct {
		name          string
		scraper       string
		resourceAttrs map[string]any
		input         []testutils.TestMetric
		expected      []testutils.TestMetric
	}{
		{
			name:    "kubeletstats",
			scraper: "kubeletstatsreceiver",
			resourceAttrs: map[string]any{
				"k8s.pod.name":       POD,
				"k8s.namespace.name": NAMESPACE,
				"k8s.device":         DEVICE,
			},
			input: []testutils.TestMetric{
				{Type: Gauge, Name: "k8s.pod.cpu_limit_utilization", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.26), Attrs: map[string]any{"k8s.pod.name": POD, "k8s.namespace.name": NAMESPACE}}},
				{Type: Gauge, Name: "k8s.pod.memory_limit_utilization", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.18), Attrs: map[string]any{"k8s.pod.name": "pod0", "k8s.pod.namespace": NAMESPACE}}},
				{Type: Gauge, Name: "k8s.pod.cpu.node.utilization", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.12), Attrs: map[string]any{"k8s.pod.name": POD, "k8s.namespace.name": NAMESPACE}}},
				{Type: Sum, Name: "k8s.pod.network.io", DP: testutils.TestDP{Ts: now, Int: testutils.Ptr(int64(1024)), Attrs: map[string]any{"device": DEVICE, "direction": "receive"}}},
				{Type: Sum, Name: "k8s.pod.network.io", DP: testutils.TestDP{Ts: now, Int: testutils.Ptr(int64(2048)), Attrs: map[string]any{"device": DEVICE, "direction": "transmit"}}},
			},
			expected: []testutils.TestMetric{
				{Type: Gauge, Name: "kubernetes.pod.cpu.usage.limit.pct", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.26), Attrs: outAttr("kubeletstatsreceiver")}},
				{Type: Gauge, Name: "kubernetes.pod.cpu.usage.node.pct", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.12), Attrs: outAttr("kubeletstatsreceiver")}},
				{Type: Gauge, Name: "kubernetes.pod.memory.usage.node.pct", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.0), Attrs: outAttr("kubeletstatsreceiver")}},
				{Type: Gauge, Name: "kubernetes.pod.memory.usage.limit.pct", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.18), Attrs: outAttr("kubeletstatsreceiver")}},
				{Type: Sum, Name: "kubernetes.pod.network.tx.bytes", DP: testutils.TestDP{Ts: now, Int: testutils.Ptr(int64(2048)), Attrs: outAttr("kubeletstatsreceiver")}},
				{Type: Sum, Name: "kubernetes.pod.network.rx.bytes", DP: testutils.TestDP{Ts: now, Int: testutils.Ptr(int64(1024)), Attrs: outAttr("kubeletstatsreceiver")}},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.name, id), func(t *testing.T) {
			sm := pmetric.NewScopeMetrics()
			sm.Scope().SetName(fmt.Sprintf("%s/%s", scopePrefix, tc.scraper))
			testutils.TestMetricToMetricSlice(t, tc.input, sm.Metrics())

			resource := pcommon.NewResource()
			resource.Attributes().FromRaw(tc.resourceAttrs)

			actual := pmetric.NewMetricSlice()
			r := NewRemapper(zaptest.NewLogger(t), remapOpts...)
			r.Remap(sm, actual, resource)
			assert.Equal(t, tc.expected, testutils.MetricSliceToTestMetric(t, actual))
		})
	}
}

func BenchmarkRemap(b *testing.B) {
	now := pcommon.NewTimestampFromTime(time.Now())
	in := map[string][]testutils.TestMetric{
		"kubeletstatsreceiver": []testutils.TestMetric{
			{Type: Gauge, Name: "k8s.pod.cpu_limit_utilization", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.26), Attrs: map[string]any{"k8s.pod.name": POD, "k8s.namespace.name": NAMESPACE}}},
			{Type: Gauge, Name: "k8s.pod.cpu.node.utilization", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.12), Attrs: map[string]any{"k8s.pod.name": POD, "k8s.namespace.name": NAMESPACE}}},
			{Type: Gauge, Name: "k8s.pod.memory_limit_utilization", DP: testutils.TestDP{Ts: now, Dbl: testutils.Ptr(0.18), Attrs: map[string]any{"k8s.pod.name": "pod0", "k8s.pod.namespace": NAMESPACE}}},
			{Type: Sum, Name: "k8s.pod.network.io", DP: testutils.TestDP{Ts: now, Int: testutils.Ptr(int64(1024)), Attrs: map[string]any{"device": DEVICE, "direction": "receive"}}},
			{Type: Sum, Name: "k8s.pod.network.io", DP: testutils.TestDP{Ts: now, Int: testutils.Ptr(int64(2048)), Attrs: map[string]any{"device": DEVICE, "direction": "transmit"}}},
		},
	}

	scopeMetrics := make([]pmetric.ScopeMetrics, 0, len(in))
	for scraper, m := range in {
		sm := pmetric.NewScopeMetrics()
		sm.Scope().SetName(fmt.Sprintf("%s/%s", scopePrefix, scraper))
		testutils.TestMetricToMetricSlice(b, m, sm.Metrics())
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
