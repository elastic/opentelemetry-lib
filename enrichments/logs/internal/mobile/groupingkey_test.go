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
		stacktraces []string
		curated     string
	}{
		{
			stacktraces: []string{readTestFile(t, "stacktrace1_a.txt"), readTestFile(t, "stacktrace1_b.txt")},
			curated:     readTestFile(t, "curated_stacktrace1.txt"),
		},
		{
			stacktraces: []string{readTestFile(t, "stacktrace2_a.txt"), readTestFile(t, "stacktrace2_b.txt"), readTestFile(t, "stacktrace2_c.txt")},
			curated:     readTestFile(t, "curated_stacktrace2.txt"),
		},
	} {
		for _, stacktrace := range tc.stacktraces {
			assert.Equal(t, tc.curated, curateStacktrace(stacktrace))
		}
	}
}

func TestCreateGroupingKey(t *testing.T) {
	for _, tc := range []struct {
		stacktraces []string
		expected_id string
	}{
		{
			stacktraces: []string{readTestFile(t, "stacktrace1_a.txt"), readTestFile(t, "stacktrace1_b.txt")},
			expected_id: "fbda93aa3a0ccb785da705ff03697bb7393cb0738e31c596e797b16ea2ba2f02",
		},
		{
			stacktraces: []string{readTestFile(t, "stacktrace2_a.txt"), readTestFile(t, "stacktrace2_b.txt"), readTestFile(t, "stacktrace2_c.txt")},
			expected_id: "10b6726bc6d9aab69389b8b5dfbd9df1a6e08db75b0777b4f2d0456adeaed50e",
		},
	} {
		for _, stacktrace := range tc.stacktraces {
			assert.Equal(t, tc.expected_id, CreateGroupingKey(stacktrace))
		}
	}
}

func readTestFile(t *testing.T, file_name string) string {
	bytes, err := os.ReadFile(filepath.Join("testdata", file_name))
	require.NoError(t, err)
	return string(bytes)
}
