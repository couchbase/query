//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package sort

import (
	"runtime"
	"runtime/debug"
	"sort"

	"github.com/couchbase/query/logging"
)

/*
ParallelPDQ splits large inputs into ranges with the same partitioning step as
ParallelIntrosort (doPivot), sorts the ranges in separate goroutines, and sorts each
range of fewer than _PDQ_LEAF elements with the standard library's pdqsort.

Inputs smaller than two leaves go straight to pdqsort, and input that is already in
order, or in reverse order, is detected in one pass of at most n comparisons. A panic
in any of its goroutines is re-raised in the caller's goroutine.
*/
type ParallelPDQ struct{}

// Ranges smaller than this are sorted by pdqsort in a single goroutine.
const _PDQ_LEAF = 2048

func (ParallelPDQ) Name() string {
	return "parallel-pdqsort"
}

// Sort makes one call to data.Len and O(n*log(n)) calls to data.Less and data.Swap. The
// sort is not guaranteed to be stable. At most runtime.GOMAXPROCS(0) goroutines sort at a
// time.
func (ParallelPDQ) Sort(data sort.Interface) {
	sortParallelPDQ(data, runtime.GOMAXPROCS(0))
}

// sortParallelPDQ sorts data with at most parallelism goroutines sorting at a time,
// counting the caller's.
func sortParallelPDQ(data sort.Interface, parallelism int) {
	n := data.Len()
	if n < 2*_PDQ_LEAF || parallelism == 1 {
		sort.Sort(data)
		return
	}
	if presorted(data, n) {
		return
	}

	// A goroutine sorts only while it holds a token, so at most parallelism goroutines
	// sort at a time. Goroutines are started without waiting for a token, so that one is
	// ready to run whenever a token is returned. Each waiting goroutine owns an unsorted
	// range of at least _PDQ_LEAF elements, so at most n/_PDQ_LEAF wait at once (about 5k
	// for 10M rows, a few MB of stacks); measured peaks were a quarter of that or less.
	tokens := make(chan struct{}, parallelism)
	tokens <- struct{}{}
	if r := parallelPDQ(data, 0, n, depthLimit(n), tokens); r != nil {
		panic(r)
	}
}

/*
parallelPDQ sorts data[a:b], holding one of tokens. It returns the token before it waits
for the goroutines it started, and returns the value of the first panic raised in any of
them. A panic in this goroutine is not recovered, so that it keeps the stack of the code
that raised it; the goroutines this call started still finish first.
*/
func parallelPDQ(data sort.Interface, a, b, maxDepth int, tokens chan struct{}) (childPanic any) {
	var children chan any
	childCount := 0
	defer func() {
		<-tokens
		for ; childCount > 0; childCount-- {
			if r := <-children; r != nil && childPanic == nil {
				childPanic = r
			}
		}
	}()

	// Partitioning a range smaller than two leaves cannot give two sides that are both
	// big enough for a goroutine, so pdqsort sorts it. Past the depth limit, pdqsort's
	// own heapsort fallback bounds the work.
	for b-a >= 2*_PDQ_LEAF && maxDepth > 0 {
		maxDepth--
		mlo, mhi := doPivot(data, a, b)

		// Keep the larger side in this goroutine; hand off the smaller one.
		var lo, hi int
		if mlo-a < b-mhi {
			lo, hi, a = a, mlo, mhi
		} else {
			lo, hi, b = mhi, b, mlo
		}
		if hi-lo >= _PDQ_LEAF {
			if children == nil {
				// A call starts one goroutine per partitioning step, about
				// lg(n/_PDQ_LEAF) of them, so 64 is rarely reached. Past it, a finished
				// goroutine has already returned its token and only waits for the
				// deferred function above to receive its result.
				children = make(chan any, 64)
			}
			childCount++
			go func(lo, hi, depth int) {
				tokens <- struct{}{}
				children <- childPDQ(data, lo, hi, depth, tokens)
			}(lo, hi, maxDepth)
		} else if hi-lo > 1 {
			sort.Sort(subrange{data, lo, hi})
		}
	}
	if b-a > 1 {
		sort.Sort(subrange{data, a, b})
	}
	return nil
}

/*
childPDQ runs parallelPDQ in a goroutine started by parallelPDQ, and returns the value of
a panic instead of letting it stop the process. The caller's goroutine raises the panic
again without this goroutine's stack, so the stack is logged here.
*/
func childPDQ(data sort.Interface, a, b, maxDepth int, tokens chan struct{}) (panicked any) {
	defer func() {
		if r := recover(); r != nil {
			logging.Severef("sort: panic in a sort goroutine: %v\n%s", r, debug.Stack())
			panicked = r
		}
	}()
	return parallelPDQ(data, a, b, maxDepth, tokens)
}

/*
presorted reports whether data is already in increasing order, or was in decreasing
order and has been reversed. It makes at most n comparisons, and returns after a few
on unordered input.
*/
func presorted(data sort.Interface, n int) bool {
	i := 1
	for i < n && !data.Less(i, i-1) {
		i++
	}
	if i == n {
		return true
	}

	// data[0:i] is non-decreasing and data[i] < data[i-1], so the whole input can only
	// be non-increasing if data[0:i] is a run of equal elements.
	if data.Less(0, i-1) {
		return false
	}
	for i++; i < n && !data.Less(i-1, i); i++ {
	}
	if i < n {
		return false
	}
	for lo, hi := 0, n-1; lo < hi; lo, hi = lo+1, hi-1 {
		data.Swap(lo, hi)
	}
	return true
}

// depthLimit returns 2*ceil(lg(n+1)), the partitioning depth after which pdqsort takes over.
func depthLimit(n int) int {
	depth := 0
	for i := n; i > 0; i >>= 1 {
		depth++
	}
	return depth * 2
}

// subrange presents data[lo:hi] as a sort.Interface indexed from 0.
type subrange struct {
	data   sort.Interface
	lo, hi int
}

func (s subrange) Len() int           { return s.hi - s.lo }
func (s subrange) Less(i, j int) bool { return s.data.Less(s.lo+i, s.lo+j) }
func (s subrange) Swap(i, j int)      { s.data.Swap(s.lo+i, s.lo+j) }
