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
	"strconv"
	"strings"
)

// function ParseQuantity:
// parse a string denotating a memory quantity into the number of bytes it denotes.
// e.g. given the string "10K", return 10240
//	given the string "512B", return 512
// Return an error if the number part of the string cannot be converted to an integer
func ParseQuantity(s string) (int, error) {
	quantityTypes := []string{"mb", "kb", "k", "m", "b"}
	l, n, m := len(s), 1, 1

	s = strings.ToLower(s)
	if s[l-1] == 'b' {
		n = 2
	}
	switch rune(s[l-n]) {
	case 'm':
		m = 1024 * 1024
	case 'k':
		m = 1024
	}
	for _, suf := range quantityTypes {
		s = strings.TrimSuffix(s, suf)
	}
	n, err := strconv.Atoi(s)
	return n * m, err
}
