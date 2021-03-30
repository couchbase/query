//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package value

type Pairs []Pair

// Key-value-options pair
type Pair struct {
	Name    string
	Value   Value
	Options Value
}

type AnnotatedPairs []AnnotatedPair

// Key-value pair
type AnnotatedPair struct {
	Name  string
	Value AnnotatedValue
}

type VPairs []VPair

// Key-value pair
type VPair struct {
	Name  Value
	Value Value
}

type AnnotatedJoinPairs []AnnotatedJoinPair

// Value-JoinKeys pair
type AnnotatedJoinPair struct {
	Value AnnotatedValue
	Keys  []string
}
