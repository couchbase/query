//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package settings

import (
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

func PlanStabilityAvailable() bool {
	return true
}

func updatePlanStabilitySetting(requestId string, val interface{}) errors.Error {
	psMap, ok := val.(map[string]interface{})
	if !ok {
		return errors.NewSettingsInvalidValue(PLAN_STABILITY, "map[string]interface{}", val)
	}
	// GetPlanStabilitySetting() returns a copy of the settings
	planStability, err := GetPlanStabilitySetting()
	if err != nil {
		return err
	}
	var oldMode PlanStabilityMode
	if oldModeVal, ok := planStability["mode"]; ok {
		oldMode = PlanStabilityMode(getIntValue(oldModeVal, int(PS_MODE_OFF)))
	}

	for kk, vv := range psMap {
		if actual, ok := vv.(value.Value); ok {
			vv = actual.Actual()
		}

		// When JSON is unmarshalled into an interface, numbers are unmarshalled into float.
		if f, ok := vv.(float64); ok && value.IsInt(f) {
			vv = int64(f)
		}

		switch kk {
		case "mode":
			var newMode PlanStabilityMode
			switch vv := vv.(type) {
			case string:
				// when user sets the setting
				if mode, ok := _PS_MODE_MAP[strings.ToLower(vv)]; ok {
					planStability[kk] = mode
					newMode = mode
				} else {
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "'off'/'prepared_only'/'ad_hoc'/'ad_hoc_read_only'", vv)
				}
			case int64:
				// when setting comes from metakv
				mode := PlanStabilityMode(vv)
				if mode >= PS_MODE_OFF && mode <= PS_MODE_AD_HOC_READ_ONLY {
					planStability[kk] = mode
					newMode = mode
				} else {
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			default:
				return errors.NewSettingsInvalidType(PLAN_STABILITY+".mode", "string", vv)
			}
			err := planCache.UpdatePlanStabilityMode(oldMode, newMode, requestId)
			if err != nil {
				return err
			}
		case "error_policy":
			switch vv := vv.(type) {
			case string:
				// when user sets the setting
				if error_policy, ok := _PS_ERROR_POLICY_MAP[strings.ToLower(vv)]; ok {
					planStability[kk] = error_policy
				} else {
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "'strict'/'moderate'/'flexible'", vv)
				}
			case int64:
				// when setting comes from metakv
				error_policy := PlanStabilityErrorPolicy(vv)
				if error_policy >= PS_ERROR_FLEXIBLE && error_policy <= PS_ERROR_STRICT {
					planStability[kk] = error_policy
				} else {
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "", vv)
				}
			default:
				return errors.NewSettingsInvalidType(PLAN_STABILITY+".error_policy", "string", vv)
			}
		default:
			return errors.NewSettingsInvalidValue(PLAN_STABILITY+"."+kk, "", nil)
		}
	}
	// update settings once all processed for plan stability
	globalSettings.setSetting(PLAN_STABILITY, planStability)

	return nil
}
