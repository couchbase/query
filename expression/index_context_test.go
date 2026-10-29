/*
Copyright 2026-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package expression

import (
	"fmt"
	"sync"
	"testing"

	"github.com/couchbase/query/value"
)

var _ InlistContext = (*IndexContext)(nil)

// in-list of n values 1000..1000+n-1, followed by extra
func inlistConstant(n int, extra ...any) Expression {
	vals := make([]any, 0, n+len(extra))
	for i := 0; i < n; i++ {
		vals = append(vals, 1000+i)
	}
	vals = append(vals, extra...)
	return NewConstant(vals)
}

func inlistItems() []value.Value {
	return []value.Value{
		value.NewValue(map[string]any{"c1": 1000}),
		value.NewValue(map[string]any{"c1": 1014}),
		value.NewValue(map[string]any{"c1": 1015}),
		value.NewValue(map[string]any{"c1": 5}),
		value.NewValue(map[string]any{"c1": "1000"}),
		value.NewValue(map[string]any{"c1": nil}),
		value.NewValue(map[string]any{"c2": 1}),
	}
}

func TestIndexContextInlistHash(t *testing.T) {
	for _, n := range []int{_INLIST_HASH_THRESHOLD - 1, _INLIST_HASH_THRESHOLD, 100} {
		for _, withNull := range []bool{false, true} {
			var list Expression
			if withNull {
				list = inlistConstant(n-1, nil)
			} else {
				list = inlistConstant(n)
			}
			in := NewIn(NewIdentifier("c1"), list).(*In)
			notIn := NewNot(NewIn(NewIdentifier("c1"), list))
			name := fmt.Sprintf("n=%d,null=%v", n, withNull)

			context := NewIndexContext()
			in.EnableInlistHash(context)
			notIn.EnableInlistHash(context)

			for _, item := range inlistItems() {
				for _, expr := range []Expression{in, notIn} {
					expected, err := expr.Evaluate(item, NewIndexContext())
					if err != nil {
						t.Fatalf("%s: %v", name, err)
					}
					actual, _, err := expr.EvaluateForIndex(item, context)
					if err != nil {
						t.Fatalf("%s: %v", name, err)
					}
					if !expected.EquivalentTo(actual) {
						t.Errorf("%s: %v on %v: expected %v, got %v", name, expr, item, expected, actual)
					}
				}
			}

			ih := context.(*IndexContext).GetInlistHash(in)
			if ih == nil {
				t.Fatalf("%s: in-list hash not registered", name)
			}
			if n >= _INLIST_HASH_THRESHOLD && ih.hashTab == nil {
				t.Errorf("%s: expected hash table to be built", name)
			} else if n < _INLIST_HASH_THRESHOLD && ih.hashTab != nil {
				t.Errorf("%s: expected no hash table below threshold", name)
			}

			in.ResetMemory(context)
			notIn.ResetMemory(context)
			if len(context.(*IndexContext).inlistHashMap) != 0 {
				t.Errorf("%s: in-list hash not released", name)
			}
		}
	}
}

func TestIndexContextInlistHashNested(t *testing.T) {
	c1 := NewIdentifier("c1")
	in1 := NewIn(c1, inlistConstant(20)).(*In)
	in2 := NewIn(c1, inlistConstant(30)).(*In)
	in3 := NewIn(c1, inlistConstant(40)).(*In)
	cond := NewOr(
		NewAnd(NewNot(in1), NewIsNotNull(c1)),
		NewSearchedCase(WhenTerms{&WhenTerm{When: in2, Then: NewConstant(true)}}, in3),
	)

	context := NewIndexContext().(*IndexContext)
	cond.EnableInlistHash(context)
	for _, in := range []*In{in1, in2, in3} {
		if context.GetInlistHash(in) == nil {
			t.Errorf("in-list hash not registered for %v", in)
		}
	}

	for _, item := range inlistItems() {
		expected, err := cond.Evaluate(item, NewIndexContext())
		if err != nil {
			t.Fatal(err)
		}
		actual, _, err := cond.EvaluateForIndex(item, context)
		if err != nil {
			t.Fatal(err)
		}
		if !expected.EquivalentTo(actual) {
			t.Errorf("%v on %v: expected %v, got %v", cond, item, expected, actual)
		}
	}

	cond.ResetMemory(context)
	if len(context.inlistHashMap) != 0 {
		t.Errorf("in-list hash not released: %v", context.inlistHashMap)
	}
}

// concurrent evaluations sharing one context (and thus one hash table),
// run with -race to check the hash table is built once and read without races
func TestIndexContextInlistHashConcurrent(t *testing.T) {
	list := inlistConstant(99, nil)
	in := NewIn(NewIdentifier("c1"), list).(*In)
	notIn := NewNot(NewIn(NewIdentifier("c1"), list))
	items := inlistItems()

	expected := make([][]value.Value, len(items))
	for i, item := range items {
		for _, expr := range []Expression{in, notIn} {
			v, err := expr.Evaluate(item, NewIndexContext())
			if err != nil {
				t.Fatal(err)
			}
			expected[i] = append(expected[i], v)
		}
	}

	context := NewIndexContext().(*IndexContext)
	const workers = 8
	for w := 0; w < workers; w++ {
		in.EnableInlistHash(context)
		notIn.EnableInlistHash(context)
	}

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := 0; r < 100; r++ {
				for i, item := range items {
					for j, expr := range []Expression{in, notIn} {
						actual, _, err := expr.EvaluateForIndex(item, context)
						if err != nil {
							errs <- err
							return
						}
						if !expected[i][j].EquivalentTo(actual) {
							errs <- fmt.Errorf("%v on %v: expected %v, got %v", expr, item, expected[i][j], actual)
							return
						}
					}
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	ih := context.GetInlistHash(in)
	if ih == nil || ih.hashTab == nil {
		t.Fatalf("expected hash table to be built")
	}
	// the hash table is kept until the last instance is done
	for w := 0; w < workers; w++ {
		if context.GetInlistHash(in) != ih || ih.hashTab == nil {
			t.Fatalf("in-list hash released with %d instances remaining", workers-w)
		}
		in.ResetMemory(context)
		notIn.ResetMemory(context)
	}
	if context.GetInlistHash(in) != nil || ih.hashTab != nil {
		t.Errorf("in-list hash not released")
	}
	if context.inlistHashOn.Load() {
		t.Errorf("in-list hash flag not cleared")
	}
}

func BenchmarkIndexContextInlist(b *testing.B) {
	item := value.NewValue(map[string]any{"c1": 5})
	for _, n := range []int{100, 1000} {
		in := NewIn(NewIdentifier("c1"), inlistConstant(n))
		b.Run(fmt.Sprintf("linear/%d", n), func(b *testing.B) {
			context := NewIndexContext()
			for i := 0; i < b.N; i++ {
				in.EvaluateForIndex(item, context)
			}
		})
		b.Run(fmt.Sprintf("hash/%d", n), func(b *testing.B) {
			context := NewIndexContext()
			in.EnableInlistHash(context)
			defer in.ResetMemory(context)
			for i := 0; i < b.N; i++ {
				in.EvaluateForIndex(item, context)
			}
		})
	}
}

func BenchmarkIndexContextInlistParallel(b *testing.B) {
	item := value.NewValue(map[string]any{"c1": 5})
	for _, n := range []int{_INLIST_HASH_THRESHOLD - 1, 100} {
		in := NewIn(NewIdentifier("c1"), inlistConstant(n))
		b.Run(fmt.Sprintf("disabled/%d", n), func(b *testing.B) {
			context := NewIndexContext()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					in.EvaluateForIndex(item, context)
				}
			})
		})
		b.Run(fmt.Sprintf("enabled/%d", n), func(b *testing.B) {
			context := NewIndexContext()
			in.EnableInlistHash(context)
			defer in.ResetMemory(context)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					in.EvaluateForIndex(item, context)
				}
			})
		})
	}
}
