//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package settings

import (
	"encoding/json"
	"sync"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const (
	_SETTINGS_PATH     = "/query/settings/"
	_SETTINGS_SETTINGS = _SETTINGS_PATH + "global_settings"
)

var _accepted_settings map[string]bool = map[string]bool{
	"plan_stability": true,
}

func InitSettings() {
	globalSettings = new(querySettings)
	globalSettings.settings = make(map[string]interface{}, 4)

	val, _, err := metakv.Get(_SETTINGS_SETTINGS)
	if err != nil {
		logging.Errorf("SETTINGS: Error getting global settings from metakv: %v", err)
	} else if len(val) == 0 {
		// save default settings if not already there
		defSettings := defaultSettings()
		defSettings["node"] = distributed.RemoteAccess().WhoAmI()
		bytes, err := json.Marshal(defSettings)
		if err != nil {
			logging.Errorf("SETTINGS: Error marshalling default settings: %v", err)
		} else {
			err = metakv.Add(_SETTINGS_SETTINGS, bytes)
			if err != nil && err != metakv.ErrRevMismatch {
				logging.Errorf("SETTINGS: Error adding default settings to metakv: %v", err)
			}
		}
		// globalSettings shares defSettings, so remove the "node"
		delete(defSettings, "node")
	} else {
		// process initial settings
		processSettings(val, "")
	}

	// monitor entry
	go metakv.RunObserveChildren(_SETTINGS_PATH, callback, make(chan struct{}))
}

func callback(kve metakv.KVEntry) error {
	processSettings(kve.Value, distributed.RemoteAccess().WhoAmI())
	return nil
}

func processSettings(val []byte, thisNode string) {
	var vmap map[string]interface{}
	err := json.Unmarshal(val, &vmap)
	if err != nil {
		logging.Errorf("SETTINGS: Error unmarshalling new global settings: %v", err)
		return
	}

	if n, ok := vmap["node"]; ok {
		if thisNode != "" && n == thisNode {
			// no need to process settings from this node
			return
		}
		delete(vmap, "node")
	}

	logging.Infof("SETTINGS: New global settings received: %v", vmap)

	removed := make(map[string]interface{})
	invalid := make(map[string]interface{})
	globalSettings.Lock()
	for k, v := range globalSettings.settings {
		if _, ok := vmap[k]; !ok {
			removed[k] = v
		}
	}
	for k, v := range vmap {
		if _, ok := _accepted_settings[k]; ok {
			globalSettings.settings[k] = v
		} else {
			invalid[k] = v
		}
	}
	logging.Infof("SETTINGS: updated settings: %v", globalSettings.settings)
	globalSettings.Unlock()

	if len(removed) > 0 {
		logging.Infof("SETTINGS: settings removed: %v", removed)
	}
	if len(invalid) > 0 {
		logging.Infof("SETTINGS: invalid settings: %v", invalid)
	}

	logging.Infof("SETTINGS: Processing of new global settings completed.")
}

func defaultSettings() map[string]interface{} {
	rv := map[string]interface{}{
		"plan_stability": map[string]interface{}{
			"enabled": false,
		},
	}
	globalSettings.Lock()
	globalSettings.settings = rv
	globalSettings.Unlock()
	return rv
}

func FetchSettings() (map[string]interface{}, errors.Error) {
	val, _, err := metakv.Get(_SETTINGS_SETTINGS)
	if err != nil {
		return nil, errors.NewSettingsMetaKVError(err, "Error getting global settings from metakv")
	}

	var vmap map[string]interface{}
	err = json.Unmarshal(val, &vmap)
	if err != nil {
		return nil, errors.NewSettingsMetaKVError(err, "Error unmarshalling global settings")
	}
	delete(vmap, "node")

	return vmap, nil
}

func UpdateSettings(settings interface{}) (errors.Error, errors.Errors) {
	if actual, ok := settings.(value.Value); ok {
		settings = actual.Actual()
	}
	settingsMap, ok := settings.(map[string]interface{})
	if !ok {
		return errors.NewSettingsInvalidType("settings", settings), nil
	}

	for k, v := range settingsMap {
		if actual, ok := v.(value.Value); ok {
			v = actual.Actual()
		}

		// When JSON is unmarshalled into an interface, numbers are unmarshalled into float.
		if f, ok := v.(float64); ok && value.IsInt(f) {
			v = int64(f)
		}

		switch k {
		case "plan_stability":
			psMap, ok := v.(map[string]interface{})
			if !ok {
				return errors.NewSettingsInvalidValue("plan_stability", "map[string]interface{}", v), nil
			}
			// getSettings() returns a copy of the settings
			psSetting := globalSettings.getSetting(k)
			planStability, ok := psSetting.(map[string]interface{})
			if !ok {
				return errors.NewSettingsInvalidValue("plan_stability", "map[string]interface{}", psSetting), nil
			}
			for kk, vv := range psMap {
				switch kk {
				case "enabled":
					planStability[kk] = vv
				default:
					return errors.NewSettingsInvalidValue("plan_stability."+kk, "", nil), nil
				}
			}
			// update settings once all processed for plan stability
			globalSettings.setSetting(k, planStability)
		}
	}

	return nil, nil
}

func PersistSettings() errors.Error {
	allSettings := globalSettings.getAllSettings()
	allSettings["node"] = distributed.RemoteAccess().WhoAmI()
	bytes, err := json.Marshal(allSettings)
	if err != nil {
		logging.Errorf("SETTINGS: Error marshalling global settings: %v", err)
		return errors.NewSettingsError(err, "Error marshalling global settings")
	}
	err = metakv.Set(_SETTINGS_SETTINGS, bytes, nil)
	if err != nil {
		logging.Errorf("SETTINGS: Error updating global settings in metakv: %v", err)
		return errors.NewSettingsMetaKVError(err, "Error updating global settings in metakv")
	}
	return nil
}

var globalSettings *querySettings

type querySettings struct {
	sync.RWMutex
	settings map[string]interface{}
}

func (this *querySettings) getAllSettings() map[string]interface{} {
	var allSettings map[string]interface{}
	this.RLock()
	allSettings = make(map[string]interface{}, len(this.settings))
	for k, v := range this.settings {
		// return a copy when appropriate
		switch setting := v.(type) {
		case map[string]interface{}:
			vmap := make(map[string]interface{}, len(setting))
			for kk, vv := range setting {
				vmap[kk] = vv
			}
			allSettings[k] = vmap
		case bool, string, int64, float64, int32, float32, int, uint, uint32, uint64, uintptr:
			allSettings[k] = setting
		default:
			// leave it as nil to catch unintended usage
		}
	}
	this.RUnlock()

	return allSettings
}

func (this *querySettings) setSetting(name string, value interface{}) {
	this.Lock()
	this.settings[name] = value
	this.Unlock()
}

func (this *querySettings) getSetting(name string) interface{} {
	var rv interface{}
	this.RLock()
	setting, ok := this.settings[name]
	if ok {
		// return a copy when appropriate
		switch setting := setting.(type) {
		case map[string]interface{}:
			vmap := make(map[string]interface{}, len(setting))
			for k, v := range setting {
				vmap[k] = v
			}
			rv = vmap
		case bool, string, int64, float64, int32, float32, int, uint, uint32, uint64, uintptr:
			rv = setting
		default:
			// leave it as nil to catch unintended usage
		}
	}
	this.RUnlock()

	return rv
}

func SetSetting(name string, value interface{}) {
	globalSettings.setSetting(name, value)
}

func GetSetting(name string) interface{} {
	return globalSettings.getSetting(name)
}
