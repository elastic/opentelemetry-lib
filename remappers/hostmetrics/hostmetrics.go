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
	"path"
	"strings"

	"github.com/elastic/opentelemetry-lib/remappers/common"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const scopePrefix = "otelcol/hostmetricsreceiver"

var scraperToElasticDataset = map[string]string{
	"cpu":        "system.cpu",
	"disk":       "system.diskio",
	"filesystem": "system.filesystem",
	"load":       "system.load",
	"memory":     "system.memory",
	"network":    "system.network",
	"paging":     "system.memory",
	"processes":  "system.process.summary",
	"process":    "system.process",
}

type remapFunc func(pmetric.MetricSlice, pmetric.MetricSlice, pcommon.Resource, func(pmetric.NumberDataPoint)) error

var remapFuncs = map[string]remapFunc{
	"cpu":        remapCPUMetrics,
	"memory":     remapMemoryMetrics,
	"load":       remapLoadMetrics,
	"process":    remapProcessMetrics,
	"processes":  remapProcessesMetrics,
	"network":    remapNetworkMetrics,
	"disk":       remapDiskMetrics,
	"filesystem": remapFilesystemMetrics,
}

// Remapper maps the OTel hostmetrics to Elastic system metrics. These remapped
// metrics power the curated Kibana dashboards. Each datapoint translated using
// the remapper has the `event.processor` attribute set to `hostmetrics`.
type Remapper struct {
	logger *zap.Logger
	cfg    config
}

// NewRemapper creates a new instance of hostmetrics remapper.
func NewRemapper(logger *zap.Logger, opts ...Option) *Remapper {
	return &Remapper{
		cfg:    newConfig(opts...),
		logger: logger,
	}
}

// Remap remaps an OTel ScopeMetrics to a list of OTel metrics such that the
// remapped metrics could be trivially converted into Elastic system metrics.
// It accepts the resource attributes to enrich the remapped metrics as per
// Elastic convention. The current remapping logic assumes that each Metric
// in the ScopeMetric will have datapoints for a single timestamp only. The
// remapped metrics are added to the output `MetricSlice`.
func (r *Remapper) Remap(
	src pmetric.ScopeMetrics,
	out pmetric.MetricSlice,
	resource pcommon.Resource,
) {
	if !r.Valid(src) {
		return
	}

	scope := src.Scope()
	scraper := path.Base(scope.Name())

	dataset, ok := scraperToElasticDataset[scraper]
	if !ok {
		r.logger.Warn("no dataset defined for scraper", zap.String("scraper", scraper))
		return
	}
	datasetMutator := func(m pmetric.NumberDataPoint) {
		m.Attributes().PutStr(common.EventDatasetLabel, dataset)
		m.Attributes().PutStr(common.EventModuleLabel, "system")
		if r.cfg.SystemIntegrationDataset {
			m.Attributes().PutStr(common.DatastreamDatasetLabel, dataset)
		}
	}

	remapFunc, ok := remapFuncs[scraper]
	if !ok {
		return
	}
	err := remapFunc(src.Metrics(), out, resource, datasetMutator)
	if err != nil {
		r.logger.Warn(
			"failed to remap OTel hostmetrics",
			zap.String("scope", scope.Name()),
			zap.Error(err),
		)
	}
}

// Valid validates a ScopeMetric against the hostmetrics remapper requirements.
// Hostmetrics remapper only remaps metrics from hostmetricsreceiver.
func (r *Remapper) Valid(sm pmetric.ScopeMetrics) bool {
	return strings.HasPrefix(sm.Scope().Name(), scopePrefix)
}
