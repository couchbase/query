//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/couchbase/cbauth/metakv"
)

// the default plan stability setting is written to metakv only over the settings this node last saw
func TestPersistDefaultPlanStabilitySetting(t *testing.T) {
	defer func(g func() ([]byte, any, error), s func([]byte, any) error, gs *querySettings, rev any) {
		getStoredSettings = g
		setStoredSettings = s
		globalSettings = gs
		setSettingsRev(rev)
	}(getStoredSettings, setStoredSettings, globalSettings, getSettingsRev())

	// this node's setting has been reset
	globalSettings = &querySettings{settings: map[string]any{PLAN_STABILITY: defaultPlanStabilitySettings()}}

	seenRev := []byte("5")
	for _, tc := range []struct {
		name     string
		rev      any    // revision this node last saw
		stored   string // settings in metakv (read only if rev is nil)
		getErr   error
		setErr   error
		writes   int
		writeRev any
		err      bool
	}{
		{"written over the settings last seen", seenRev, "", nil, nil, 1, seenRev, false},
		{"changed since last seen (another node, or turned on again)", seenRev, "", nil, metakv.ErrRevMismatch,
			1, seenRev, false},
		{"write fails", seenRev, "", nil, fmt.Errorf("metakv not available"), 1, seenRev, true},
		{"empty revision: not seen", []byte{}, "", nil, nil, 1, []byte("7"), false},
		{"not seen, default stored", nil, `{"plan_stability":{"mode":0,"error_policy":1},"node":"n1"}`, nil, nil,
			0, nil, false},
		{"not seen, not default stored", nil, `{"plan_stability":{"mode":2,"error_policy":1}}`, nil, nil,
			1, []byte("7"), false},
		{"not seen, error policy not default", nil, `{"plan_stability":{"mode":0,"error_policy":2}}`, nil, nil,
			1, []byte("7"), false},
		{"not seen, nothing stored", nil, ``, nil, nil, 1, []byte("7"), false},
		{"not seen, not JSON", nil, `{`, nil, nil, 1, []byte("7"), false},
		{"not seen, metakv get fails", nil, ``, fmt.Errorf("metakv not available"), nil, 0, nil, true},
	} {
		setSettingsRev(tc.rev)
		stored, getErr, setErr := tc.stored, tc.getErr, tc.setErr
		getStoredSettings = func() ([]byte, any, error) {
			return []byte(stored), []byte("7"), getErr
		}
		writes := 0
		var writeRev any
		var written []byte
		setStoredSettings = func(val []byte, rev any) error {
			writes++
			writeRev = rev
			written = val
			return setErr
		}

		err := PersistDefaultPlanStabilitySetting()
		if (err != nil) != tc.err {
			t.Errorf("%s: got error %v, expected error %v", tc.name, err, tc.err)
		}
		if writes != tc.writes {
			t.Errorf("%s: %d writes, expected %d", tc.name, writes, tc.writes)
			continue
		}
		if writes == 0 {
			continue
		}
		wb, _ := writeRev.([]byte)
		eb, _ := tc.writeRev.([]byte)
		if (writeRev == nil) != (tc.writeRev == nil) || !bytes.Equal(wb, eb) {
			t.Errorf("%s: written with revision %v, expected %v", tc.name, writeRev, tc.writeRev)
		}
		var vmap map[string]any
		if json.Unmarshal(written, &vmap) != nil || !isDefaultPlanStability(vmap[PLAN_STABILITY]) {
			t.Errorf("%s: written settings %s do not have the default plan stability setting", tc.name, written)
		}
	}
}
