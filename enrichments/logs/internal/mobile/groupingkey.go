package mobile

import "regexp"

func curateStacktrace(s string) string {
	unwanted_pattern := regexp.MustCompile("(:\\s.+)|[\r\n\\s]+")
	return unwanted_pattern.ReplaceAllString(s, "")
}
