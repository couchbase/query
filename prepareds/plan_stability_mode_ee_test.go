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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

// MB-75471: a plan stability mode change made on another node removes the entries the new mode does not keep
// from this node's prepared cache
func TestApplyPlanStabilityMode(t *testing.T) {
	type entry struct {
		name                   string
		adHoc, inlineUdf, save bool
	}
	entries := []entry{
		{"adhoc1", true, false, false},
		{GetUdfPreparedName("default:b.s.f1"), false, true, false},
		{"p1", false, false, false},
		{"saved1", false, false, true},
	}

	for _, tc := range []struct {
		oldMode, newMode settings.PlanStabilityMode
		removed          bool // ad hoc and inline UDF entries removed
	}{
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_PREPARED_ONLY, true},
		{settings.PS_MODE_AD_HOC_READ_ONLY, settings.PS_MODE_PREPARED_ONLY, true},
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_OFF, true},
		{settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_OFF, true},
		{settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_AD_HOC, false},
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_AD_HOC_READ_ONLY, false},
		{settings.PS_MODE_OFF, settings.PS_MODE_AD_HOC, false},
	} {
		// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
		PreparedsInit(16)
		for _, e := range entries {
			prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
			prepared.SetName(e.name)
			prepared.SetText("SELECT 1")
			prepared.SetAdHoc(e.adHoc)
			prepared.SetPersist(e.save)
			if e.inlineUdf {
				prepared.SetInlineUdf()
			}
			prepareds.add(prepared, false, false, nil)
		}

		prepareds.ApplyPlanStabilityMode(tc.oldMode, tc.newMode)

		for _, e := range entries {
			cached := prepareds.get(e.name, false) != nil
			if expected := !(tc.removed && (e.adHoc || e.inlineUdf)); cached != expected {
				t.Errorf("%v to %v: %s cached %v, expected %v", tc.oldMode, tc.newMode, e.name, cached, expected)
			}
		}
	}
}
