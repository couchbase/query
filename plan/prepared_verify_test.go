//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"strings"
	"sync"
	"testing"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// MB-75478: the verification error indicators are not saved with the plan (encoded_plan); SAVE is
func TestVerificationErrorNotEncoded(t *testing.T) {
	prepared := NewPrepared(NewSequence(NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
	prepared.SetName("p1")
	prepared.SetText("SELECT 1")
	prepared.SetPersist(true)
	prepared.SetFatalError(true)
	prepared.SetErrorCount(3)

	encoded, err := prepared.BuildEncodedPlan()
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	decoded, _, er := NewPreparedFromEncodedPlan(encoded, false, nil)
	if er != nil {
		t.Fatalf("decoding: %v", er)
	}
	if decoded.HasVerificationError() || decoded.HasFatalError() || decoded.ErrorCount() != 0 {
		t.Errorf("decoded plan has a verification error: fatal %v, count %d", decoded.HasFatalError(),
			decoded.ErrorCount())
	}
	if !decoded.Persist() {
		t.Errorf("decoded plan lost SAVE")
	}
	if !prepared.HasFatalError() || prepared.ErrorCount() != 3 {
		t.Errorf("the plan lost its verification error: fatal %v, count %d", prepared.HasFatalError(),
			prepared.ErrorCount())
	}
}

// MB-75478: concurrent executions update and read the verification error indicators, and SAVE, of a cached
// plan (run with -race)
func TestPreparedConcurrentAccess(t *testing.T) {
	prepared := NewPrepared(nil, nil, nil, nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				prepared.errCount.Add(1)
				_ = prepared.HasVerificationError()
				_ = prepared.ErrorCount()
				prepared.SetPersist(j%2 == 0)
				_ = prepared.Persist()
				if i == 0 && j == 500 {
					prepared.SetFatalError(true)
				}
			}
		}(i)
	}
	wg.Wait()
	if prepared.ErrorCount() != 8000 || !prepared.HasFatalError() {
		t.Errorf("error count %d, fatal %v", prepared.ErrorCount(), prepared.HasFatalError())
	}
}

