//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package sort

import (
	"math/rand"
	"runtime/debug"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
)

type ints []int

func (p ints) Len() int           { return len(p) }
func (p ints) Less(i, j int) bool { return p[i] < p[j] }
func (p ints) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func makeInts(n int, shape string, r *rand.Rand) []int {
	a := make([]int, n)
	for i := range a {
		switch shape {
		case "random":
			a[i] = r.Int()
		case "sorted":
			a[i] = i
		case "reversed":
			a[i] = n - i
		case "reversedTies":
			a[i] = (n - i) / 3
		case "equal":
			a[i] = 7
		case "fewUnique":
			a[i] = r.Intn(8)
		case "organPipe":
			a[i] = min(i, n-i)
		}
	}
	return a
}

// algorithms lists every Algorithm in this package; the tests and benchmarks run each one.
var algorithms = []Algorithm{PDQ{}, ParallelIntrosort{}, ParallelPDQ{}}

// Every algorithm runs the same cases. Run with -race to check the concurrency rules of
// Algorithm.
func TestAlgorithms(t *testing.T) {
	sizes := []int{0, 1, 2, 7, 8, 9, 255, 256, 257, 2047, 2048, 4095, 4096, 4097, 100_000}
	shapes := []string{"random", "sorted", "reversed", "reversedTies", "equal", "fewUnique", "organPipe"}
	for _, a := range algorithms {
		t.Run(a.Name(), func(t *testing.T) {
			r := rand.New(rand.NewSource(1))
			for _, n := range sizes {
				for _, shape := range shapes {
					got := makeInts(n, shape, r)
					want := slices.Clone(got)
					slices.Sort(want)
					a.Sort(ints(got))
					if !slices.Equal(got, want) {
						t.Fatalf("n=%d %s: not sorted", n, shape)
					}
					// ParallelPDQ also runs with fewer goroutines than GOMAXPROCS, so
					// that goroutines wait for one another's tokens.
					if _, ok := a.(ParallelPDQ); ok {
						for _, parallelism := range []int{1, 2, 3} {
							got = makeInts(n, shape, r)
							want = slices.Clone(got)
							slices.Sort(want)
							sortParallelPDQ(ints(got), parallelism)
							if !slices.Equal(got, want) {
								t.Fatalf("n=%d %s parallelism=%d: not sorted", n, shape, parallelism)
							}
						}
					}
				}
			}
		})
	}
}

func TestStrings(t *testing.T) {
	a := []string{"b", "c", "", "a"}
	Strings(a)
	if !slices.Equal(a, []string{"", "a", "b", "c"}) {
		t.Fatalf("Strings: got %q", a)
	}
}

// MB-25900: when the depth limit switches a range to heapsort, quickSort must still wait
// for the goroutines it started before it returns and closes their channel.
func TestParallelIntrosortDepthLimit(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for range 20 {
		src := makeInts(100_000, "random", r)
		for maxDepth := 1; maxDepth <= 4; maxDepth++ {
			got := slices.Clone(src)
			quickSort(ints(got), 0, len(got), maxDepth, nil)
			if !slices.IsSorted(got) {
				t.Fatalf("maxDepth=%d: not sorted", maxDepth)
			}
		}
	}
}

// Same as TestParallelIntrosortDepthLimit, for ParallelPDQ's handover to pdqsort.
func TestParallelPDQDepthLimit(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for range 20 {
		src := makeInts(100_000, "random", r)
		for maxDepth := 0; maxDepth <= 4; maxDepth++ {
			got := slices.Clone(src)
			tokens := make(chan struct{}, 3)
			tokens <- struct{}{}
			if p := parallelPDQ(ints(got), 0, len(got), maxDepth, tokens); p != nil {
				t.Fatalf("maxDepth=%d: panic %v", maxDepth, p)
			}
			if !slices.IsSorted(got) {
				t.Fatalf("maxDepth=%d: not sorted", maxDepth)
			}
		}
	}
}

// panicky panics in Less once it has been called limit times, from whichever goroutine
// makes that call.
type panicky struct {
	ints
	calls *atomic.Int64
	limit int64
}

func (p panicky) Less(i, j int) bool {
	if p.calls.Add(1) == p.limit {
		panic("less failed")
	}
	return p.ints.Less(i, j)
}

/*
A panic in one of ParallelPDQ's goroutines must reach the caller's goroutine; if it did
not, it would stop the test binary. A panic in the caller's goroutine must keep the stack
of the Less call that raised it, which Context.Recover logs: limits 1 and 1,000 are
reached in presorted and in the first doPivot, before any goroutine starts.
*/
func TestParallelPDQPanic(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for _, tc := range []struct {
		limit    int64
		inCaller bool
	}{{1, true}, {1_000, true}, {250_000, false}, {1_000_000, false}} {
		data := panicky{ints(makeInts(100_000, "random", r)), new(atomic.Int64), tc.limit}
		func() {
			defer func() {
				if p := recover(); p != "less failed" {
					t.Errorf("limit=%d: recovered %v, want the Less panic", tc.limit, p)
				} else if stack := string(debug.Stack()); tc.inCaller && !strings.Contains(stack, "panicky.Less") {
					t.Errorf("limit=%d: the stack lost the Less call:\n%s", tc.limit, stack)
				}
			}()
			ParallelPDQ{}.Sort(data)
		}()
	}
}

// ParallelPDQ never has more goroutines sorting at once than it has tokens.
func TestParallelPDQLimit(t *testing.T) {
	for _, parallelism := range []int{1, 2, 4} {
		var active, peak atomic.Int64
		data := counting{ints(makeInts(100_000, "random", rand.New(rand.NewSource(5)))), &active, &peak}
		sortParallelPDQ(data, parallelism)
		if !sort.IsSorted(data.ints) {
			t.Fatalf("parallelism=%d: not sorted", parallelism)
		}
		if peak.Load() > int64(parallelism) {
			t.Fatalf("parallelism=%d: %d goroutines called Less at once", parallelism, peak.Load())
		}
	}
}

// counting records the largest number of goroutines that were in Less at the same time.
type counting struct {
	ints
	active, peak *atomic.Int64
}

func (c counting) Less(i, j int) bool {
	n := c.active.Add(1)
	defer c.active.Add(-1)
	for p := c.peak.Load(); n > p && !c.peak.CompareAndSwap(p, n); p = c.peak.Load() {
	}
	return c.ints.Less(i, j)
}

func BenchmarkAlgorithms(b *testing.B) {
	for _, shape := range []string{"random", "sorted", "reversed"} {
		src := makeInts(200_000, shape, rand.New(rand.NewSource(1)))
		work := make([]int, len(src))
		for _, a := range algorithms {
			b.Run(shape+"/"+a.Name(), func(b *testing.B) {
				for b.Loop() {
					copy(work, src)
					a.Sort(ints(work))
				}
			})
		}
	}
}
