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

// when settings come from metakv, validate the settings (including proper type conversion)
func validatePlanStabilitySetting(val any) (map[string]any, errors.Error) {
	planStability, ok := val.(map[string]any)
	if !ok {
		return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY, "map[string]interface{}", val)
	}
	for kk, vv := range planStability {
		if actual, ok := vv.(value.Value); ok {
			vv = actual.Actual()
		}

		// When JSON is unmarshalled into an interface, numbers are unmarshalled into float.
		if f, ok := vv.(float64); ok && value.IsInt(f) {
			vv = int64(f)
		}

		switch kk {
		case "mode":
			switch vv := vv.(type) {
			case int64:
				// when setting comes from metakv
				mode := PlanStabilityMode(vv)
				if mode >= PS_MODE_OFF && mode <= PS_MODE_AD_HOC_READ_ONLY {
					planStability[kk] = mode
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			default:
				return nil, errors.NewSettingsInvalidType(PLAN_STABILITY+".mode", "string", vv)
			}
		case "error_policy":
			switch vv := vv.(type) {
			case int64:
				// when setting comes from metakv
				error_policy := PlanStabilityErrorPolicy(vv)
				if error_policy >= PS_ERROR_FLEXIBLE && error_policy <= PS_ERROR_STRICT {
					planStability[kk] = error_policy
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "", vv)
				}
			default:
				return nil, errors.NewSettingsInvalidType(PLAN_STABILITY+".error_policy", "string", vv)
			}
		default:
			return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+"."+kk, "", nil)
		}
	}

	return planStability, nil
}

func UpdatePersistPlanStabilitySetting(requestId string, val any) errors.Error {
	// MB-75471: the update and its write to metakv under the lock, so that updates are written in the
	// order they are made
	settingsUpdateLock.Lock()
	defer settingsUpdateLock.Unlock()

	old := globalSettings.getAllSettings()
	err := updatePlanStabilitySetting(requestId, val)
	if err != nil {
		return err
	}
	err = PersistSettings()
	if err != nil {
		// the other nodes do not get the new setting, keep the old one
		restoreSettings(old)
	}
	return err
}

// the caller holds settingsUpdateLock
func updatePlanStabilitySetting(requestId string, val any) errors.Error {
	// validate the new setting completely before changing anything
	newSetting, err := parsePlanStabilitySetting(val)
	if err != nil {
		return err
	}
	newMode := newSetting["mode"].(PlanStabilityMode)

	// GetPlanStabilitySetting() returns a copy of the settings
	oldSetting, err := GetPlanStabilitySetting()
	if err != nil {
		return err
	}
	var oldMode PlanStabilityMode
	if oldModeVal, ok := oldSetting["mode"]; ok {
		oldMode = getPlanStabilityModeValue(oldModeVal)
	}

	// when turning plan stability on, QUERY_METADATA must exist before the new mode is in effect: a plan
	// saved while it is being created would find it missing, and reset plan stability (see
	// prepareds.HandleMissingQueryMetadata())
	if oldMode == PS_MODE_OFF && newMode != PS_MODE_OFF {
		err = planCache.EnsureQueryMetadata(requestId)
		if err != nil {
			return err
		}
	}

	// the new mode is in effect before the prepared cache and the saved plans are updated: when turning
	// plan stability on, plans prepared in the meantime are saved; when turning it off, they are not
	globalSettings.setSetting(PLAN_STABILITY, newSetting)
	err = planCache.UpdatePlanStabilityMode(oldMode, newMode, requestId)
	if err != nil {
		// the setting is not saved (to metakv), keep the old one
		globalSettings.setSetting(PLAN_STABILITY, oldSetting)
		return err
	}

	return nil
}

