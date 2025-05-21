package mobile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
		for i := 0; i < len(tc.stacktraces); i++ {
			assert.Equal(t, tc.curated, curateStacktrace(tc.stacktraces[i]))
		}
	}
}

func readTestFile(t *testing.T, file_name string) string {
	bytes, err := os.ReadFile(filepath.Join("testdata", file_name))
	if err != nil {
		t.Fatalf("Could not read test file '%v'", file_name)
	}
	return string(bytes)
}
