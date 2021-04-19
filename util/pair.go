//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

import "strings"

type Pairs []Pair

// Key-value pair
type Pair struct {
	Name  string
	Value interface{}
}

// for sorting
func (s Pairs) Len() int {
	return len(s)
}

func (s Pairs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Pairs) Less(i, j int) bool {
	return strings.Compare(s[i].Name, s[j].Name) < 0
}

type IPairs []IPair

// Key-value pair
type IPair struct {
	Name  interface{}
	Value interface{}
}
