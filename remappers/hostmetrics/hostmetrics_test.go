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
	"github.com/elastic/opentelemetry-lib/remappers/internal"
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

	// Test values to make assertion easier
	PPID      int64  = 101
	ProcOwner string = "root"
	ProcPath  string = "/bin/run"
	ProcName  string = "runner"
	Cmdline   string = "./dist/otelcol-ishleen-custom --config collector.yml"
	Device    string = "en0"
	Disk      string = "nvme0n1p128"
)

func TestRemap(t *testing.T) {
	doTestRemap(t, "without_system_integration", WithSystemIntegrationDataset(false))
	doTestRemap(t, "with_system_integration", WithSystemIntegrationDataset(true))
}

func doTestRemap(t *testing.T, id string, remapOpts ...Option) {
	t.Helper()

	systemIntegration := newConfig(remapOpts...).SystemIntegrationDataset
	outAttr := func(scraper string) map[string]any {
		m := map[string]any{"otel_remapped": true}
		if systemIntegration {
			m[common.DatastreamDatasetLabel] = scraperToElasticDataset[scraper]
		}

		switch scraper {
		case "process":
			m["process.parent.pid"] = PPID
			m["user.name"] = ProcOwner
			m["process.executable"] = ProcPath
			m["process.name"] = ProcName
			m["system.process.cmdline"] = Cmdline
		case "network":
			m["system.network.name"] = Device
		case "disk":
			m["system.diskio.name"] = Disk
		}
		return m
	}
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, tc := range []struct {
		name          string
		scraper       string
		resourceAttrs map[string]any
		input         []internal.TestMetric
		expected      []internal.TestMetric
	}{
		{
			name:    "cpu",
			scraper: "cpu",
			input: []internal.TestMetric{
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.26), Attrs: map[string]any{"cpu": "cpu0", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.24), Attrs: map[string]any{"cpu": "cpu0", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.5), Attrs: map[string]any{"cpu": "cpu0", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.1), Attrs: map[string]any{"cpu": "cpu0", "state": "steal"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.24), Attrs: map[string]any{"cpu": "cpu1", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.44), Attrs: map[string]any{"cpu": "cpu1", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.32), Attrs: map[string]any{"cpu": "cpu1", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.05), Attrs: map[string]any{"cpu": "cpu1", "state": "steal"}}},
				{Type: Sum, Name: "system.cpu.logical.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4))}},
			},
			expected: []internal.TestMetric{
				{Type: Gauge, Name: "system.cpu.total.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(1.33), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.idle.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.82), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.system.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.68), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.user.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.5), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.steal.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.15), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.iowait.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.nice.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.irq.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.softirq.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Sum, Name: "system.cpu.cores", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4)), Attrs: outAttr("cpu")}},
				{Type: Sum, Name: "system.load.cores", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4)), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.total.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.3325), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.idle.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.205), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.system.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.17), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.user.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.125), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.steal.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0375), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.iowait.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.nice.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.irq.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.softirq.norm.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
			},
		},
		{
			name:    "cpu_without_logical_count",
			scraper: "cpu",
			input: []internal.TestMetric{
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.26), Attrs: map[string]any{"cpu": "cpu0", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.24), Attrs: map[string]any{"cpu": "cpu0", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.5), Attrs: map[string]any{"cpu": "cpu0", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.1), Attrs: map[string]any{"cpu": "cpu0", "state": "steal"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.24), Attrs: map[string]any{"cpu": "cpu1", "state": "user"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.44), Attrs: map[string]any{"cpu": "cpu1", "state": "system"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.32), Attrs: map[string]any{"cpu": "cpu1", "state": "idle"}}},
				{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.05), Attrs: map[string]any{"cpu": "cpu1", "state": "steal"}}},
			},
			expected: []internal.TestMetric{
				{Type: Gauge, Name: "system.cpu.total.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(1.33), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.idle.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.82), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.system.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.68), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.user.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.5), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.steal.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.15), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.iowait.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.nice.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.irq.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
				{Type: Gauge, Name: "system.cpu.softirq.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("cpu")}},
			},
		},
		{
			name:    "load",
			scraper: "load",
			input: []internal.TestMetric{
				{Type: Gauge, Name: "system.cpu.load_average.1m", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.14)}},
				{Type: Gauge, Name: "system.cpu.load_average.5m", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.12)}},
				{Type: Gauge, Name: "system.cpu.load_average.15m", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.05)}},
			},
			expected: []internal.TestMetric{
				{Type: Gauge, Name: "system.load.1", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.14), Attrs: outAttr("load")}},
				{Type: Gauge, Name: "system.load.5", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.12), Attrs: outAttr("load")}},
				{Type: Gauge, Name: "system.load.15", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.05), Attrs: outAttr("load")}},
			},
		},
		{
			name:    "memory",
			scraper: "memory",
			input: []internal.TestMetric{
				{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1024)), Attrs: map[string]any{"state": "buffered"}}},
				{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(512)), Attrs: map[string]any{"state": "cached"}}},
				{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(256)), Attrs: map[string]any{"state": "inactive"}}},
				{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048)), Attrs: map[string]any{"state": "free"}}},
				{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(128)), Attrs: map[string]any{"state": "slab_reclaimable"}}},
				{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(64)), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
				{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4096)), Attrs: map[string]any{"state": "used"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.133), Attrs: map[string]any{"state": "buffered"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.066), Attrs: map[string]any{"state": "cached"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.033), Attrs: map[string]any{"state": "inactive"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.266), Attrs: map[string]any{"state": "free"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.016), Attrs: map[string]any{"state": "slab_reclaimable"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.008), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
				{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.533), Attrs: map[string]any{"state": "used"}}},
			},
			expected: []internal.TestMetric{
				// total = used + free + buffered + cached as gopsutil calculates used = total - free - buffered - cached
				{Type: Sum, Name: "system.memory.total", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(7680)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.free", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.cached", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(512)), Attrs: outAttr("memory")}},
				// used = used + buffered + cached as gopsutil calculates used = total - free - buffered - cached
				{Type: Sum, Name: "system.memory.used.bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(5632)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.actual.used.bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(5312)), Attrs: outAttr("memory")}},
				{Type: Sum, Name: "system.memory.actual.free", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2368)), Attrs: outAttr("memory")}},
				{Type: Gauge, Name: "system.memory.used.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.734), Attrs: outAttr("memory")}},
				{Type: Gauge, Name: "system.memory.actual.used.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.69), Attrs: outAttr("memory")}},
			},
		},
		{
			name:    "process",
			scraper: "process",
			resourceAttrs: map[string]any{
				"process.parent_pid":      PPID,
				"process.owner":           ProcOwner,
				"process.executable.path": ProcPath,
				"process.executable.name": ProcName,
				"process.command_line":    Cmdline,
			},
			input: []internal.TestMetric{
				{Type: Sum, Name: "process.threads", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(7))}},
				{Type: Gauge, Name: "process.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(15.0)}},
				{Type: Sum, Name: "process.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048))}},
				{Type: Sum, Name: "process.memory.virtual", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(128))}},
				{Type: Sum, Name: "process.open_file_descriptors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(10))}},
				{Type: Sum, Name: "process.cpu.time", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(3)), Attrs: map[string]any{"state": "system"}}},
				{Type: Sum, Name: "process.cpu.time", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4)), Attrs: map[string]any{"state": "user"}}},
				{Type: Sum, Name: "process.cpu.time", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(5)), Attrs: map[string]any{"state": "wait"}}},
				{Type: Sum, Name: "process.disk.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1024))}},
				{Type: Sum, Name: "process.disk.operations", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(10))}},
			},
			expected: []internal.TestMetric{
				{Type: Sum, Name: "process.cpu.start_time", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.num_threads", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(7)), Attrs: outAttr("process")}},
				{Type: Gauge, Name: "system.process.memory.rss.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.15), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.memory.rss.bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.memory.size", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(128)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.fd.open", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(10)), Attrs: outAttr("process")}},
				{Type: Gauge, Name: "process.memory.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.15), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.total.value", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.system.ticks", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.user.ticks", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.cpu.total.ticks", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.read_bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.write_bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.read_ops", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Sum, Name: "system.process.io.write_ops", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(0)), Attrs: outAttr("process")}},
				{Type: Gauge, Name: "system.process.cpu.total.pct", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.0), Attrs: outAttr("process")}},
			},
		},
		{
			name:    "processes",
			scraper: "processes",
			input: []internal.TestMetric{
				{Type: Sum, Name: "system.processes.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(7)), Attrs: map[string]any{"status": "idle"}}},
				{Type: Sum, Name: "system.processes.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(3)), Attrs: map[string]any{"status": "sleeping"}}},
				{Type: Sum, Name: "system.processes.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(5)), Attrs: map[string]any{"status": "stopped"}}},
				{Type: Sum, Name: "system.processes.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1)), Attrs: map[string]any{"status": "zombies"}}},
				{Type: Sum, Name: "system.processes.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2)), Attrs: map[string]any{"status": "running"}}},
				{Type: Sum, Name: "system.processes.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2)), Attrs: map[string]any{"status": "paging"}}},
			},
			expected: []internal.TestMetric{
				{Type: Sum, Name: "system.process.summary.idle", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(7)), Attrs: outAttr("processes")}},
				{Type: Sum, Name: "system.process.summary.sleeping", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(3)), Attrs: outAttr("processes")}},
				{Type: Sum, Name: "system.process.summary.stopped", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(5)), Attrs: outAttr("processes")}},
				{Type: Sum, Name: "system.process.summary.zombie", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1)), Attrs: outAttr("processes")}},
				{Type: Sum, Name: "system.process.summary.running", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2)), Attrs: outAttr("processes")}},
				{Type: Sum, Name: "system.process.summary.total", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(20)), Attrs: outAttr("processes")}},
			},
		},
		{
			name:    "network",
			scraper: "network",
			input: []internal.TestMetric{
				{Type: Sum, Name: "system.network.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1024)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
				{Type: Sum, Name: "system.network.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
				{Type: Sum, Name: "system.network.packets", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(11)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
				{Type: Sum, Name: "system.network.packets", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(9)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
				{Type: Sum, Name: "system.network.dropped", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(3)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
				{Type: Sum, Name: "system.network.dropped", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
				{Type: Sum, Name: "system.network.errors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
				{Type: Sum, Name: "system.network.errors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
			},
			expected: []internal.TestMetric{
				{Type: Sum, Name: "system.network.in.bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1024)), Attrs: outAttr("network")}},
				{Type: Sum, Name: "system.network.out.bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048)), Attrs: outAttr("network")}},
				{Type: Sum, Name: "system.network.in.packets", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(11)), Attrs: outAttr("network")}},
				{Type: Sum, Name: "system.network.out.packets", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(9)), Attrs: outAttr("network")}},
				{Type: Sum, Name: "system.network.in.dropped", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(3)), Attrs: outAttr("network")}},
				{Type: Sum, Name: "system.network.out.dropped", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4)), Attrs: outAttr("network")}},
				{Type: Sum, Name: "system.network.in.errors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1)), Attrs: outAttr("network")}},
				{Type: Sum, Name: "system.network.out.errors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2)), Attrs: outAttr("network")}},
			},
		},
		{
			name:    "disk",
			scraper: "disk",
			input: []internal.TestMetric{
				{Type: Sum, Name: "system.disk.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1888256)), Attrs: map[string]any{"device": Disk, "direction": "read"}}},
				{Type: Sum, Name: "system.disk.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(512)), Attrs: map[string]any{"device": Disk, "direction": "write"}}},
				{Type: Sum, Name: "system.disk.operations", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(15390)), Attrs: map[string]any{"device": Disk, "direction": "read"}}},
				{Type: Sum, Name: "system.disk.operations", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(371687)), Attrs: map[string]any{"device": Disk, "direction": "write"}}},
				{Type: Sum, Name: "system.disk.operation_time", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(11.182), Attrs: map[string]any{"device": Disk, "direction": "read"}}},
				{Type: Sum, Name: "system.disk.operation_time", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(617.289), Attrs: map[string]any{"device": Disk, "direction": "write"}}},
				{Type: Sum, Name: "system.disk.io_time", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(520.3), Attrs: map[string]any{"device": Disk}}},
				{Type: Sum, Name: "system.disk.pending_operations", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(102)), Attrs: map[string]any{"device": Disk}}},
			},
			expected: []internal.TestMetric{
				{Type: Sum, Name: "system.diskio.read.bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1888256)), Attrs: outAttr("disk")}},
				{Type: Sum, Name: "system.diskio.write.bytes", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(512)), Attrs: outAttr("disk")}},
				{Type: Sum, Name: "system.diskio.read.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(15390)), Attrs: outAttr("disk")}},
				{Type: Sum, Name: "system.diskio.write.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(371687)), Attrs: outAttr("disk")}},
				{Type: Sum, Name: "system.diskio.read.time", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(11182.0), Attrs: outAttr("disk")}},
				{Type: Sum, Name: "system.diskio.write.time", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(617289.0), Attrs: outAttr("disk")}},
				{Type: Sum, Name: "system.diskio.io.time", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(520300.0), Attrs: outAttr("disk")}},
				{Type: Sum, Name: "system.diskio.io.ops", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(102)), Attrs: outAttr("disk")}},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.name, id), func(t *testing.T) {
			sm := pmetric.NewScopeMetrics()
			sm.Scope().SetName(fmt.Sprintf("%s/%s", scopePrefix, tc.scraper))
			internal.TestMetricToMetricSlice(t, tc.input, sm.Metrics())

			resource := pcommon.NewResource()
			resource.Attributes().FromRaw(tc.resourceAttrs)

			actual := pmetric.NewMetricSlice()
			r := NewRemapper(zaptest.NewLogger(t), remapOpts...)
			r.Remap(sm, actual, resource)
			assert.Empty(t, cmp.Diff(tc.expected, internal.MetricSliceToTestMetric(t, actual), cmpopts.EquateApprox(0, 0.001)))
		})
	}
}

