//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package settings

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

/*
 * Plan Stability mode:
 *
 * By default plan stability is OFF. When it is turned on, one of two modes can be used:
 *   - prepared_only: query plan will be saved for prepared statements only
 *   - ad_hoc: query plan will be saved for all statements
 *
 * The mode can be set/changed via:
 *
 *   UPDATE system:settings SET plan_stability.mode = "off"/"prepared_only"/"ad_hoc"
 */
type PlanStabilityMode int

const (
	PS_MODE_OFF = PlanStabilityMode(iota)
	PS_MODE_PREPARED_ONLY
	PS_MODE_AD_HOC
)

var _PS_MODE_MAP map[string]PlanStabilityMode = map[string]PlanStabilityMode{
	"off":           PS_MODE_OFF,
	"prepared_only": PS_MODE_PREPARED_ONLY,
	"ad_hoc":        PS_MODE_AD_HOC,
}

func (this PlanStabilityMode) String() string {
	switch this {
	case PS_MODE_OFF:
		return "off"
	case PS_MODE_PREPARED_ONLY:
		return "prepared_only"
	case PS_MODE_AD_HOC:
		return "ad_hoc"
	}
	return fmt.Sprintf("invalid plan stability mode (%d)", this)
}

/*
 * Plan Stability error policy:
 *
 * When plan verification fails for a saved prepared plan (for whatever reason), the error policy
 * determines the actions that will follow:
 *   - strict: an error will be returned to the user, the saved plan remains unchanged
 *   - moderate: the query will be reprepared and executed, but the reprepared plan is only used
 *               for the current execution, the saved plan remains unchanged
 *   - flexible: the query will be reprepared and executed, and the reprepared plan will be saved
 *               (replacing the currently saved plan)
 *
 * The error policy can be set/changed via:
 *
 *   UPDATE system:settings SET plan_stability.error_policy = "strict"/"moderate"/"flexible"
 */
type PlanStabilityErrorPolicy int

const (
	PS_ERROR_FLEXIBLE = PlanStabilityErrorPolicy(iota)
	PS_ERROR_MODERATE
	PS_ERROR_STRICT
)

var _PS_ERROR_POLICY_MAP map[string]PlanStabilityErrorPolicy = map[string]PlanStabilityErrorPolicy{
	"flexible": PS_ERROR_FLEXIBLE,
	"moderate": PS_ERROR_MODERATE,
	"strict":   PS_ERROR_STRICT,
}

func (this PlanStabilityErrorPolicy) String() string {
	switch this {
	case PS_ERROR_FLEXIBLE:
		return "flexible"
	case PS_ERROR_MODERATE:
		return "moderate"
	case PS_ERROR_STRICT:
		return "strict"
	}
	return fmt.Sprintf("invalid plan stability error policy (%d)", this)
}

func defaultPlanStabilitySettings() map[string]interface{} {
	return map[string]interface{}{
		"mode":         PS_MODE_OFF,
		"error_policy": PS_ERROR_MODERATE,
	}
}

func updatePlanStabilitySetting(enterprise bool, val interface{}) errors.Error {
	if !enterprise {
		return errors.NewSettingsEnterpriseOnly("Plan Stability")
	}
	psMap, ok := val.(map[string]interface{})
	if !ok {
		return errors.NewSettingsInvalidValue(PLAN_STABILITY, "map[string]interface{}", val)
	}
	// getSettings() returns a copy of the settings
	psSetting := globalSettings.getSetting(PLAN_STABILITY)
	planStability, ok := psSetting.(map[string]interface{})
	if !ok {
		return errors.NewSettingsInvalidValue(PLAN_STABILITY, "map[string]interface{}", psSetting)
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
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "'off'/'prepared_only'/'ad_hoc'", vv)
				}
			case int64:
				// when setting comes from metakv
				mode := PlanStabilityMode(vv)
				if mode >= PS_MODE_OFF && mode <= PS_MODE_AD_HOC {
					planStability[kk] = mode
					newMode = mode
				} else {
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			default:
				return errors.NewSettingsInvalidType(PLAN_STABILITY+".mode", "string", vv)
			}
			err := planCache.UpdatePlanStabilityMode(oldMode, newMode)
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

func GetPlanStabilityMode() PlanStabilityMode {
	mode := PS_MODE_OFF
	globalSettings.RLock()
	setting_val := globalSettings.settings[PLAN_STABILITY]
	if ps_setting, ok := setting_val.(map[string]interface{}); ok {
		if ps_mode_val, ok := ps_setting["mode"]; ok {
			mode = PlanStabilityMode(getIntValue(ps_mode_val, int(mode)))
		}
	}
	globalSettings.RUnlock()
	return mode
}

func IsPlanStabilityEnabled() bool {
	mode := GetPlanStabilityMode()
	return mode == PS_MODE_PREPARED_ONLY || mode == PS_MODE_AD_HOC
}

func IsPlanStabilityDisabled() bool {
	mode := GetPlanStabilityMode()
	return mode == PS_MODE_OFF
}

func IsPlanStabilityPreparedOnly() bool {
	mode := GetPlanStabilityMode()
	return mode == PS_MODE_PREPARED_ONLY
}

func IsPlanStabilityAdHoc() bool {
	mode := GetPlanStabilityMode()
	return mode == PS_MODE_AD_HOC
}

func getPlanStabilityErrorPolicy() PlanStabilityErrorPolicy {
	error_policy := PS_ERROR_MODERATE
	globalSettings.RLock()
	setting_val := globalSettings.settings[PLAN_STABILITY]
	if ps_setting, ok := setting_val.(map[string]interface{}); ok {
		if ps_error_policy_val, ok := ps_setting["error_policy"]; ok {
			error_policy = PlanStabilityErrorPolicy(getIntValue(ps_error_policy_val, int(error_policy)))
		}
	}
	globalSettings.RUnlock()
	return error_policy
}

func IsPlanStabilityErrorFlexible() bool {
	error_policy := getPlanStabilityErrorPolicy()
	return error_policy == PS_ERROR_FLEXIBLE
}

func IsPlanStabilityErrorModerate() bool {
	error_policy := getPlanStabilityErrorPolicy()
	return error_policy == PS_ERROR_MODERATE
}

func IsPlanStabilityErrorStrict() bool {
	error_policy := getPlanStabilityErrorPolicy()
	return error_policy == PS_ERROR_STRICT
}

// represents prepareds

type PlanCache interface {
	UpdatePlanStabilityMode(oldMode, newMode PlanStabilityMode) errors.Error
}

var planCache PlanCache

func SetPlanCache(pc PlanCache) {
	planCache = pc
}
