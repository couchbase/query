//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Package sort sorts in-memory data for the query engine.
package sort

import "sort"

/*
Algorithm sorts data in place. Callers such as the ORDER BY buffer rely on these rules:
  - The sort need not be stable, so callers must not depend on the order of equal
    elements.
  - Less and Swap may be called from several goroutines, but each element, pivots
    included, is touched only by the goroutine whose range contains it:
    execution.Order's Less evaluates and caches ORDER BY terms on the elements it
    compares.
  - A panic in a goroutine that the implementation starts should be re-raised in the
    caller's goroutine, so that the operator's recover handles it. ParallelIntrosort
    does not do this yet.
*/
type Algorithm interface {
	Name() string
	Sort(data sort.Interface)
}

// Sort sorts data in place. The sort is not guaranteed to be stable. It uses ParallelIntrosort,
// which returns equal elements in the same order as earlier releases (MB-63544).
func Sort(data sort.Interface) {
	ParallelIntrosort{}.Sort(data)
}

// Strings sorts a slice of strings in increasing order.
func Strings(a []string) {
	Sort(sort.StringSlice(a))
}