func BenchmarkRemap(b *testing.B) {
	now := pcommon.NewTimestampFromTime(time.Now())
	in := map[string][]internal.TestMetric{
		"cpu": []internal.TestMetric{
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.26), Attrs: map[string]any{"cpu": "cpu0", "state": "user"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.24), Attrs: map[string]any{"cpu": "cpu0", "state": "system"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.5), Attrs: map[string]any{"cpu": "cpu0", "state": "idle"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.1), Attrs: map[string]any{"cpu": "cpu0", "state": "steal"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.24), Attrs: map[string]any{"cpu": "cpu1", "state": "user"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.44), Attrs: map[string]any{"cpu": "cpu1", "state": "system"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.32), Attrs: map[string]any{"cpu": "cpu1", "state": "idle"}}},
			{Type: Gauge, Name: "system.cpu.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.05), Attrs: map[string]any{"cpu": "cpu1", "state": "steal"}}},
			{Type: Sum, Name: "system.cpu.logical.count", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4))}},
		},
		"load": []internal.TestMetric{
			{Type: Gauge, Name: "system.cpu.load_average.1m", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.14)}},
			{Type: Gauge, Name: "system.cpu.load_average.5m", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.12)}},
			{Type: Gauge, Name: "system.cpu.load_average.15m", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.05)}},
		},
		"memory": []internal.TestMetric{
			{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1024)), Attrs: map[string]any{"state": "buffered"}}},
			{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(512)), Attrs: map[string]any{"state": "cached"}}},
			{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(256)), Attrs: map[string]any{"state": "inactive"}}},
			{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048)), Attrs: map[string]any{"state": "free"}}},
			{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(128)), Attrs: map[string]any{"state": "slab_reclaimable"}}},
			{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(64)), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
			{Type: Sum, Name: "system.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4096)), Attrs: map[string]any{"state": "used"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.133), Attrs: map[string]any{"state": "buffered"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.066), Attrs: map[string]any{"state": "cached"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.033), Attrs: map[string]any{"state": "inactive"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.266), Attrs: map[string]any{"state": "free"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.016), Attrs: map[string]any{"state": "slab_reclaimable"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.008), Attrs: map[string]any{"state": "slab_unreclaimable"}}},
			{Type: Gauge, Name: "system.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(0.533), Attrs: map[string]any{"state": "used"}}},
		},
		"process": []internal.TestMetric{
			{Type: Sum, Name: "process.threads", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(7))}},
			{Type: Gauge, Name: "process.memory.utilization", DP: internal.TestDP{Ts: now, Dbl: internal.Ptr(15.0)}},
			{Type: Sum, Name: "process.memory.usage", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048))}},
			{Type: Sum, Name: "process.memory.virtual", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(128))}},
			{Type: Sum, Name: "process.open_file_descriptors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(10))}},
			{Type: Sum, Name: "process.cpu.time", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(3)), Attrs: map[string]any{"state": "system"}}},
			{Type: Sum, Name: "process.cpu.time", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4)), Attrs: map[string]any{"state": "user"}}},
			{Type: Sum, Name: "process.cpu.time", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(5)), Attrs: map[string]any{"state": "wait"}}},
			{Type: Sum, Name: "process.disk.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1024))}},
			{Type: Sum, Name: "process.disk.operations", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(10))}},
		},
		"network": []internal.TestMetric{
			{Type: Sum, Name: "system.network.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1024)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
			{Type: Sum, Name: "system.network.io", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2048)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
			{Type: Sum, Name: "system.network.packets", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(11)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
			{Type: Sum, Name: "system.network.packets", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(9)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
			{Type: Sum, Name: "system.network.dropped", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(3)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
			{Type: Sum, Name: "system.network.dropped", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(4)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
			{Type: Sum, Name: "system.network.errors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(1)), Attrs: map[string]any{"device": Device, "direction": "receive"}}},
			{Type: Sum, Name: "system.network.errors", DP: internal.TestDP{Ts: now, Int: internal.Ptr(int64(2)), Attrs: map[string]any{"device": Device, "direction": "transmit"}}},
		},
	}

	scopeMetrics := make([]pmetric.ScopeMetrics, 0, len(in))
	for scraper, m := range in {
		sm := pmetric.NewScopeMetrics()
		sm.Scope().SetName(fmt.Sprintf("%s/%s", scopePrefix, scraper))
		internal.TestMetricToMetricSlice(b, m, sm.Metrics())
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
