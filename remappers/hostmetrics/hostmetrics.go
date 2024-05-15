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
	"processes":  "system.process",
	"process":    "system.process",
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
// The current remapping logic assumes that each Metric in the ScopeMetric
// will have datapoints for a single timestamp only. The remapped metrics are
// added to the output `MetricSlice`.
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

	var dataset string // an empty dataset defers setting dataset to the caller
	if r.cfg.SystemIntegrationDataset {
		var ok bool
		dataset, ok = scraperToElasticDataset[scraper]
		if !ok {
			r.logger.Warn("no dataset defined for scraper", zap.String("scraper", scraper))
			return
		}
	}

	var err error
	switch scraper {
	case "cpu":
		err = remapCPUMetrics(src.Metrics(), out, resource, dataset)
	case "memory":
		err = remapMemoryMetrics(src.Metrics(), out, resource, dataset)
	case "load":
		err = remapLoadMetrics(src.Metrics(), out, resource, dataset)
	case "process":
		err = remapProcessMetrics(src.Metrics(), out, resource, dataset)
	case "processes":
		err = remapProcessesMetrics(src.Metrics(), out, resource, dataset)
	case "network":
		err = remapNetworkMetrics(src.Metrics(), out, resource, dataset)
	}

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
