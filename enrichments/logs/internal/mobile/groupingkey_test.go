package mobile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurateStacktrace(t *testing.T) {
	for _, tc := range []struct {
		name        string
		stacktraces []string
		curated     string
	}{
		{
			name:        "standalone_stacktrace",
			stacktraces: []string{readTestFile(t, "stacktrace1_a.txt"), readTestFile(t, "stacktrace1_b.txt")},
			curated:     readTestFile(t, "curated_stacktrace1.txt"),
		},
		{
			name:        "stacktrace_with_cause",
			stacktraces: []string{readTestFile(t, "stacktrace2_a.txt"), readTestFile(t, "stacktrace2_b.txt"), readTestFile(t, "stacktrace2_c.txt")},
			curated:     readTestFile(t, "curated_stacktrace2.txt"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, stacktrace := range tc.stacktraces {
				assert.Equal(t, tc.curated, curateStacktrace(stacktrace))
			}
		})
	}
}

func TestCreateGroupingKey(t *testing.T) {
	for _, tc := range []struct {
		name        string
		stacktraces []string
		expectedId  string
	}{
		{
			name:        "standalone_stacktrace",
			stacktraces: []string{readTestFile(t, "stacktrace1_a.txt"), readTestFile(t, "stacktrace1_b.txt")},
			expectedId:  "fbda93aa3a0ccb785da705ff03697bb7393cb0738e31c596e797b16ea2ba2f02",
		},
		{
			name:        "stacktrace_with_cause",
			stacktraces: []string{readTestFile(t, "stacktrace2_a.txt"), readTestFile(t, "stacktrace2_b.txt"), readTestFile(t, "stacktrace2_c.txt")},
			expectedId:  "10b6726bc6d9aab69389b8b5dfbd9df1a6e08db75b0777b4f2d0456adeaed50e",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, stacktrace := range tc.stacktraces {
				assert.Equal(t, tc.expectedId, CreateGroupingKey(stacktrace))
			}
		})
	}
}

func readTestFile(t *testing.T, fileName string) string {
	bytes, err := os.ReadFile(filepath.Join("testdata", fileName))
	require.NoError(t, err)
	return string(bytes)
}
