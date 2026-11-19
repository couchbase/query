//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

// an encoded plan that can not be decoded (unknown operator)
func undecodablePlan(name string) string {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte(`{"name":"` + name + `","text":"SELECT 1","operator":{"#operator":"NoSuchOperator"}}`))
	w.Close()
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

// a plan from another node (distributed, remote prime, or supplied by the client) under plan stability that
// is not put in the cache is marked unloaded, so that a cache miss loads the saved plan and applies the
// error policy, instead of returning 4040
func TestDecodePolicy(t *testing.T) {
	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)

	for _, tc := range []struct {
		name          string
		queryContext  string
		planStability bool
		unloaded      string // name expected to be marked unloaded ("": none)
	}{
		{"dp1", "", true, "dp1"},
		{"dp2", "default:b.s", true, "dp2(default:b.s)"},
		{"dp3", "", false, ""},
	} {
		clearUnloaded(tc.name)
		clearUnloaded(tc.unloaded)

		// STRICT: the plan can not be decoded, and is not re-planned (reprep only with plan stability)
		_, err, _ := decodePrepared(tc.name, tc.queryContext, undecodablePlan(tc.name), false, nil,
			tc.planStability, false, tc.planStability, false, false, settings.PS_MODE_PREPARED_ONLY,
			settings.PS_ERROR_STRICT, logging.NULL_LOG)
		if err == nil {
			t.Errorf("%s: expected an error", tc.name)
		} else if tc.planStability && err.Code() != errors.E_REPREPARE {
			t.Errorf("%s: expected the STRICT reprepare error, got %v", tc.name, err)
		}
		if tc.unloaded != "" && !isUnloaded(tc.unloaded) {
			t.Errorf("%s: %s not marked unloaded", tc.name, tc.unloaded)
		}
		if tc.unloaded == "" && isUnloaded(tc.name) {
			t.Errorf("%s: marked unloaded without plan stability", tc.name)
		}

		// the unloaded names are package-global: leave none for other tests
		clearUnloaded(tc.name)
		clearUnloaded(tc.unloaded)
	}
}

// splits "[node]name" keys, as the cluster implementation does (the default stub never does)
type splitKeyRemote struct {
	distributed.SystemRemoteAccess
}

func (this splitKeyRemote) SplitKey(key string) (string, string) {
	if strings.HasPrefix(key, "[") {
		if i := strings.Index(key, "]"); i > 0 {
			return key[1:i], key[i+1:]
		}
	}
	return "", key
}

