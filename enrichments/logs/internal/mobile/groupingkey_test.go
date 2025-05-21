package mobile

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateGroupingKeyForStacktrace(t *testing.T) {
	stacktrace_path := filepath.Join("testdata", "stacktrace1_a.txt")
	bytes, err := os.ReadFile(stacktrace_path)
	if err != nil {
		assert.Fail(t, "Could not read test file")
	}
	stacktrace := string(bytes)

	fmt.Printf("The stacktrace: %s", stacktrace)
}
