//  Copyright 2025-Present Couchbase, Inc.
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
	"strings"
	"sync"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const (
	_SETTINGS_PATH     = "/query/query_managed_settings/"
	_SETTINGS_SETTINGS = _SETTINGS_PATH + "global_settings"
)

const (
	PLAN_STABILITY = "plan_stability"
)

func InitSettings() {
	globalSettings = new(querySettings)
	globalSettings.settings = make(map[string]any, 4)

	val, rev, err := metakv.Get(_SETTINGS_SETTINGS)
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
		setSettingsRev(rev)
		processSettings(val, "", true)
	}

	// monitor entry
	go metakv.RunObserveChildren(_SETTINGS_PATH, callback, make(chan struct{}))
}

func callback(kve metakv.KVEntry) error {
	if kve.Path == _SETTINGS_SETTINGS {
		// including the settings written by this node, which processSettings() skips
		setSettingsRev(kve.Rev)
		processSettings(kve.Value, distributed.RemoteAccess().WhoAmI(), false)
	}
	return nil
}

// MB-75471: serializes changes to the settings on this node: updates (with their write to metakv), and
// settings received from metakv
var settingsUpdateLock sync.Mutex

// initial: the settings read at startup
func processSettings(val []byte, thisNode string, initial bool) {
	var vmap map[string]any
	err := json.Unmarshal(val, &vmap)
	if err != nil {
		logging.Errorf("SETTINGS: Error unmarshalling new global settings: %v", err)
		return
	}

	// not while an update on this node is in progress: an update that fails sets the old setting back
	// MB-75508: this includes checking for settings from this node: an update made since (e.g. while these settings
	// waited for the lock) has replaced, or cleared, the settings this node is writing (see takeUnconfirmedWrite())
	settingsUpdateLock.Lock()
	defer settingsUpdateLock.Unlock()

	if n, ok := vmap["node"]; ok {
		if thisNode != "" && n == thisNode {
			// no need to process settings from this node, unless the write reported an error (MB-75508)
			if !takeUnconfirmedWrite(val) {
				return
			}
			logging.Infof("SETTINGS: global settings written by this node, which reported an error, were " +
				"written: processing them")
		}
		delete(vmap, "node")
	}

	logging.Infof("SETTINGS: New global settings received: %v", vmap)

	// a plan stability setting not present in the new settings is not changed
	oldMode := GetPlanStabilityMode()

	removed := make(map[string]any)
	invalid := make(map[string]any)
	globalSettings.Lock()
	for k, v := range globalSettings.settings {
		if _, ok := vmap[k]; !ok {
			removed[k] = v
		}
	}
	for k, v := range vmap {
		switch k {
		case PLAN_STABILITY:
			planStability, err := validatePlanStabilitySetting(v)
			if err != nil {
				logging.Errorf("SETTINGS: Error processing plan stability settings: %v", err)
			} else {
				globalSettings.settings[k] = planStability
			}
		default:
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

	// MB-75471: a plan stability mode change made on another node: update this node's prepared cache (the
	// node that made the change updates the saved plans); not for the initial settings, when the prepared
	// cache is empty
	if newMode := GetPlanStabilityMode(); !initial && newMode != oldMode && planCache != nil {
		planCache.ApplyPlanStabilityMode(oldMode, newMode)
	}

	logging.Infof("SETTINGS: Processing of new global settings completed.")
}

func defaultSettings() map[string]any {
	rv := map[string]any{
		PLAN_STABILITY: defaultPlanStabilitySettings(),
	}
	globalSettings.Lock()
	globalSettings.settings = rv
	globalSettings.Unlock()
	return rv
}

func FetchSettings(remap bool) (map[string]any, errors.Error) {
	val, _, err := metakv.Get(_SETTINGS_SETTINGS)
	if err != nil {
		return nil, errors.NewSettingsMetaKVError(err, "Error getting global settings from metakv")
	}

	var vmap map[string]any
	err = json.Unmarshal(val, &vmap)
	if err != nil {
		return nil, errors.NewSettingsMetaKVError(err, "Error unmarshalling global settings")
	}
	delete(vmap, "node")

	if remap {
		if planStability, ok := vmap[PLAN_STABILITY]; ok {
			remapPlanStability, err := remapPlanStabilitySetting(planStability)
			if err != nil {
				return nil, err
			}
			vmap[PLAN_STABILITY] = remapPlanStability
		}
	}

	return vmap, nil
}

// update the settings, and write them to metakv
func UpdatePersistSettings(enterprise bool, requestId string, settings any) (errors.Error, errors.Errors) {
	// MB-75471: the update and its write to metakv under the lock, so that updates are written in the
	// order they are made
	settingsUpdateLock.Lock()
	defer settingsUpdateLock.Unlock()

	old := globalSettings.getAllSettings()
	err, warnings := updateSettings(enterprise, requestId, settings)
	if err == nil {
		err = PersistSettings()
		if err != nil {
			// the other nodes do not get the new settings, keep the old ones
			restoreSettings(old)
		}
	}
	return err, warnings
}

// set back settings returned by getAllSettings() (a copy), e.g. when writing new settings to metakv fails;
// the prepared cache and the saved plans are not changed back
func restoreSettings(old map[string]any) {
	globalSettings.Lock()
	globalSettings.settings = old
	globalSettings.Unlock()
}

func updateSettings(enterprise bool, requestId string, settings any) (errors.Error, errors.Errors) {
	if actual, ok := settings.(value.Value); ok {
		settings = actual.Actual()
	}
	settingsMap, ok := settings.(map[string]any)
	if !ok {
		return errors.NewSettingsInvalidType("settings", "", settings), nil
	}

	hasPlanStability := false
	var invalid []string
	for k, v := range settingsMap {
		switch k {
		case PLAN_STABILITY:
			// valid setting
			hasPlanStability = true
		default:
			invalid = append(invalid, fmt.Sprintf("'%s':'%v'", k, v))
		}
	}
	if len(invalid) > 0 {
		errMsg := fmt.Sprintf("{ %s }", strings.Join(invalid, ","))
		logging.Errorf("SETTINGS: Invalid settings specified in UPDATE statement: %v", errMsg)
		return errors.NewSettingsError(nil, fmt.Sprintf("Invalid settings specified: %v", errMsg)), nil
	}

	// if the new document does not contain plan stability settings (e.g. UNSET used), use default
	if !hasPlanStability {
		settingsMap[PLAN_STABILITY] = defaultPlanStabilitySettings()
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
		case PLAN_STABILITY:
			err := updatePlanStabilitySetting(requestId, v)
			if err != nil {
				logging.Errorf("SETTINGS: Error updating Plan Stability setting: %v", err)
				return err, nil
			}
		}
	}

	return nil, nil
}

// the metakv revision of the settings this node last saw
var settingsRev struct {
	sync.Mutex
	rev any
}

func setSettingsRev(rev any) {
	if b, ok := rev.([]byte); ok && len(b) == 0 {
		rev = nil
	}
	settingsRev.Lock()
	settingsRev.rev = rev
	settingsRev.Unlock()
}

func getSettingsRev() any {
	settingsRev.Lock()
	rev := settingsRev.rev
	settingsRev.Unlock()
	return rev
}

func getMetakvSettings() ([]byte, any, error) {
	return metakv.Get(_SETTINGS_SETTINGS)
}

func setMetakvSettings(val []byte, rev any) error {
	return metakv.Set(_SETTINGS_SETTINGS, val, rev)
}

func marshalSettings() ([]byte, errors.Error) {
	allSettings := globalSettings.getAllSettings()
	allSettings["node"] = distributed.RemoteAccess().WhoAmI()
	bytes, err := json.Marshal(allSettings)
	if err != nil {
		logging.Errorf("SETTINGS: Error marshalling global settings: %v", err)
		return nil, errors.NewSettingsError(err, "Error marshalling global settings")
	}
	return bytes, nil
}

func PersistSettings() errors.Error {
	bytes, er := marshalSettings()
	if er != nil {
		return er
	}
	setUnconfirmedWrite(bytes)
	err := setStoredSettings(bytes, nil)
	if err != nil {
		logging.Errorf("SETTINGS: Error updating global settings in metakv: %v", err)
		return errors.NewSettingsMetaKVError(err, "Error updating global settings in metakv")
	}
	setUnconfirmedWrite(nil)
	return nil
}

// MB-75508: the settings this node is writing to metakv, until the write succeeds: a write that reports an error
// may have been written all the same (e.g. a timeout once written). The caller then sets the old settings back,
// while the other nodes get the new settings; when they come back from metakv (see callback()), this node
// processes them too, rather than skipping its own settings, so that it does not stay on the old settings. Only
// the last write is kept: settings written before, which come back after (e.g. late), are skipped as usual.
// Writes are made, and the settings coming back checked (processSettings()), under settingsUpdateLock: settings
// that come back while their write is in progress are checked once it completes (and are skipped if it succeeded).
var unconfirmedWrite struct {
	sync.Mutex
	val []byte
}

func setUnconfirmedWrite(val []byte) {
	unconfirmedWrite.Lock()
	unconfirmedWrite.val = val
	unconfirmedWrite.Unlock()
}

// whether val is the settings this node is writing, or failed to write, to metakv; if so it is no longer kept
func takeUnconfirmedWrite(val []byte) bool {
	unconfirmedWrite.Lock()
	rv := unconfirmedWrite.val != nil && bytes.Equal(unconfirmedWrite.val, val)
	if rv {
		unconfirmedWrite.val = nil
	}
	unconfirmedWrite.Unlock()
	return rv
}

var globalSettings *querySettings

type querySettings struct {
	sync.RWMutex
	settings map[string]any
}

func (this *querySettings) getAllSettings() map[string]any {
	var allSettings map[string]any
	this.RLock()
	allSettings = make(map[string]any, len(this.settings))
	for k, v := range this.settings {
		// return a copy when appropriate
		switch setting := v.(type) {
		case map[string]any:
			vmap := make(map[string]any, len(setting))
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

func (this *querySettings) setSetting(name string, value any) {
	this.Lock()
	this.settings[name] = value
	this.Unlock()
}

func (this *querySettings) getSetting(name string) any {
	var rv any
	this.RLock()
	setting, ok := this.settings[name]
	if ok {
		// return a copy when appropriate
		switch setting := setting.(type) {
		case map[string]any:
			vmap := make(map[string]any, len(setting))
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

func SetSetting(name string, value any) {
	globalSettings.setSetting(name, value)
}

func GetSetting(name string) any {
	return globalSettings.getSetting(name)
}
