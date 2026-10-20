// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sort

import "sort"

/*
ParallelIntrosort is an introsort (quicksort with median-of-three pivots, heapsort once
recursion gets too deep, insertion sort for ranges of 7 or fewer elements) that sorts
every partition of 256 or more elements in its own goroutine. It was forked in 2014 from
the Go standard library's sort, which replaced this algorithm with pdqsort in Go 1.19.

A panic in one of the goroutines it starts is not recovered and stops the process.
*/
type ParallelIntrosort struct{}

func (ParallelIntrosort) Name() string {
	return "parallel-introsort"
}

/*
Sort makes one call to data.Len to determine n, and O(n*log(n)) calls to data.Less
and data.Swap. The sort is not guaranteed to be stable.
*/
func (ParallelIntrosort) Sort(data sort.Interface) {
	// Switch to heapsort if depth of 2*ceil(lg(n+1)) is reached.
	n := data.Len()
	maxDepth := 0
	for i := n; i > 0; i >>= 1 {
		maxDepth++
	}
	maxDepth *= 2
	quickSort(data, 0, n, maxDepth, nil)
}

// Insertion sort
func insertionSort(data sort.Interface, a, b int) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && data.Less(j, j-1); j-- {
			data.Swap(j, j-1)
		}
	}
}

// siftDown implements the heap property on data[lo, hi).
// first is an offset into the array where the root of the heap lies.
func siftDown(data sort.Interface, lo, hi, first int) {
	root := lo
	for {
		child := 2*root + 1
		if child >= hi {
			break
		}
		if child+1 < hi && data.Less(first+child, first+child+1) {
			child++
		}
		if !data.Less(first+root, first+child) {
			return
		}
		data.Swap(first+root, first+child)
		root = child
	}
}

func heapSort(data sort.Interface, a, b int) {
	first := a
	lo := 0
	hi := b - a

	// Build heap with greatest element at top.
	for i := (hi - 1) / 2; i >= 0; i-- {
		siftDown(data, i, hi, first)
	}

	// Pop elements, largest first, into end of data.
	for i := hi - 1; i >= 0; i-- {
		data.Swap(first, first+i)
		siftDown(data, lo, i, first)
	}
}

// Quicksort, following Bentley and McIlroy,
// ``Engineering a Sort Function,'' SP&E November 1993.

// medianOfThree moves the median of the three values data[a], data[b], data[c] into data[a].
func medianOfThree(data sort.Interface, a, b, c int) {
	m0 := b
	m1 := a
	m2 := c
	// bubble sort on 3 elements
	if data.Less(m1, m0) {
		data.Swap(m1, m0)
	}
	if data.Less(m2, m1) {
		data.Swap(m2, m1)
	}
	if data.Less(m1, m0) {
		data.Swap(m1, m0)
	}
	// now data[m0] <= data[m1] <= data[m2]
}

func swapRange(data sort.Interface, a, b, n int) {
	for i := 0; i < n; i++ {
		data.Swap(a+i, b+i)
	}
}

func doPivot(data sort.Interface, lo, hi int) (midlo, midhi int) {
	m := lo + (hi-lo)/2 // Written like this to avoid integer overflow.
	if hi-lo > 40 {
		// Tukey's ``Ninther,'' median of three medians of three.
		s := (hi - lo) / 8
		medianOfThree(data, lo, lo+s, lo+2*s)
		medianOfThree(data, m, m-s, m+s)
		medianOfThree(data, hi-1, hi-1-s, hi-1-2*s)
	}
	medianOfThree(data, lo, m, hi-1)

	// Invariants are:
	//	data[lo] = pivot (set up by ChoosePivot)
	//	data[lo <= i < a] = pivot
	//	data[a <= i < b] < pivot
	//	data[b <= i < c] is unexamined
	//	data[c <= i < d] > pivot
	//	data[d <= i < hi] = pivot
	//
	// Once b meets c, can swap the "= pivot" sections
	// into the middle of the slice.
	pivot := lo
	a, b, c, d := lo+1, lo+1, hi, hi
	for {
		for b < c {
			if data.Less(b, pivot) { // data[b] < pivot
				b++
			} else if !data.Less(pivot, b) { // data[b] = pivot
				data.Swap(a, b)
				a++
				b++
			} else {
				break
			}
		}
		for b < c {
			if data.Less(pivot, c-1) { // data[c-1] > pivot
				c--
			} else if !data.Less(c-1, pivot) { // data[c-1] = pivot
				data.Swap(c-1, d-1)
				c--
				d--
			} else {
				break
			}
		}
		if b >= c {
			break
		}
		// data[b] > pivot; data[c-1] < pivot
		data.Swap(b, c-1)
		b++
		c--
	}

	n := min(b-a, a-lo)
	swapRange(data, lo, b-n, n)

	n = min(hi-d, d-c)
	swapRange(data, c, hi-n, n)

	return lo + b - a, hi - (d - c)
}

type childChannel chan bool

const _PTHRESH = 256

func quickSort(data sort.Interface, a, b, maxDepth int, parent childChannel) {
	defer notify(parent)

	var children childChannel
	childCount := 0

	if b-a >= _PTHRESH {
		children = make(childChannel, 64)
		defer close(children)
	}

	for b-a > 7 {
		if maxDepth == 0 {
			heapSort(data, a, b)
			// MB-25900 await children
			for ; childCount > 0; childCount-- {
				<-children
			}
			return
		}
		maxDepth--
		mlo, mhi := doPivot(data, a, b)

		// Avoiding recursion on the larger subproblem guarantees
		// a stack depth of at most lg(b-a).
		if mlo-a < b-mhi {
			if (mlo < mhi) && (mlo-a >= _PTHRESH) {
				go quickSort(data, a, mlo, maxDepth, children)
				childCount++
			} else {
				quickSort(data, a, mlo, maxDepth, nil)
			}
			a = mhi // i.e., quickSort(data, mhi, b)
		} else {
			if (mlo < mhi) && (b-mhi >= _PTHRESH) {
				go quickSort(data, mhi, b, maxDepth, children)
				childCount++
			} else {
				quickSort(data, mhi, b, maxDepth, nil)
			}
			b = mlo // i.e., quickSort(data, a, mlo)
		}
	}

	// Await children
	for ; childCount > 0; childCount-- {
		<-children
	}

	if b-a > 1 {
		insertionSort(data, a, b)
	}
}

func notify(parent childChannel) {
	if parent != nil {
		parent <- false
	}
}
