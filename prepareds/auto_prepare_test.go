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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
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
		if added, err := AddAutoPreparePlan(stmt, prepared, settings.PS_MODE_OFF, nil); err != nil || !added {
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

// MB-75514 test hooks: saves, saved plan lookups (by name), the mode, verification and re-planning
type autoPrepareHooks struct {
	saves      int
	loads      []string
	failVerify map[*plan.Prepared]bool
	replans    int
}

func setAutoPrepareHooks(t *testing.T, mode settings.PlanStabilityMode) *autoPrepareHooks {
	h := &autoPrepareHooks{failVerify: make(map[*plan.Prepared]bool)}
	p, l, m, v, r := persistPreparedFn, loadPreparedFn, planStabilityModeFn, verifyPreparedFn, reprepareFn
	t.Cleanup(func() {
		persistPreparedFn, loadPreparedFn, planStabilityModeFn, verifyPreparedFn, reprepareFn = p, l, m, v, r
	})
	persistPreparedFn = func(*plan.Prepared) errors.Error {
		h.saves++
		return nil
	}
	loadPreparedFn = func(name string, _ settings.PlanStabilityMode, _ settings.PlanStabilityErrorPolicy) (
		*plan.Prepared, errors.Error) {
		h.loads = append(h.loads, name)
		return nil, nil
	}
	planStabilityModeFn = func() settings.PlanStabilityMode { return mode }
	verifyPreparedFn = func(prepared *plan.Prepared) errors.Error {
		if h.failVerify[prepared] {
			return errors.NewPlanVerificationError("index not found", nil)
		}
		return nil
	}
	reprepareFn = func(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
		planStability bool, planStabilityMode settings.PlanStabilityMode,
		planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log,
		scanConsistency datastore.ScanConsistency, encode bool) (*plan.Prepared, errors.Error) {
		h.replans++
		pl := plan.NewPrepared(nil, nil, nil, nil, nil)
		pl.SetName(prepared.Name())
		pl.SetText(prepared.Text())
		pl.SetNamespace(prepared.Namespace())
		pl.SetQueryContext(prepared.QueryContext())
		pl.SetAutoPrepared(prepared.AutoPrepared())
		pl.SetAdHoc(prepared.AdHoc())
		pl.SetPersist(prepared.Persist())
		pl.SetPreparedTime(time.Now())
		return pl, nil
	}
	return h
}

const autoPrepareText = "SELECT 1 AS one FROM b USE KEYS \"x\""

// an ad hoc statement's plan, as built for a request
func autoPreparePlan(name, queryContext string, adHoc bool) *plan.Prepared {
	prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
	prepared.SetName(name)
	prepared.SetText(autoPrepareText)
	prepared.SetNamespace("default")
	prepared.SetQueryContext(queryContext)
	prepared.SetAdHoc(adHoc)
	prepared.SetPreparedTime(time.Now())
	return prepared
}

func autoPrepareContext(queryContext string, mode settings.PlanStabilityMode,
	policy settings.PlanStabilityErrorPolicy) *planner.PrepareContext {
	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, "req1", queryContext, nil, nil, 0, 0, false, false, nil, nil, nil, true,
		mode, policy, datastore.UNBOUNDED)
	return &prepContext
}

// MB-75514: an ad hoc statement run with a query context finds its plan, cached and saved under its name and the
// query context
func TestAutoPrepareQueryContext(t *testing.T) {
	h := setAutoPrepareHooks(t, settings.PS_MODE_AD_HOC)
	PreparedsInit(16)

	stmt, err := n1ql.ParseStatement2(autoPrepareText, "default", "")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	const queryContext = "default:b.s"
	prepContext := autoPrepareContext(queryContext, settings.PS_MODE_AD_HOC, settings.PS_ERROR_MODERATE)
	name := GetAutoPrepareName(autoPrepareText, prepContext)
	fullName := encodeName(name, queryContext)

	// not cached: the saved plan is looked for under the name and the query context
	if prep, err := GetAutoPreparePlan(name, autoPrepareText, "default", prepContext); prep != nil || err != nil {
		t.Fatalf("not cached: got %v, %v", prep, err)
	}
	if len(h.loads) != 1 || h.loads[0] != fullName {
		t.Errorf("expected the saved plan looked for under %s, got %v", fullName, h.loads)
	}

	// cached (and saved): found
	prepared := autoPreparePlan(name, queryContext, true)
	if added, err := AddAutoPreparePlan(stmt, prepared, settings.PS_MODE_AD_HOC, nil); !added || err != nil {
		t.Fatalf("not added: %v", err)
	}
	if h.saves != 1 {
		t.Errorf("expected the plan saved, %d saves", h.saves)
	}
	h.loads = nil
	if prep, err := GetAutoPreparePlan(name, autoPrepareText, "default", prepContext); prep != prepared ||
		err != nil || len(h.loads) != 0 {
		t.Errorf("cached: got %p, %v, loads %v; expected the cached plan %p", prep, err, h.loads, prepared)
	}

	// the statistics of the cache entry are recorded
	RecordPreparedMetrics(prepared, time.Second, time.Second)
	if ce := prepareds.get(fullName, false); ce == nil || ce.ServiceTime == 0 {
		t.Errorf("expected the statistics of %s recorded", fullName)
	}
}

