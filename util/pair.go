//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type Pairs []Pair

// Key-value pair
type Pair struct {
	Name  string
	Value interface{}
}

type IPairs []IPair

// Key-value pair
type IPair struct {
	Name  interface{}
	Value interface{}
}
