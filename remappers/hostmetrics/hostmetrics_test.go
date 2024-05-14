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
	"testing"
	"time"

	"github.com/elastic/opentelemetry-lib/remappers/common"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zaptest"
)

var (
	Sum   = pmetric.MetricTypeSum
	Gauge = pmetric.MetricTypeGauge
)

func TestRemap(t *testing.T) {
	doTestRemap(t, "without_system_integration", WithSystemIntegrationDataset(false))
	doTestRemap(t, "with_system_integration", WithSystemIntegrationDataset(true))
}

func doTestRemap(t *testing.T, id string, remapOpts ...Option) {
	t.Helper()

	systemIntegration := newConfig(remapOpts...).SystemIntegrationDataset
	outAttr := func(scraper string) map[string]any {
		m := map[string]any{common.OTelTranslatedLabel: true}
		if systemIntegration {
			m[common.DatastreamDatasetLabel] = fmt.Sprintf("system.%s", scraper)
		}
		return m
	}
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, tc := range []struct {
		name     string
		scraper  string
		input    []testMetric
		expected []testMetric
	}{
		{
			name:    "cpu",
			scraper: "cpu",
			input: []testMetric{
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.26), Attrs: map[string]any{"cpu": "cpu0", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.24), Attrs: map[string]any{"cpu": "cpu0", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.5), Attrs: map[string]any{"cpu": "cpu0", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.1), Attrs: map[string]any{"cpu": "cpu0", "state": "steal"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.24), Attrs: map[string]any{"cpu": "cpu1", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.44), Attrs: map[string]any{"cpu": "cpu1", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.32), Attrs: map[string]any{"cpu": "cpu1", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.05), Attrs: map[string]any{"cpu": "cpu1", "state": "steal"}}},
				{Type: Sum, Name: "system.cpu.logical.count", DP: testDP{Ts: now, Int: ptr(int64(4))}},
			},
			expected: []testMetric{
				{Type: Gauge, Name: "system.cpu.total.pct", DP: testDP{Ts: now, Dbl: ptr(1.33), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.idle.pct", DP: testDP{Ts: now, Dbl: ptr(0.82), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.system.pct", DP: testDP{Ts: now, Dbl: ptr(0.68), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.user.pct", DP: testDP{Ts: now, Dbl: ptr(0.5), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.steal.pct", DP: testDP{Ts: now, Dbl: ptr(0.15), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.wait.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.nice.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.irq.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.softirq.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Sum, Name: "system.cpu.cores", DP: testDP{Ts: now, Int: ptr(int64(4)), Attrs: outAttr("cpu")}},
				{Type: Sum, Name: "system.load.cores", DP: testDP{Ts: now, Int: ptr(int64(4)), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.total.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.3325), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.idle.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.205), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.system.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.17), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.user.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.125), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.steal.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.0375), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.wait.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.nice.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.irq.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.softirq.norm.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
			},
		},
		{
			name:    "cpu_without_logical_count",
			scraper: "cpu",
			input: []testMetric{
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.26), Attrs: map[string]any{"cpu": "cpu0", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.24), Attrs: map[string]any{"cpu": "cpu0", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.5), Attrs: map[string]any{"cpu": "cpu0", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.1), Attrs: map[string]any{"cpu": "cpu0", "state": "steal"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.24), Attrs: map[string]any{"cpu": "cpu1", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.44), Attrs: map[string]any{"cpu": "cpu1", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.32), Attrs: map[string]any{"cpu": "cpu1", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.05), Attrs: map[string]any{"cpu": "cpu1", "state": "steal"}}},
			},
			expected: []testMetric{
				{Type: Gauge, Name: "system.cpu.total.pct", DP: testDP{Ts: now, Dbl: ptr(1.33), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.idle.pct", DP: testDP{Ts: now, Dbl: ptr(0.82), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.system.pct", DP: testDP{Ts: now, Dbl: ptr(0.68), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.user.pct", DP: testDP{Ts: now, Dbl: ptr(0.5), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.steal.pct", DP: testDP{Ts: now, Dbl: ptr(0.15), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.wait.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.nice.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.irq.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.softirq.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("cpu")}},
			},
		},
		{
			name:    "load",
			scraper: "load",
			input: []testMetric{
				{Type: Gauge, Name: "system.cpu.load_average.1m", DP: testDP{Ts: now, Dbl: ptr(0.14)}},
				{Type: Gauge, Name: "system.cpu.load_average.5m", DP: testDP{Ts: now, Dbl: ptr(0.12)}},
				{Type: Gauge, Name: "system.cpu.load_average.15m", DP: testDP{Ts: now, Dbl: ptr(0.05)}},
			},
			expected: []testMetric{
				{Type: Gauge, Name: "system.load.1", DP: testDP{Ts: now, Dbl: ptr(0.14), Attrs: outAttr("load")}},
				{Type: Gauge, Name: "system.load.5", DP: testDP{Ts: now, Dbl: ptr(0.12), Attrs: outAttr("load")}},
				{Type: Gauge, Name: "system.load.15", DP: testDP{Ts: now, Dbl: ptr(0.05), Attrs: outAttr("load")}},
			},
		},
		{
			name:    "memory",
			scraper: "memory",
			input: []testMetric{
				{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(1024)), Attrs: map[string]any{"state": "buffered"}}},
				{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(512)), Attrs: map[string]any{"state": "cached"}}},
				{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(256)), Attrs: map[string]any{"state": "inactive"}}},
				{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(2048)), Attrs: map[string]any{"state": "free"}}},
				{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(128)), Attrs: map[string]any{"state": "slab_reclaimable"}}},
				{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(64)), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
				{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(4096)), Attrs: map[string]any{"state": "used"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.133), Attrs: map[string]any{"state": "buffered"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.066), Attrs: map[string]any{"state": "cached"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.033), Attrs: map[string]any{"state": "inactive"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.266), Attrs: map[string]any{"state": "free"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.016), Attrs: map[string]any{"state": "slab_reclaimable"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.008), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.533), Attrs: map[string]any{"state": "used"}}},
			},
			expected: []testMetric{
				// total = used + free + buffered + cached as gopsutil calculates used = total - free - buffered - cached
				{Type: Sum, Name: "system.memory.total", DP: testDP{Ts: now, Int: ptr(int64(7680)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.free", DP: testDP{Ts: now, Int: ptr(int64(2048)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.cached", DP: testDP{Ts: now, Int: ptr(int64(512)), Attrs: outAttr("memory")}},
				// used = used + buffered + cached as gopsutil calculates used = total - free - buffered - cached
				{Type: Sum, Name: "system.memory.used.bytes", DP: testDP{Ts: now, Int: ptr(int64(5632)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.actual.used.bytes", DP: testDP{Ts: now, Int: ptr(int64(5312)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.actual.free", DP: testDP{Ts: now, Int: ptr(int64(2368)), Attrs: outAttr("memory")}},
				{Type: Gauge, Name: "system.memory.used.pct", DP: testDP{Ts: now, Dbl: ptr(0.734), Attrs: outAttr("memory")}},
				{Type: Gauge, Name: "system.memory.actual.used.pct", DP: testDP{Ts: now, Dbl: ptr(0.69), Attrs: outAttr("memory")}},
			},
		},
		{
			name:    "process",
			scraper: "process",
			input: []testMetric{
				{Type: Sum, Name: "process.threads", DP: testDP{Ts: now, Int: ptr(int64(7))}},
				{Type: Gauge, Name: "process.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(15.0)}},
				{Type: Sum, Name: "process.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(2048))}},
				{Type: Sum, Name: "process.memory.virtual", DP: testDP{Ts: now, Int: ptr(int64(128))}},
				{Type: Sum, Name: "process.open_file_descriptors", DP: testDP{Ts: now, Int: ptr(int64(10))}},
				{Type: Sum, Name: "process.cpu.time", DP: testDP{Ts: now, Int: ptr(int64(3)), Attrs: map[string]any{"state": "system"}}},
				{Type: Sum, Name: "process.cpu.time", DP: testDP{Ts: now, Int: ptr(int64(4)), Attrs: map[string]any{"state": "user"}}},
				{Type: Sum, Name: "process.cpu.time", DP: testDP{Ts: now, Int: ptr(int64(5)), Attrs: map[string]any{"state": "wait"}}},
				{Type: Sum, Name: "process.disk.io", DP: testDP{Ts: now, Int: ptr(int64(1024))}},
				{Type: Sum, Name: "process.disk.operations", DP: testDP{Ts: now, Int: ptr(int64(10))}},
			},
			expected: []testMetric{
				{Type: Sum, Name: "process.cpu.start_time", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.num_threads", DP: testDP{Ts: now, Int: ptr(int64(7)), Attrs: outAttr("process")}},
				{Type: Gauge, Name: "system.process.memory.rss.pct", DP: testDP{Ts: now, Dbl: ptr(0.15), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.memory.rss.bytes", DP: testDP{Ts: now, Int: ptr(int64(2048)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.memory.size", DP: testDP{Ts: now, Int: ptr(int64(128)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.fd.open", DP: testDP{Ts: now, Int: ptr(int64(10)), Attrs: outAttr("process")}},
				{Type: Gauge, Name: "process.memory.pct", DP: testDP{Ts: now, Dbl: ptr(0.15), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.total.value", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.system.ticks", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.user.ticks", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.total.ticks", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.read_bytes", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.write_bytes", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.read_ops", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.write_ops", DP: testDP{Ts: now, Int: ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Gauge, Name: "system.process.cpu.total.pct", DP: testDP{Ts: now, Dbl: ptr(0.0), Attrs: outAttr("process")}},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.name, id), func(t *testing.T) {
			sm := pmetric.NewScopeMetrics()
			sm.Scope().SetName(fmt.Sprintf("%s/%s", scopePrefix, tc.scraper))
			testMetricToMetricSlice(t, tc.input, sm.Metrics())

			actual := pmetric.NewMetricSlice()
			r := NewRemapper(zaptest.NewLogger(t), remapOpts...)
			r.Remap(sm, actual, pcommon.NewResource())
			assert.Empty(t, cmp.Diff(tc.expected, metricSliceToTestMetric(t, actual), cmpopts.EquateApprox(0, 0.001)))
		})
	}
}

func BenchmarkRemap(b *testing.B) {
	now := pcommon.NewTimestampFromTime(time.Now())
	in := map[string][]testMetric{
		"cpu": []testMetric{
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.26), Attrs: map[string]any{"cpu": "cpu0", "state": "user"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.24), Attrs: map[string]any{"cpu": "cpu0", "state": "system"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.5), Attrs: map[string]any{"cpu": "cpu0", "state": "idle"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.1), Attrs: map[string]any{"cpu": "cpu0", "state": "steal"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.24), Attrs: map[string]any{"cpu": "cpu1", "state": "user"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.44), Attrs: map[string]any{"cpu": "cpu1", "state": "system"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.32), Attrs: map[string]any{"cpu": "cpu1", "state": "idle"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: testDP{Ts: now, Dbl: ptr(0.05), Attrs: map[string]any{"cpu": "cpu1", "state": "steal"}}},
			{Type: Sum, Name: "system.cpu.logical.count", DP: testDP{Ts: now, Int: ptr(int64(4))}},
		},
		"load": []testMetric{
			{Type: Gauge, Name: "system.cpu.load_average.1m", DP: testDP{Ts: now, Dbl: ptr(0.14)}},
			{Type: Gauge, Name: "system.cpu.load_average.5m", DP: testDP{Ts: now, Dbl: ptr(0.12)}},
			{Type: Gauge, Name: "system.cpu.load_average.15m", DP: testDP{Ts: now, Dbl: ptr(0.05)}},
		},
		"memory": []testMetric{
			{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(1024)), Attrs: map[string]any{"state": "buffered"}}},
			{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(512)), Attrs: map[string]any{"state": "cached"}}},
			{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(256)), Attrs: map[string]any{"state": "inactive"}}},
			{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(2048)), Attrs: map[string]any{"state": "free"}}},
			{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(128)), Attrs: map[string]any{"state": "slab_reclaimable"}}},
			{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(64)), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
			{Type: Sum, Name: "system.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(4096)), Attrs: map[string]any{"state": "used"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.133), Attrs: map[string]any{"state": "buffered"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.066), Attrs: map[string]any{"state": "cached"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.033), Attrs: map[string]any{"state": "inactive"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.266), Attrs: map[string]any{"state": "free"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.016), Attrs: map[string]any{"state": "slab_reclaimable"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.008), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(0.533), Attrs: map[string]any{"state": "used"}}},
		},
		"process": []testMetric{
			{Type: Sum, Name: "process.threads", DP: testDP{Ts: now, Int: ptr(int64(7))}},
			{Type: Gauge, Name: "process.memory.utilization", DP: testDP{Ts: now, Dbl: ptr(15.0)}},
			{Type: Sum, Name: "process.memory.usage", DP: testDP{Ts: now, Int: ptr(int64(2048))}},
			{Type: Sum, Name: "process.memory.virtual", DP: testDP{Ts: now, Int: ptr(int64(128))}},
			{Type: Sum, Name: "process.open_file_descriptors", DP: testDP{Ts: now, Int: ptr(int64(10))}},
			{Type: Sum, Name: "process.cpu.time", DP: testDP{Ts: now, Int: ptr(int64(3)), Attrs: map[string]any{"state": "system"}}},
			{Type: Sum, Name: "process.cpu.time", DP: testDP{Ts: now, Int: ptr(int64(4)), Attrs: map[string]any{"state": "user"}}},
			{Type: Sum, Name: "process.cpu.time", DP: testDP{Ts: now, Int: ptr(int64(5)), Attrs: map[string]any{"state": "wait"}}},
			{Type: Sum, Name: "process.disk.io", DP: testDP{Ts: now, Int: ptr(int64(1024))}},
			{Type: Sum, Name: "process.disk.operations", DP: testDP{Ts: now, Int: ptr(int64(10))}},
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
