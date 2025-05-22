package mobile

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

func CreateGroupingKey(stacktrace string) string {
	hash_bytes := sha256.Sum256([]byte(curateStacktrace(stacktrace)))
	return hex.EncodeToString(hash_bytes[:])
}

func curateStacktrace(stacktrace string) string {
	unwanted_pattern := regexp.MustCompile(`(:\s.+)|[\r\n\s]+`)
	return unwanted_pattern.ReplaceAllString(stacktrace, "")
}
