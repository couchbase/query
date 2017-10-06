//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

/*
Note: The input slices must be sorted beforehand.
*/
func SortedStringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if a == nil || b == nil {
		return a == nil && b == nil
	}

	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

/*
This code copied verbatim from
http://rosettacode.org/wiki/Reverse_a_string#Go. With gratitude to
rosettacode.org and the author(s).
*/

// reversePreservingCombiningCharacters interprets its argument as UTF-8
// and ignores bytes that do not form valid UTF-8.  return value is UTF-8.
func ReversePreservingCombiningCharacters(s string) string {
	if s == "" {
		return ""
	}
	p := []rune(s)
	r := make([]rune, len(p))
	start := len(r)
	for i := 0; i < len(p); {
		// quietly skip invalid UTF-8
		if p[i] == utf8.RuneError {
			i++
			continue
		}
		j := i + 1
		for j < len(p) && (unicode.Is(unicode.Mn, p[j]) ||
			unicode.Is(unicode.Me, p[j]) || unicode.Is(unicode.Mc, p[j])) {
			j++
		}
		for k := j - 1; k >= i; k-- {
			start--
			r[start] = p[k]
		}
		i = j
	}
	return (string(r[start:]))
}

func TrimSpace(s string) string {

	if len(s) > 0 {
		if s[0] == ' ' || s[len(s)-1] == ' ' {
			s = strings.TrimSpace(s)
		}
	}

	return s
}