// a plan that decodes, but fails verification on this node (e.g. this node does not see its index yet)
func TestDecodeVerifyPolicy(t *testing.T) {
	defer func(v func(*plan.Prepared) errors.Error, r func(*plan.Prepared, map[string]bool, *time.Duration, bool,
		settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy, logging.Log,
		datastore.ScanConsistency) (*plan.Prepared, errors.Error), ra distributed.SystemRemoteAccess) {
		verifyPreparedFn = v
		reprepareFn = r
		distributed.SetRemoteAccess(ra)
	}(verifyPreparedFn, reprepareFn, distributed.RemoteAccess())

	distributed.SetRemoteAccess(splitKeyRemote{distributed.RemoteAccess()})
	verifyPreparedFn = func(*plan.Prepared) errors.Error {
		return errors.NewPlanVerificationError("index not found", nil)
	}
	var replanned *plan.Prepared
	reprepareFn = func(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
		planStability bool, planStabilityMode settings.PlanStabilityMode,
		planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log,
		scanConsistency datastore.ScanConsistency) (*plan.Prepared, errors.Error) {
		replanned = plan.NewPrepared(nil, nil, nil, nil, nil)
		replanned.SetName(prepared.Name())
		replanned.SetText(prepared.Text())
		return replanned, nil
	}

	for _, tc := range []struct {
		name          string // as received, e.g. with a [node] prefix
		cacheName     string
		planStability bool
		policy        settings.PlanStabilityErrorPolicy
		err           bool
		cached        bool
		unloaded      bool
	}{
		{"vp1", "vp1", true, settings.PS_ERROR_STRICT, true, false, true},
		{"vp2", "vp2", true, settings.PS_ERROR_MODERATE, false, false, true},
		{"vp3", "vp3", true, settings.PS_ERROR_FLEXIBLE, false, true, false},
		{"vp4", "vp4", false, settings.PS_ERROR_STRICT, false, true, false}, // plan stability off
		{"[node1]vp5", "vp5", true, settings.PS_ERROR_MODERATE, false, false, true},
		{"[node1]vp6", "vp6", true, settings.PS_ERROR_STRICT, true, false, true},
	} {
		// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
		PreparedsInit(16)
		clearUnloaded(tc.cacheName)
		replanned = nil

		// a plan that decodes without a datastore
		prepared := plan.NewPrepared(plan.NewSequence(plan.NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
		prepared.SetName(tc.cacheName)
		prepared.SetText("SELECT 1")
		encoded, er := prepared.BuildEncodedPlan()
		if er != nil {
			t.Fatalf("%s: encoding: %v", tc.name, er)
		}
		mode := settings.PS_MODE_OFF
		if tc.planStability {
			mode = settings.PS_MODE_PREPARED_ONLY
		}

		decoded, err, _ := decodePrepared(tc.name, "", encoded, false, nil, true, false, tc.planStability, false,
			false, mode, tc.policy, logging.NULL_LOG)
		ce := prepareds.get(tc.cacheName, false)
		if (err != nil) != tc.err || (ce != nil) != tc.cached || isUnloaded(tc.cacheName) != tc.unloaded {
			t.Errorf("%s (%v): error %v, cached %v, unloaded %v; expected error %v, cached %v, unloaded %v",
				tc.name, tc.policy, err, ce != nil, isUnloaded(tc.cacheName), tc.err, tc.cached, tc.unloaded)
		}
		if !tc.err && (decoded == nil || decoded != replanned) {
			t.Errorf("%s (%v): expected the re-planned plan, got %v", tc.name, tc.policy, decoded)
		}
		if tc.cached && ce.Prepared != replanned {
			t.Errorf("%s (%v): the re-planned plan is not the cached plan", tc.name, tc.policy)
		}

		// the unloaded names are package-global: leave none for other tests
		clearUnloaded(tc.cacheName)
	}
}

// MODERATE, a plan that can not be decoded: re-planned for this execution only, not cached, marked unloaded
func TestDecodeModeratePolicy(t *testing.T) {
	defer func(r func(*plan.Prepared, map[string]bool, *time.Duration, bool, settings.PlanStabilityMode,
		settings.PlanStabilityErrorPolicy, logging.Log, datastore.ScanConsistency) (*plan.Prepared, errors.Error)) {
		reprepareFn = r
	}(reprepareFn)

	var replanned *plan.Prepared
	reprepareFn = func(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
		planStability bool, planStabilityMode settings.PlanStabilityMode,
		planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log,
		scanConsistency datastore.ScanConsistency) (*plan.Prepared, errors.Error) {
		replanned = plan.NewPrepared(nil, nil, nil, nil, nil)
		replanned.SetName(prepared.Name())
		replanned.SetText(prepared.Text())
		return replanned, nil
	}

	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)
	clearUnloaded("mp1")
	decoded, err, _ := decodePrepared("mp1", "", undecodablePlan("mp1"), false, nil, true, false, true, false, false,
		settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE, logging.NULL_LOG)
	if err != nil || decoded == nil || decoded != replanned || prepareds.get("mp1", false) != nil || !isUnloaded("mp1") {
		t.Errorf("MODERATE: error %v, re-planned plan returned %v, cached %v, unloaded %v", err,
			decoded != nil && decoded == replanned, prepareds.get("mp1", false) != nil, isUnloaded("mp1"))
	}
	clearUnloaded("mp1")
}