// validate a new plan stability setting, set by the user or (UpdatePersistPlanStabilitySetting()) restored,
// and return it in internal form; a field not present (e.g. UNSET used) gets its default value
func parsePlanStabilitySetting(val any) (map[string]any, errors.Error) {
	psMap, ok := val.(map[string]any)
	if !ok {
		return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY, "map[string]interface{}", val)
	}

	planStability := defaultPlanStabilitySettings()
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
			switch vv := vv.(type) {
			case string:
				// when user sets the setting
				if mode, ok := _PS_MODE_MAP[strings.ToLower(vv)]; ok {
					planStability[kk] = mode
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "'off'/'prepared_only'/'ad_hoc'/'ad_hoc_read_only'", vv)
				}
			case int64:
				// when setting comes from metakv
				mode := PlanStabilityMode(vv)
				if mode >= PS_MODE_OFF && mode <= PS_MODE_AD_HOC_READ_ONLY {
					planStability[kk] = mode
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			case PlanStabilityMode:
				mode := vv
				if mode >= PS_MODE_OFF && mode <= PS_MODE_AD_HOC_READ_ONLY {
					planStability[kk] = mode
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			default:
				return nil, errors.NewSettingsInvalidType(PLAN_STABILITY+".mode", "string", vv)
			}
		case "error_policy":
			switch vv := vv.(type) {
			case string:
				// when user sets the setting
				if error_policy, ok := _PS_ERROR_POLICY_MAP[strings.ToLower(vv)]; ok {
					planStability[kk] = error_policy
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "'strict'/'moderate'/'flexible'", vv)
				}
			case int64:
				// when setting comes from metakv
				error_policy := PlanStabilityErrorPolicy(vv)
				if error_policy >= PS_ERROR_FLEXIBLE && error_policy <= PS_ERROR_STRICT {
					planStability[kk] = error_policy
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "", vv)
				}
			case PlanStabilityErrorPolicy:
				error_policy := vv
				if error_policy >= PS_ERROR_FLEXIBLE && error_policy <= PS_ERROR_STRICT {
					planStability[kk] = error_policy
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "", vv)
				}
			default:
				return nil, errors.NewSettingsInvalidType(PLAN_STABILITY+".error_policy", "string", vv)
			}
		default:
			return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+"."+kk, "", nil)
		}
	}
	return planStability, nil
}

// Given plan stability settings in integer form, remap to string form for easier understanding
// (this function changes the input map and thus caller should use a copy when necessary)
func remapPlanStabilitySetting(psSetting any) (map[string]any, errors.Error) {
	planStability, ok := psSetting.(map[string]any)
	if !ok {
		return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY, "map[string]interface{}", psSetting)
	}
	for kk, vv := range planStability {
		if actual, ok := vv.(value.Value); ok {
			vv = actual.Actual()
		}

		// When JSON is unmarshalled into an interface, numbers are unmarshalled into float.
		if f, ok := vv.(float64); ok && value.IsInt(f) {
			vv = int64(f)
		}

		switch kk {
		case "mode":
			switch vv := vv.(type) {
			case string:
				// when user sets the setting
				// no-op
			case int64:
				// when setting comes from metakv
				mode := PlanStabilityMode(vv)
				if mode >= PS_MODE_OFF && mode <= PS_MODE_AD_HOC_READ_ONLY {
					planStability[kk] = mode.String()
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			case PlanStabilityMode:
				mode := vv
				if mode >= PS_MODE_OFF && mode <= PS_MODE_AD_HOC_READ_ONLY {
					planStability[kk] = mode.String()
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			default:
				return nil, errors.NewSettingsInvalidType(PLAN_STABILITY+".mode", "string", vv)
			}
		case "error_policy":
			switch vv := vv.(type) {
			case string:
				// when user sets the setting
				// no-op
			case int64:
				// when setting comes from metakv
				error_policy := PlanStabilityErrorPolicy(vv)
				if error_policy >= PS_ERROR_FLEXIBLE && error_policy <= PS_ERROR_STRICT {
					planStability[kk] = error_policy.String()
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "", vv)
				}
			case PlanStabilityErrorPolicy:
				error_policy := vv
				if error_policy >= PS_ERROR_FLEXIBLE && error_policy <= PS_ERROR_STRICT {
					planStability[kk] = error_policy.String()
				} else {
					return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+".error_policy", "", vv)
				}
			default:
				return nil, errors.NewSettingsInvalidType(PLAN_STABILITY+".error_policy", "string", vv)
			}
		default:
			return nil, errors.NewSettingsInvalidValue(PLAN_STABILITY+"."+kk, "", nil)
		}
	}

	return planStability, nil
}

func DeleteUdfPrepared(udfName string) errors.Error {
	if IsPlanStabilityEnabled() {
		return planCache.DeleteUdfPrepared(udfName, GetPlanStabilityMode(), GetPlanStabilityErrorPolicy())
	}

	return nil
}
