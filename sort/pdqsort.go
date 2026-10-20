//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package sort

import "sort"

/*
PDQ is the Go standard library's pattern-defeating quicksort (Go 1.19 and later). It
runs in the caller's goroutine, takes O(n) comparisons on sorted or reversed input and
O(n*log(n)) in the worst case. Because it calls the standard library, it picks up
improvements in new Go releases without changes here.
*/
type PDQ struct{}

func (PDQ) Name() string {
	return "pdqsort"
}

// Sort makes one call to data.Len and O(n*log(n)) calls to data.Less and data.Swap. The
// sort is not guaranteed to be stable.
func (PDQ) Sort(data sort.Interface) {
	sort.Sort(data)
}
