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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/settings"
)

func TestReprepareAllowed(t *testing.T) {
	policies := []settings.PlanStabilityErrorPolicy{
		settings.PS_ERROR_FLEXIBLE,
		settings.PS_ERROR_MODERATE,
		settings.PS_ERROR_STRICT,
	}

	for _, policy := range policies {
		// plan stability off: always re-prepare and replace
		replace, err := reprepareAllowed(false, policy)
		if err != nil || !replace {
			t.Errorf("plan stability off, policy %v: expected replace, got replace %v err %v", policy, replace, err)
		}

		// plan stability on: the error policy applies
		replace, err = reprepareAllowed(true, policy)
		switch policy {
		case settings.PS_ERROR_FLEXIBLE:
			if err != nil || !replace {
				t.Errorf("FLEXIBLE: expected replace, got replace %v err %v", replace, err)
			}
		case settings.PS_ERROR_MODERATE:
			if err != nil || replace {
				t.Errorf("MODERATE: expected no replace, got replace %v err %v", replace, err)
			}
		case settings.PS_ERROR_STRICT:
			if err == nil || err.Code() != errors.E_REPREPARE {
				t.Errorf("STRICT: expected reprepare error, got replace %v err %v", replace, err)
			}
		}
	}
}
