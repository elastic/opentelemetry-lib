// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mobile

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cespare/xxhash/v2"
)

var (
	// Regex patterns for java stack trace processing
	errorOrCausePattern = regexp.MustCompile(`^((?:Caused\sby:\s[^:]+)|(?:[^\s][^:]+))(:\s.+)?$`)
	callSitePattern     = regexp.MustCompile(`^\s+at\s.+(:\d+)\)$`)
	unwantedPattern     = regexp.MustCompile(`\s+`)
	allLinesPattern     = regexp.MustCompile(`(m?).+`)

	// Regex patterns for swift stack trace processing
	iosCrashedThreadPattern = regexp.MustCompile(`^Thread\s+(\d+)\s+Crashed:$`)
	iosThreadPattern        = regexp.MustCompile(`^Thread\s+(\d+):$`)
)

func CreateJavaStacktraceGroupingKey(stacktrace string) string {
	hash := xxhash.Sum64String(curateJavaStacktrace(stacktrace))
	return fmt.Sprintf("%016x", hash)
}

func curateJavaStacktrace(stacktrace string) string {
	curatedLines := allLinesPattern.ReplaceAllStringFunc(stacktrace, func(s string) string {
		if errorOrCausePattern.MatchString(s) {
			return errorOrCausePattern.ReplaceAllString(s, "$1")
		}
		if callSitePattern.MatchString(s) {
			return strings.Replace(s, callSitePattern.FindStringSubmatch(s)[1], "", 1)
		}
		return s
	})

	return unwantedPattern.ReplaceAllString(curatedLines, "")
}

func CreateSwiftStacktraceGroupingKey(stacktrace string) string {
	// Extract the crashed thread block
	crashThreadContent := extractCrashedThread(stacktrace)

	// Hash the crashed thread content using xxhash
	hash := xxhash.Sum64String(crashThreadContent)
	return fmt.Sprintf("%016x", hash)
}

func extractCrashedThread(stacktrace string) string {
	lines := strings.Split(stacktrace, "\n")

	var crashedThreadLines []string
	var inCrashedThread bool

	for i, line := range lines {
		// Check if this line indicates the start of the crashed thread
		if iosCrashedThreadPattern.MatchString(line) {
			inCrashedThread = true
			crashThreadMatch := iosCrashedThreadPattern.FindStringSubmatch(line)
			crashThreadNum := crashThreadMatch[1]
			crashThreadHeader := fmt.Sprintf("Thread %s Crashed:", crashThreadNum)
			crashThreadVerbatim := fmt.Sprintf("Thread%sCrashed", crashThreadNum)
			crashThreadFormatted := unwantedPattern.ReplaceAllString(crashThreadVerbatim, "")
			crashThreadHeader = unwantedPattern.ReplaceAllString(crashThreadHeader, "")

			// Start capturing the crashed thread, beginning with the header
			crashGrepKey := fmt.Sprintf("%s:", crashThreadFormatted)
			crashedThreadLines = append(crashedThreadLines, crashGrepKey)
			continue
		}

		// Check if we've found the end of the crashed thread section
		// End is marked by either a new thread section or an empty line followed by another section
		if inCrashedThread && (iosThreadPattern.MatchString(line) ||
			(len(strings.TrimSpace(line)) == 0 && i+1 < len(lines) && len(strings.TrimSpace(lines[i+1])) > 0 &&
				!strings.HasPrefix(strings.TrimSpace(lines[i+1]), "0"))) {
			break
		}

		// Add line to the crashed thread content if we're in the crashed thread section
		if inCrashedThread && len(strings.TrimSpace(line)) > 0 {
			// Format the line to remove unnecessary whitespace
			formattedLine := unwantedPattern.ReplaceAllString(line, "")
			crashLines := unwantedPattern.ReplaceAllString(formattedLine, "")
			crashLines = strings.ReplaceAll(crashLines, ":", "")
			crashLines = strings.ReplaceAll(crashLines, "+", "")
			crashLines = strings.ReplaceAll(crashLines, "0x", "")
			crashLines = strings.Join(strings.Fields(crashLines), "")
			crashLines = strings.Trim(crashLines, "0123456789")

			// Add the formatted line to our result
			if len(crashLines) > 0 {
				crashedThreadLines = append(crashedThreadLines, crashLines)
			}
		}
	}

	// Join all lines with a separator to create the final string for hashing
	return strings.Join(crashedThreadLines, "")
}
