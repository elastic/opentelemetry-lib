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
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

func CreateGroupingKey(stacktrace string) string {
	hashBytes := sha256.Sum256([]byte(curateStacktrace(stacktrace)))
	return hex.EncodeToString(hashBytes[:])
}

func curateStacktrace(stacktrace string) string {
	errorOrCausePattern := regexp.MustCompile(`^((?:Caused\sby:\s[^:]+)|(?:[^\s][^:]+))(:\s.+)?$`)
	callSitePattern := regexp.MustCompile(`^\s+at\s.+(:\d+)\)$`)
	unwantedPattern := regexp.MustCompile(`\s+`)

	curatedLines := regexp.MustCompile(`(m?).+`).ReplaceAllStringFunc(stacktrace, func(s string) string {
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
