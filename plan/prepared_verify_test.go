//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"sync"
	"testing"

	"github.com/couchbase/query/algebra"
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
	decoded, _, er := NewPreparedFromEncodedPlan(encoded, false)
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
