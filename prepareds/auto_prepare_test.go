//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"testing"

	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

// MB-75435: an auto-prepared plan carries the statement type, which later executions of the statement
// take from the cached plan
func TestAutoPreparePlanType(t *testing.T) {
	defer func(m func() settings.PlanStabilityMode) {
		planStabilityModeFn = m
	}(planStabilityModeFn)
	planStabilityModeFn = func() settings.PlanStabilityMode { return settings.PS_MODE_OFF }

	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)

	for _, tc := range []struct{ name, text, typ string }{
		{"ap1", "SELECT 1 AS one FROM b USE KEYS \"x\"", "SELECT"},
		{"ap2", "DELETE FROM b USE KEYS \"x\"", "DELETE"},
		{"ap3", "UPDATE b USE KEYS \"x\" SET a = 1", "UPDATE"},
	} {
		stmt, err := n1ql.ParseStatement2(tc.text, "default", "")
		if err != nil {
			t.Fatalf("%s: parse error: %v", tc.text, err)
		}
		prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
		prepared.SetName(tc.name)
		prepared.SetText(tc.text)
		if added, err := AddAutoPreparePlan(stmt, prepared, settings.PS_MODE_OFF); err != nil || !added {
			t.Fatalf("%s: not added (%v)", tc.text, err)
		}
		ce := prepareds.get(encodeName(tc.name, ""), false)
		if ce == nil {
			t.Fatalf("%s: not in the cache", tc.text)
		}
		if ce.Prepared.Type() != tc.typ {
			t.Errorf("%s: cached plan type %q, expected %q", tc.text, ce.Prepared.Type(), tc.typ)
		}
	}
}
