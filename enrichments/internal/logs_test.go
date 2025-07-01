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

package elastic

import (
	"path/filepath"
	"testing"

	"github.com/elastic/opentelemetry-lib/enrichments"
	"github.com/elastic/opentelemetry-lib/enrichments/config"
	"github.com/google/go-cmp/cmp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrichResourceLog(t *testing.T) {
	traceFile := filepath.Join("testdata", "logs.yaml")
	logs, err := golden.ReadLogs(traceFile)
	require.NoError(t, err)
	resourceLogs := logs.ResourceLogs().At(0)
	logRecords := resourceLogs.ScopeLogs().At(0).LogRecords()

	// This is needed because the yaml unmarshalling is not yet aware of this new field
	logRecords.At(2).SetEventName("field.name")

	enricher := enrichments.NewEnricher(config.Enabled())
	enricher.EnrichLogs(logs)

	t.Run("resource_enrichment", func(t *testing.T) {
		resourceAttributes := resourceLogs.Resource().Attributes()
		expectedResourceAttributes := map[string]any{
			"service.name":           "my.service",
			"agent.name":             "android/java",
			"agent.version":          "unknown",
			"telemetry.sdk.name":     "android",
			"telemetry.sdk.language": "java",
		}
		assert.Empty(t, cmp.Diff(resourceAttributes.AsRaw(), expectedResourceAttributes))
	})

	for i, tc := range []struct {
		name             string
		processedAsEvent bool
	}{
		{
			name:             "regular_log",
			processedAsEvent: false,
		},
		{
			name:             "event_by_attribute",
			processedAsEvent: true,
		},
		{
			name:             "event_by_field",
			processedAsEvent: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			eventKind, ok := logRecords.At(i).Attributes().Get("event.kind")
			if ok {
				assert.Equal(t, "event", eventKind.AsString())
				assert.True(t, tc.processedAsEvent)
			} else {
				assert.False(t, tc.processedAsEvent)
			}
		})
	}
}
