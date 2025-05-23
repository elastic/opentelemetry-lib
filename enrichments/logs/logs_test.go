package logs

import (
	"path/filepath"
	"testing"

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

	enricher := NewEnricher()
	enricher.Enrich(logs)

	t.Run("resource_enrichment", func(t *testing.T) {
		resourceAttributes := resourceLogs.Resource().Attributes()
		expectedResourceAttributes := map[string]any{
			"service.name":           "my.service",
			"agent.name":             "android/java",
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
