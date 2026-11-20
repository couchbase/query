//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package settings

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
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

const PS_MODE_DEFAULT = PS_MODE_OFF

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
	"default":          PS_MODE_DEFAULT,
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

const PS_ERROR_DEFAULT = PS_ERROR_MODERATE

var _PS_ERROR_POLICY_MAP map[string]PlanStabilityErrorPolicy = map[string]PlanStabilityErrorPolicy{
	"flexible": PS_ERROR_FLEXIBLE,
	"moderate": PS_ERROR_MODERATE,
	"strict":   PS_ERROR_STRICT,
	"default":  PS_ERROR_DEFAULT,
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
		"mode":         PS_MODE_DEFAULT,
		"error_policy": PS_ERROR_DEFAULT,
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

// MB-75508: reset this node's plan stability setting to the default setting (e.g. QUERY_METADATA is missing), under
// settingsUpdateLock, as other changes to the settings: an update in progress (with its write to metakv, and
// setting the old settings back if the write fails) completes first, and does not undo the reset. Returns whether
// the setting was the default setting already.
func ResetPlanStabilitySetting() bool {
	settingsUpdateLock.Lock()
	defer settingsUpdateLock.Unlock()
	wasDefault := IsDefaultPlanStabilitySetting()
	globalSettings.setSetting(PLAN_STABILITY, defaultPlanStabilitySettings())
	return wasDefault
}

// whether this node's plan stability setting is the default setting
func IsDefaultPlanStabilitySetting() bool {
	return GetPlanStabilityMode() == PS_MODE_DEFAULT && GetPlanStabilityErrorPolicy() == PS_ERROR_DEFAULT
}

func isDefaultPlanStability(val any) bool {
	ps_setting, ok := val.(map[string]any)
	if !ok {
		return false
	}
	mode, ok := ps_setting["mode"]
	if !ok || getPlanStabilityModeValue(mode) != PS_MODE_DEFAULT {
		return false
	}
	error_policy, ok := ps_setting["error_policy"]
	return ok && getPlanStabilityErrorPolicyValue(error_policy) == PS_ERROR_DEFAULT
}

// test hooks
var getStoredSettings = getMetakvSettings
var setStoredSettings = setMetakvSettings

/*
 * Write the settings, after this node's plan stability setting was set to the default setting, to metakv.
 * Every node that finds the QUERY_METADATA bucket missing resets plan stability, and the write only
 * replaces the settings this node last saw (the metakv revision): if the settings in metakv changed since,
 * another node already wrote the default setting, or the setting was changed deliberately (e.g. plan
 * stability turned on again after QUERY_METADATA was created again) and must not be overwritten.
 * Comparing the setting can not tell such a change from a setting not yet reset.
 */
func PersistDefaultPlanStabilitySetting() errors.Error {
	// MB-75508: under settingsUpdateLock, as other changes to the settings, and only if this node's setting is
	// still the default setting: an update made since (e.g. plan stability turned on again) is not overwritten
	settingsUpdateLock.Lock()
	defer settingsUpdateLock.Unlock()
	if !IsDefaultPlanStabilitySetting() {
		logging.Infof("SETTINGS: plan_stability setting changed since it was reset, DEFAULT setting not written")
		return nil
	}

	rev := getSettingsRev()
	if rev == nil {
		// this node has not seen the settings in metakv (e.g. getting them at startup failed)
		val, storedRev, err := getStoredSettings()
		if err != nil {
			return errors.NewSettingsMetaKVError(err, "Error getting global settings from metakv")
		}
		if len(val) > 0 {
			var vmap map[string]any
			if json.Unmarshal(val, &vmap) == nil && isDefaultPlanStability(vmap[PLAN_STABILITY]) {
				return nil
			}
		}
		rev = storedRev
	}

	bytes, er := marshalSettings()
	if er != nil {
		return er
	}
	err := setStoredSettings(bytes, rev)
	if err == metakv.ErrRevMismatch {
		logging.Infof("SETTINGS: global settings in metakv changed since this node last saw them, " +
			"plan_stability DEFAULT setting not written")
		return nil
	} else if err != nil {
		logging.Errorf("SETTINGS: Error updating global settings in metakv: %v", err)
		return errors.NewSettingsMetaKVError(err, "Error updating global settings in metakv")
	}
	return nil
}

func GetPlanStabilityMode() PlanStabilityMode {
	mode := PS_MODE_DEFAULT
	globalSettings.RLock()
	setting_val := globalSettings.settings[PLAN_STABILITY]
	if ps_setting, ok := setting_val.(map[string]interface{}); ok {
		if ps_mode_val, ok := ps_setting["mode"]; ok {
			mode = getPlanStabilityModeValue(ps_mode_val)
		}
	}
	globalSettings.RUnlock()
	return mode
}

func getPlanStabilityModeValue(val interface{}) PlanStabilityMode {
	mode := PS_MODE_DEFAULT
	switch val := val.(type) {
	case float64:
		if value.IsInt(val) {
			mode = PlanStabilityMode(int64(val))
		}
	case int64:
		mode = PlanStabilityMode(val)
	case PlanStabilityMode:
		mode = val
	}
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
	error_policy := PS_ERROR_DEFAULT
	globalSettings.RLock()
	setting_val := globalSettings.settings[PLAN_STABILITY]
	if ps_setting, ok := setting_val.(map[string]interface{}); ok {
		if ps_error_policy_val, ok := ps_setting["error_policy"]; ok {
			error_policy = getPlanStabilityErrorPolicyValue(ps_error_policy_val)
		}
	}
	globalSettings.RUnlock()
	return error_policy
}

func getPlanStabilityErrorPolicyValue(val interface{}) PlanStabilityErrorPolicy {
	error_policy := PS_ERROR_DEFAULT
	switch val := val.(type) {
	case float64:
		if value.IsInt(val) {
			error_policy = PlanStabilityErrorPolicy(int64(val))
		}
	case int64:
		error_policy = PlanStabilityErrorPolicy(val)
	case PlanStabilityErrorPolicy:
		error_policy = val
	}
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
	// update the prepared cache and the saved plans for a mode change made on this node
	UpdatePlanStabilityMode(oldMode, newMode PlanStabilityMode, requestId string) errors.Error
	// update the prepared cache only, for a mode change made on another node
	ApplyPlanStabilityMode(oldMode, newMode PlanStabilityMode)
	// create QUERY_METADATA if it does not exist, and wait for it
	EnsureQueryMetadata(requestId string) errors.Error
	DeleteUdfPrepared(udfName string, planStabilityMode PlanStabilityMode, planStabilityErrorPolicy PlanStabilityErrorPolicy) errors.Error
}

var planCache PlanCache

func SetPlanCache(pc PlanCache) {
	planCache = pc
}
