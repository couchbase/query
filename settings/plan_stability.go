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

	"github.com/couchbase/query/errors"
)

/*
 * Plan Stability mode:
 *
 * By default plan stability is OFF. When it is turned on, one of two modes can be used:
 *   - prepared_only: query plan will be saved for prepared statements only
 *   - ad_hoc: query plan will be saved for all statements
 *   - ad_hoc_read_only: no new query plan saved, allow usage of previously saved query plans
 *                       for ad hoc statements
 *
 * The mode can be set/changed via:
 *
 *   UPDATE system:settings SET plan_stability.mode = "off"/"prepared_only"/"ad_hoc"/"ad_hoc_read_only"
 *
 * synonyms accepted:
 *   "prepared-only", "prepared only" in addition to "prepared_only"
 *   "ad-hoc", "ad hoc" in addition to "ad_hoc"
 *   "ad-hoc-read-only", "ad hoc read only" in addition to "ad_hoc_read_only"
 */
type PlanStabilityMode int

const (
	PS_MODE_OFF = PlanStabilityMode(iota)
	PS_MODE_PREPARED_ONLY
	PS_MODE_AD_HOC
	PS_MODE_AD_HOC_READ_ONLY
)

var _PS_MODE_MAP map[string]PlanStabilityMode = map[string]PlanStabilityMode{
	"off":              PS_MODE_OFF,
	"prepared_only":    PS_MODE_PREPARED_ONLY,
	"prepared-only":    PS_MODE_PREPARED_ONLY,
	"prepared only":    PS_MODE_PREPARED_ONLY,
	"ad_hoc":           PS_MODE_AD_HOC,
	"ad-hoc":           PS_MODE_AD_HOC,
	"ad hoc":           PS_MODE_AD_HOC,
	"ad_hoc_read_only": PS_MODE_AD_HOC_READ_ONLY,
	"ad-hoc-read-only": PS_MODE_AD_HOC_READ_ONLY,
	"ad hoc read only": PS_MODE_AD_HOC_READ_ONLY,
}

func (this PlanStabilityMode) String() string {
	switch this {
	case PS_MODE_OFF:
		return "off"
	case PS_MODE_PREPARED_ONLY:
		return "prepared_only"
	case PS_MODE_AD_HOC:
		return "ad_hoc"
	case PS_MODE_AD_HOC_READ_ONLY:
		return "ad_hoc_read_only"
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

func GetPlanStabilitySetting() (map[string]interface{}, errors.Error) {
	// getSettings() returns a copy of the settings
	psSetting := globalSettings.getSetting(PLAN_STABILITY)
	planStability, ok := psSetting.(map[string]interface{})
	if !ok {
		return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY, "map[string]interface{}", psSetting)
	}
	return planStability, nil
}

func SetPlanStabilitySetting(psSetting map[string]interface{}) {
	globalSettings.setSetting(PLAN_STABILITY, psSetting)
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
	return mode == PS_MODE_PREPARED_ONLY || mode == PS_MODE_AD_HOC || mode == PS_MODE_AD_HOC_READ_ONLY
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

func IsPlanStabilityAdHocReadOnly() bool {
	mode := GetPlanStabilityMode()
	return mode == PS_MODE_AD_HOC_READ_ONLY
}

func GetPlanStabilityErrorPolicy() PlanStabilityErrorPolicy {
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
	error_policy := GetPlanStabilityErrorPolicy()
	return error_policy == PS_ERROR_FLEXIBLE
}

func IsPlanStabilityErrorModerate() bool {
	error_policy := GetPlanStabilityErrorPolicy()
	return error_policy == PS_ERROR_MODERATE
}

func IsPlanStabilityErrorStrict() bool {
	error_policy := GetPlanStabilityErrorPolicy()
	return error_policy == PS_ERROR_STRICT
}

// represents prepareds

type PlanCache interface {
	UpdatePlanStabilityMode(oldMode, newMode PlanStabilityMode, requestId string) errors.Error
}

var planCache PlanCache

func SetPlanCache(pc PlanCache) {
	planCache = pc
}
