//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package prepareds

import (
	"testing"

	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
	"github.com/couchbase/query/util"
)

// MB-75471: a plan stability mode change made on another node removes the entries the new mode does not keep
// from this node's prepared cache
func TestApplyPlanStabilityMode(t *testing.T) {
	type entry struct {
		name                                 string
		adHoc, inlineUdf, save, autoPrepared bool
	}
	entries := []entry{
		{"adhoc1", true, false, false, true}, // saved ad hoc statement (AD_HOC)
		{GetUdfPreparedName("default:b.s.f1"), false, true, false, false},
		{"p1", false, false, false, false},
		{"saved1", false, false, true, false},
		{"auto1", false, false, false, true}, // ad hoc statement cached under another mode than AD_HOC
	}

	for _, tc := range []struct {
		oldMode, newMode settings.PlanStabilityMode
		removed          bool // ad hoc and inline UDF entries removed
		adopted          bool // auto prepared entries not marked as ad hoc removed (MB-75479)
	}{
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_PREPARED_ONLY, true, false},
		{settings.PS_MODE_AD_HOC_READ_ONLY, settings.PS_MODE_PREPARED_ONLY, true, false},
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_OFF, true, false},
		{settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_OFF, true, false},
		{settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_AD_HOC, false, true},
		{settings.PS_MODE_AD_HOC_READ_ONLY, settings.PS_MODE_AD_HOC, false, true},
		{settings.PS_MODE_OFF, settings.PS_MODE_AD_HOC, false, true},
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_AD_HOC_READ_ONLY, false, false},
		{settings.PS_MODE_OFF, settings.PS_MODE_PREPARED_ONLY, false, false},
	} {
		// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
		PreparedsInit(16)
		for _, e := range entries {
			prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
			prepared.SetName(e.name)
			prepared.SetText("SELECT 1")
			prepared.SetAdHoc(e.adHoc)
			prepared.SetPersist(e.save)
			prepared.SetAutoPrepared(e.autoPrepared)
			if e.inlineUdf {
				prepared.SetInlineUdf()
			}
			prepareds.add(prepared, false, false, nil)
		}

		prepareds.ApplyPlanStabilityMode(tc.oldMode, tc.newMode)

		for _, e := range entries {
			cached := prepareds.get(e.name, false) != nil
			expected := !(tc.removed && (e.adHoc || e.inlineUdf)) &&
				!(tc.adopted && e.autoPrepared && !e.adHoc && !e.save)
			if cached != expected {
				t.Errorf("%v to %v: %s cached %v, expected %v", tc.oldMode, tc.newMode, e.name, cached, expected)
			}
		}
	}
}

// MB-75479: the cached plans saved when plan stability is turned on
func TestPersistOnTurnOn(t *testing.T) {
	newPrepared := func(autoPrepared, adHoc, save, inlineUdf bool) *plan.Prepared {
		prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
		prepared.SetAutoPrepared(autoPrepared)
		prepared.SetAdHoc(adHoc)
		prepared.SetPersist(save)
		if inlineUdf {
			prepared.SetInlineUdf()
		}
		return prepared
	}
	for _, tc := range []struct {
		name     string
		prepared *plan.Prepared
		mode     settings.PlanStabilityMode
		persist  bool
	}{
		{"prepared statement, PREPARED_ONLY", newPrepared(false, false, false, false), settings.PS_MODE_PREPARED_ONLY, true},
		{"prepared statement, AD_HOC", newPrepared(false, false, false, false), settings.PS_MODE_AD_HOC, true},
		{"prepared statement, AD_HOC_READ_ONLY", newPrepared(false, false, false, false), settings.PS_MODE_AD_HOC_READ_ONLY, true},
		{"ad hoc statement, PREPARED_ONLY", newPrepared(true, false, false, false), settings.PS_MODE_PREPARED_ONLY, false},
		{"ad hoc statement, AD_HOC", newPrepared(true, false, false, false), settings.PS_MODE_AD_HOC, false},
		{"ad hoc statement, AD_HOC_READ_ONLY", newPrepared(true, false, false, false), settings.PS_MODE_AD_HOC_READ_ONLY, false},
		{"saved with SAVE", newPrepared(false, false, true, false), settings.PS_MODE_PREPARED_ONLY, false},
		{"inline UDF", newPrepared(false, false, false, true), settings.PS_MODE_PREPARED_ONLY, false},
	} {
		if persist := persistOnTurnOn(tc.prepared, tc.mode); persist != tc.persist {
			t.Errorf("%s: persist %v, expected %v", tc.name, persist, tc.persist)
		}
	}

	// a prepared statement with a sequential scan: not saved when it is kept local (N1QL_IGNORE_IDXR_META bit set)
	defer func(h func(*plan.Prepared) bool, control uint64) {
		hasSeqScanFn = h
		util.SetN1qlFeatureControl(control)
	}(hasSeqScanFn, util.GetN1qlFeatureControl())
	hasSeqScanFn = func(*plan.Prepared) bool { return true }
	for _, keptLocal := range []bool{true, false} {
		control := util.GetN1qlFeatureControl() &^ util.N1QL_IGNORE_IDXR_META
		if keptLocal {
			control |= util.N1QL_IGNORE_IDXR_META
		}
		util.SetN1qlFeatureControl(control)
		persist := persistOnTurnOn(newPrepared(false, false, false, false), settings.PS_MODE_PREPARED_ONLY)
		if persist == keptLocal {
			t.Errorf("sequential scan, kept local %v: persist %v", keptLocal, persist)
		}
	}
}

// MB-75479: a request planned under another mode than AD_HOC does not cache its ad hoc statement once the mode
// is AD_HOC (the cached ad hoc statements have been removed, the next execution plans and saves it)
func TestAutoPrepareModeChange(t *testing.T) {
	defer func(m func() settings.PlanStabilityMode) {
		planStabilityModeFn = m
	}(planStabilityModeFn)

	const text = "SELECT 1 AS one FROM b USE KEYS \"x\""
	stmt, err := n1ql.ParseStatement2(text, "default", "")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, tc := range []struct {
		name                 string
		requestMode, nowMode settings.PlanStabilityMode
		cached               bool
	}{
		{"OFF, now AD_HOC", settings.PS_MODE_OFF, settings.PS_MODE_AD_HOC, false},
		{"PREPARED_ONLY, now AD_HOC", settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_AD_HOC, false},
		{"AD_HOC_READ_ONLY, now AD_HOC", settings.PS_MODE_AD_HOC_READ_ONLY, settings.PS_MODE_AD_HOC, false},
		{"OFF, still OFF", settings.PS_MODE_OFF, settings.PS_MODE_OFF, true},
		{"OFF, now PREPARED_ONLY", settings.PS_MODE_OFF, settings.PS_MODE_PREPARED_ONLY, true},
	} {
		// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
		PreparedsInit(16)
		nowMode := tc.nowMode
		planStabilityModeFn = func() settings.PlanStabilityMode { return nowMode }

		prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
		prepared.SetName("am1")
		prepared.SetText(text)
		added, er := AddAutoPreparePlan(stmt, prepared, tc.requestMode)
		cached := prepareds.get("am1", false) != nil
		if er != nil || added != tc.cached || cached != tc.cached {
			t.Errorf("%s: added %v, cached %v, error %v; expected cached %v", tc.name, added, cached, er, tc.cached)
		}
	}
}
