//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"slices"
	gosort "sort"
	"testing"

	"github.com/couchbase/query/sort"
)

// countingSort counts the sorts it is asked to do.
type countingSort struct {
	sort.Algorithm
	calls *int
}

func (this countingSort) Sort(data gosort.Interface) {
	*this.calls++
	this.Algorithm.Sort(data)
}

// The algorithm set with SetSortAlgorithm decides the order of values with equal keys, and
// Copy keeps it.
func TestAnnotatedArraySortAlgorithm(t *testing.T) {
	acquire := func(size int) AnnotatedValues { return make(AnnotatedValues, 0, size) }
	key := func(v AnnotatedValue) int { return v.GetValue().Actual().(map[string]any)["k"].(int) }
	lessThan := func(v1, v2 AnnotatedValue) bool { return key(v1) < key(v2) }

	// 50 values with 3 distinct keys, enough for the algorithms to order ties differently.
	values := make(AnnotatedValues, 50)
	for i := range values {
		values[i] = NewAnnotatedValue(map[string]any{"k": (i * 7) % 3})
		values[i].SetId(i)
	}
	ids := func(vals AnnotatedValues) []int {
		rv := make([]int, len(vals))
		for i, v := range vals {
			rv[i] = v.GetId().(int)
		}
		return rv
	}

	var orders [][]int
	for _, a := range []sort.Algorithm{sort.ParallelIntrosort{}, sort.ParallelPDQ{}} {
		want := append(AnnotatedValues(nil), values...)
		a.Sort(&AnnotatedArray{mem: want, less: lessThan})
		orders = append(orders, ids(want))

		calls := 0
		array := NewAnnotatedArray(acquire, nil, nil, nil, lessThan, false, nil)
		array.SetSortAlgorithm(countingSort{a, &calls})
		for _, arr := range []*AnnotatedArray{array, array.Copy()} {
			for _, v := range values {
				arr.Append(v)
			}
			var got AnnotatedValues
			arr.Foreach(func(v AnnotatedValue) bool {
				got = append(got, v)
				return true
			})
			if g, w := ids(got), ids(want); !slices.Equal(g, w) {
				t.Errorf("%s: got %v, want %v", a.Name(), g, w)
			}
		}
		if calls != 2 {
			t.Errorf("%s: the algorithm sorted %d times, want 2", a.Name(), calls)
		}
	}
	if slices.Equal(orders[0], orders[1]) {
		t.Errorf("the algorithms put equal keys in the same order, so the test checks nothing")
	}
}
