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

	enricher := NewEnricher()
	enricher.Enrich(logs)

	resource := logs.ResourceLogs().At(0).Resource()
	expectedAttributes := map[string]any{
		"service.name":           "my.service",
		"agent.name":             "android/java",
		"telemetry.sdk.name":     "android",
		"telemetry.sdk.language": "java",
	}
	assert.Empty(t, cmp.Diff(resource.Attributes().AsRaw(), expectedAttributes))
}
