package mobile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCurateStacktrace(t *testing.T) {
	stacktrace := readTestFile(t, "stacktrace1_a.txt")
	curated_stacktrace := readTestFile(t, "curated_stacktrace1.txt")

	assert.Equal(t, curated_stacktrace, curateStacktrace(stacktrace))
}

func readTestFile(t *testing.T, file_name string) string {
	bytes, err := os.ReadFile(filepath.Join("testdata", file_name))
	if err != nil {
		t.Fatalf("Could not read test file '%v'", file_name)
	}
	return string(bytes)
}
