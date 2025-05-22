package mobile

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

func CreateGroupingKey(stacktrace string) string {
	hashBytes := sha256.Sum256([]byte(curateStacktrace(stacktrace)))
	return hex.EncodeToString(hashBytes[:])
}

func curateStacktrace(stacktrace string) string {
	unwantedPattern := regexp.MustCompile(`(:\s.+)|[\r\n\s]+`)
	return unwantedPattern.ReplaceAllString(stacktrace, "")
}
