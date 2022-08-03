//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