// MB-75480: a UDF prepared entry is for the UDF body (before planning) it was generated for
func TestIsUdfExpr(t *testing.T) {
	body, err := parser.Parse("(SELECT RAW 1 FROM b AS d WHERE d.a = 1)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// planning changes the expression (e.g. covered expressions): the body before planning is recorded
	planned, err := parser.Parse("(SELECT RAW 1 FROM b AS d WHERE d.a = 1 AND true)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	prepared := NewPrepared(nil, nil, nil, nil, nil)
	prepared.SetName("__INLINE_UDF__default:f1")
	prepared.SetInlineUdf()
	prepared.SetUdfText(body.String())
	if err = prepared.SetupUdfSubqPlans(planned, algebra.NewSubqueryPlans(), false); err != nil {
		t.Fatalf("setup: %v", err)
	}

	for _, tc := range []struct {
		body string
		same bool
	}{
		{"(SELECT RAW 1 FROM b AS d WHERE d.a = 1)", true},
		{"(select raw 1 from b as d where d.a=1)", true}, // same body, written differently
		{"(SELECT RAW 2 FROM b AS d WHERE d.a = 1)", false},
		{"(SELECT RAW 1 FROM b AS d WHERE d.a = 1 AND true)", false}, // the planned expression is not the body
	} {
		expr, err := parser.Parse(tc.body)
		if err != nil {
			t.Fatalf("%s: parse: %v", tc.body, err)
		}
		same, err := prepared.IsUdfExpr(expr)
		if err != nil || same != tc.same {
			t.Errorf("%s: same %v, error %v; expected same %v", tc.body, same, err, tc.same)
		}
	}

	// an entry without the body (none saved) is not for the body
	noText := NewPrepared(nil, nil, nil, nil, nil)
	if err = noText.SetupUdfSubqPlans(body, algebra.NewSubqueryPlans(), false); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if same, err := noText.IsUdfExpr(body); same || err != nil {
		t.Errorf("no body: same %v, error %v", same, err)
	}

	// an entry that can not be read is not for the body
	bad := NewPrepared(nil, nil, nil, nil, nil)
	if same, err := bad.IsUdfExpr(body); same || err == nil {
		t.Errorf("no subquery plans: same %v, error %v", same, err)
	}
}

// MB-75515: the subquery plans of an inline UDF entry are set for the subqueries of the expression of the entry
// (the expression to run), not of another parse of the subqueries
func TestGetUdfSubqPlans(t *testing.T) {
	expr, err := parser.Parse("(SELECT RAW 1 FROM b AS d) + (SELECT RAW 1 FROM b AS d) + (SELECT RAW 2 FROM b AS e)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	subqueries, err := expression.ListSubqueries(expression.Expressions{expr}, true)
	if err != nil || len(subqueries) != 3 {
		t.Fatalf("subqueries: %v, %v", subqueries, err)
	}
	// an entry with plans for the subqueries for which include holds, and for only one of identical subqueries
	// if once
	newEntry := func(include func(string) bool, once bool) *Prepared {
		subqPlans := algebra.NewSubqueryPlans()
		seen := make(map[string]bool)
		for _, sq := range subqueries {
			text := sq.String()
			if !include(text) || (once && seen[text]) {
				continue
			}
			seen[text] = true
			subqPlans.Set(sq.(*algebra.Subquery).Select(), expr, NewQueryPlan(NewSequence(NewDummyScan(0, 0, 0, 0))),
				nil, false)
		}
		prepared := NewPrepared(nil, nil, nil, nil, nil)
		prepared.SetName("__INLINE_UDF__default:f1")
		prepared.SetInlineUdf()
		prepared.SetUdfText(expr.String())
		if err := prepared.SetupUdfSubqPlans(expr, subqPlans, false); err != nil {
			t.Fatalf("setup: %v", err)
		}
		return prepared
	}

	// plans for all the subqueries (two are identical, and share a plan)
	all := func(string) bool { return true }
	prepared := newEntry(all, false)
	prepared.SetUdfText("")
	loaded := algebra.NewSubqueryPlans()
	if err = prepared.GetUdfSubqPlans(false, loaded); err != nil {
		t.Fatalf("get: %v", err)
	}
	loadedExpr := loaded.GetExpression(false)
	if loadedExpr == nil || loadedExpr == expr || loadedExpr.String() != expr.String() {
		t.Fatalf("expected the expression of the entry, parsed: %v", loadedExpr)
	}
	loadedSubqueries, err := expression.ListSubqueries(expression.Expressions{loadedExpr}, true)
	if err != nil || len(loadedSubqueries) != 3 {
		t.Fatalf("subqueries of the expression of the entry: %v, %v", loadedSubqueries, err)
	}
	for i, sq := range loadedSubqueries {
		if _, plan, _, ok := loaded.Get(sq.(*algebra.Subquery).Select(), false); !ok || plan == nil {
			t.Errorf("no plan for subquery %d of the expression of the entry: %s", i, sq)
		}
	}
	for i, sq := range subqueries {
		if _, _, _, ok := loaded.Get(sq.(*algebra.Subquery).Select(), false); ok {
			t.Errorf("plan for subquery %d of another expression: %s", i, sq)
		}
	}
	// the body the plans are for is kept, e.g. to save the entry again
	if same, err := prepared.IsUdfExpr(expr); !same || err != nil {
		t.Errorf("expected the body of the entry kept: %v, %v", same, err)
	}

	// a subquery of the expression without a plan: the entry does not match
	prepared = newEntry(all, true)
	if err = prepared.GetUdfSubqPlans(false, algebra.NewSubqueryPlans()); err != nil {
		t.Errorf("identical subqueries share a plan: %v", err)
	}
	prepared = newEntry(func(text string) bool { return !strings.Contains(text, "raw 2") }, false)
	if err = prepared.GetUdfSubqPlans(false, algebra.NewSubqueryPlans()); !IsUdfEntryMismatch(err) {
		t.Errorf("expected the entry not to match its expression, got %v", err)
	}
}
