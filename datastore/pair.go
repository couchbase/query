//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package datastore

import (
	"github.com/couchbase/query/value"
)

type Pairs []Pair

// Key-value pair
type Pair struct {
	Key   string
	Value value.Value
}

type AnnotatedPairs []AnnotatedPair

// Key-value pair
type AnnotatedPair struct {
	Key   string
	Value value.AnnotatedValue
}
