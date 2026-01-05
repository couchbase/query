//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package settings

import (
	"strings"

	"github.com/couchbase/query/errors"
)

type PLAN_STABILITY_MODE int

const (
	_PS_MODE_OFF = PLAN_STABILITY_MODE(iota)
	_PS_MODE_PREPARED_ONLY
	_PS_MODE_AD_HOC
)

var _PS_MODE_MAP map[string]PLAN_STABILITY_MODE = map[string]PLAN_STABILITY_MODE{
	"off":           _PS_MODE_OFF,
	"prepared_only": _PS_MODE_PREPARED_ONLY,
	"ad_hoc":        _PS_MODE_AD_HOC,
}

func defaultPlanStabilitySettings() map[string]interface{} {
	return map[string]interface{}{
		"mode": _PS_MODE_OFF,
	}
}

func updatePlanStabilitySetting(val interface{}) errors.Error {
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
	for kk, vv := range psMap {
		switch kk {
		case "mode":
			switch vv := vv.(type) {
			case string:
				// when user sets the setting
				if mode, ok := _PS_MODE_MAP[strings.ToLower(vv)]; ok {
					planStability[kk] = mode
				} else {
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "'off'/'prepared_only'/'ad_hoc'", vv)
				}
			case int64:
				// when setting comes from metakv
				mode := PLAN_STABILITY_MODE(vv)
				if mode >= _PS_MODE_OFF && mode <= _PS_MODE_AD_HOC {
					planStability[kk] = mode
				} else {
					return errors.NewSettingsInvalidValue(PLAN_STABILITY+".mode", "", vv)
				}
			default:
				return errors.NewSettingsInvalidType(PLAN_STABILITY+".mode", "string", vv)
			}
		default:
			return errors.NewSettingsInvalidValue(PLAN_STABILITY+"."+kk, "", nil)
		}
	}
	// update settings once all processed for plan stability
	globalSettings.setSetting(PLAN_STABILITY, planStability)

	return nil
}

func getPlanStabilityMode() PLAN_STABILITY_MODE {
	mode := _PS_MODE_OFF
	globalSettings.RLock()
	setting_val := globalSettings.settings[PLAN_STABILITY]
	if ps_setting, ok := setting_val.(map[string]interface{}); ok {
		if ps_mode, ok := ps_setting["mode"].(PLAN_STABILITY_MODE); ok {
			mode = ps_mode
		}
	}
	globalSettings.RUnlock()
	return mode
}

func IsPlanStabilityEnabled() bool {
	mode := getPlanStabilityMode()
	return mode == _PS_MODE_PREPARED_ONLY || mode == _PS_MODE_AD_HOC
}

func IsPlanStabilityDisabled() bool {
	mode := getPlanStabilityMode()
	return mode == _PS_MODE_OFF
}

func IsPlanStabilityPreparedOnly() bool {
	mode := getPlanStabilityMode()
	return mode == _PS_MODE_PREPARED_ONLY
}

func IsPlanStabilityAdHoc() bool {
	mode := getPlanStabilityMode()
	return mode == _PS_MODE_AD_HOC
}
