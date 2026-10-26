//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"fmt"
	"testing"
)

const _BENCH_ROWS = 1024

// Rows of nFields string fields; every row appears twice, so half of the
// rows offered to the set are duplicates.
func benchDistinctRows(nFields int) []Value {
	rows := make([]Value, 0, _BENCH_ROWS)
	for i := 0; len(rows) < _BENCH_ROWS; i++ {
		obj := make(map[string]interface{}, nFields)
		for f := 0; f < nFields; f++ {
			obj[fmt.Sprintf("f%02d", f)] = fmt.Sprintf("v%d_%d", i, f)
		}
		v := NewValue(obj)
		rows = append(rows, v, v)
	}
	return rows
}

func benchDistinct(b *testing.B, nFields int, once bool) {
	rows := benchDistinctRows(nFields)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		set := NewSet(_BENCH_ROWS, false, false)
		i := 0
		for pb.Next() {
			row := rows[i]
			if once {
				set.PutIfAbsent(row, row)
			} else if !set.Has(row) {
				set.Put(row, row)
			}
			i++
			if i == len(rows) {
				i = 0
				set = NewSet(_BENCH_ROWS, false, false)
			}
		}
	})
}

func BenchmarkDistinctHasPut5(b *testing.B)       { benchDistinct(b, 5, false) }
func BenchmarkDistinctPutIfAbsent5(b *testing.B)  { benchDistinct(b, 5, true) }
func BenchmarkDistinctHasPut40(b *testing.B)      { benchDistinct(b, 40, false) }
func BenchmarkDistinctPutIfAbsent40(b *testing.B) { benchDistinct(b, 40, true) }

func TestSetPutIfAbsent(t *testing.T) {
	for _, collect := range []bool{false, true} {
		set := NewSet(4, collect, false)
		obj := NewValue(map[string]interface{}{"a": "x", "b": 1})
		arr := NewValue([]interface{}{"x", 1})
		for _, v := range []Value{obj, arr, NewValue("s"), NewValue(1), NULL_VALUE, MISSING_VALUE} {
			if !set.PutIfAbsent(v, v) {
				t.Errorf("collect=%v: first PutIfAbsent(%v) = false", collect, v)
			}
			if set.PutIfAbsent(v, v) {
				t.Errorf("collect=%v: second PutIfAbsent(%v) = true", collect, v)
			}
			if !set.Has(v) {
				t.Errorf("collect=%v: Has(%v) = false after PutIfAbsent", collect, v)
			}
		}
		if set.Len() != 6 || (collect && len(set.Values()) != 6) {
			t.Errorf("collect=%v: Len = %d", collect, set.Len())
		}
	}
}

func TestSetMerge(t *testing.T) {
	a := NewSet(4, true, false)
	b := NewSet(4, true, false)
	shared := NewValue(map[string]interface{}{"a": "x"})
	for _, v := range []Value{shared, NewValue([]interface{}{1}), NewValue("s"), NewValue(1), NULL_VALUE} {
		a.Add(v)
	}
	for _, v := range []Value{shared, NewValue([]interface{}{2}), NewValue("t"), NewValue(1.5),
		NewValue(true), MISSING_VALUE} {
		b.Add(v)
	}
	a.Merge(b)
	if a.Len() != 10 || len(a.Values()) != 10 {
		t.Errorf("Len = %d, Values = %d; want 10", a.Len(), len(a.Values()))
	}
	for _, v := range b.Values() {
		if !a.Has(v) {
			t.Errorf("Has(%v) = false after Merge", v)
		}
	}
	empty := NewSet(4, true, false)
	empty.Merge(a)
	if empty.Len() != a.Len() {
		t.Errorf("Merge into empty: Len = %d; want %d", empty.Len(), a.Len())
	}
}

func TestMultiSetRemoveIfPresent(t *testing.T) {
	ms := NewMultiSet(4, false, false)
	obj := NewValue(map[string]interface{}{"a": "x"})
	arr := NewValue([]interface{}{1, "x"})
	for _, v := range []Value{obj, obj, arr, NewValue("s")} {
		ms.Add(v)
	}
	for _, c := range []struct {
		v    Value
		want bool
	}{{obj, true}, {obj, true}, {obj, false}, {arr, true}, {arr, false},
		{NewValue("s"), true}, {NewValue("s"), false}, {NewValue(1), false}} {
		if got := ms.RemoveIfPresent(c.v); got != c.want {
			t.Errorf("RemoveIfPresent(%v) = %v; want %v", c.v, got, c.want)
		}
	}
	if ms.Len() != 0 {
		t.Errorf("Len = %d; want 0", ms.Len())
	}
}