// MB-75514: a cached auto prepared statement that is not ad hoc (not saved) is not saved when it is re-planned; a
// saved ad hoc statement is (MB-75479)
func TestAutoPrepareReplan(t *testing.T) {
	for _, adHoc := range []bool{false, true} {
		h := setAutoPrepareHooks(t, settings.PS_MODE_AD_HOC_READ_ONLY)
		PreparedsInit(16)

		stmt, err := n1ql.ParseStatement2(autoPrepareText, "default", "")
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		prepContext := autoPrepareContext("", settings.PS_MODE_AD_HOC_READ_ONLY, settings.PS_ERROR_FLEXIBLE)
		name := GetAutoPrepareName(autoPrepareText, prepContext)
		prepared := autoPreparePlan(name, "", adHoc)
		if added, err := AddAutoPreparePlan(stmt, prepared, settings.PS_MODE_AD_HOC_READ_ONLY, nil); !added ||
			err != nil {
			t.Fatalf("ad hoc %v: not added: %v", adHoc, err)
		}

		// the plan can not be used: re-planned, and replaces the cached plan
		h.failVerify[prepared] = true
		h.saves = 0
		prep, err := GetAutoPreparePlan(name, autoPrepareText, "default", prepContext)
		ce := prepareds.get(name, false)
		if err != nil || prep == nil || prep == prepared || h.replans != 1 || ce == nil || ce.Prepared != prep {
			t.Errorf("ad hoc %v: got %v, %v, %d re-plans; expected the re-planned plan cached", adHoc, prep, err,
				h.replans)
		}
		expectedSaves := map[bool]int{false: 0, true: 1}[adHoc]
		if h.saves != expectedSaves {
			t.Errorf("ad hoc %v: %d saves, expected %d", adHoc, h.saves, expectedSaves)
		}
	}
}

// MB-75514: a plan built in a transaction is not cached (nor saved); transaction statements are cached, not saved
func TestAutoPrepareTransaction(t *testing.T) {
	h := setAutoPrepareHooks(t, settings.PS_MODE_AD_HOC)
	PreparedsInit(16)

	stmt, err := n1ql.ParseStatement2(autoPrepareText, "default", "")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	prepared := autoPreparePlan("tx1", "", true)
	added, err := AddAutoPreparePlan(stmt, prepared, settings.PS_MODE_AD_HOC, map[string]bool{})
	if added || err != nil || prepareds.get("tx1", false) != nil || h.saves != 0 {
		t.Errorf("in a transaction: added %v, %v, %d saves; expected not cached, not saved", added, err, h.saves)
	}

	for _, text := range []string{"START TRANSACTION", "COMMIT TRANSACTION", "ROLLBACK TRANSACTION"} {
		stmt, err := n1ql.ParseStatement2(text, "default", "")
		if err != nil {
			t.Fatalf("%s: parse error: %v", text, err)
		}
		prepared := autoPreparePlan(text, "", true)
		prepared.SetText(text)
		added, err := AddAutoPreparePlan(stmt, prepared, settings.PS_MODE_AD_HOC, nil)
		if !added || err != nil || prepareds.get(text, false) == nil || h.saves != 0 || prepared.AdHoc() {
			t.Errorf("%s: added %v, %v, %d saves, ad hoc %v; expected cached, not saved, not ad hoc", text,
				added, err, h.saves, prepared.AdHoc())
		}
	}
}
