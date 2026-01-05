//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package aus

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth/metakv"
	query_ee "github.com/couchbase/query-ee"
	"github.com/couchbase/query-ee/aus"
	"github.com/couchbase/query-ee/aus/bridge"
	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/migration"
	"github.com/couchbase/query/scheduler"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	_AUS_PATH                    = "/query/auto_update_statistics/"
	_AUS_GLOBAL_SETTINGS_PATH    = _AUS_PATH + "global_settings"
	_AUS_SETTINGS_BUCKET_DOC_KEY = _AUS_SETTING_DOC_PREFIX + "bucket"

	_AUS_SETTING_DOC_PREFIX              = "aus_setting::"
	_AUS_COORDINATION_DOC_PREFIX         = "aus_coord::"
	_AUS_CLEANUP_COORDINATION_DOC_PREFIX = "aus_coord_cleanup::"
	_AUS_CHANGE_DOC_PREFIX               = "aus_change_history::"

	_MAX_RETRY                 = 10
	_RETRY_INTERVAL            = 30 * time.Second
	_SCHEDULING_RETRY_INTERVAL = 5 * time.Second
	_MAX_LOAD_FACTOR           = 80
	_BATCH_SIZE                = 16
)

var ausCfg ausConfig

// since we fetch documents from _system._query 1 at a time
var _STRING_ANNOTATED_POOL *value.StringAnnotatedPool

var defaultKeyspaceSettings = keyspaceSettings{enable: value.NONE, changePercentage: -1, update_stats_timeout: -1}

type ausConfig struct {
	sync.RWMutex
	server      *server.Server
	settings    ausGlobalSettings
	initialized bool

	// Indicates the version of the global settings that was used to schedule/ enable the current task.
	// This field is only changes when the new global settings received, change the enablement or schedule.
	configVersion string

	// Stores information about the running tasks window
	// This is to prevent multiple tasks from running at once on the node.
	runningAusEnd     time.Time
	runningCleanupEnd time.Time
}

type ausGlobalSettings struct {
	enable           bool
	allBuckets       bool
	changePercentage int
	schedule         ausSchedule
	// Whether AUS should create statistics that are missing.
	// If false, AUS will only update existing statistics.
	createMissing bool
	version       string
}

type ausSchedule struct {
	startTime time.Time
	endTime   time.Time
	timezone  *time.Location
	days      []bool
}

const (
	_AUS_TASK_CLASS         = "auto_update_statistics"
	_AUS_CLEANUP_TASK_CLASS = "auto_update_statistics_cleanup"
)

type taskInfo struct {
	startTime time.Time
	endTime   time.Time

	// The configVersion that was used to schedule this task.
	// This field should be used to perform an equality check against the current global configVersion before the task is
	// executed or [re]scheduled. If the check fails, it means that there were new settings received that changed the config and
	// the task operation must not proceed.
	version     string
	class       string
	sessionName string
}

func (this *ausConfig) setInitialized(init bool) {
	this.Lock()
	this.initialized = init
	this.Unlock()
}

func (this *ausConfig) setRunningAusEnd(lock bool, end time.Time) {
	if lock {
		this.Lock()
		defer this.Unlock()
	}

	ausCfg.runningAusEnd = end
}

func (this *ausConfig) setRunningCleanupEnd(lock bool, end time.Time) {
	if lock {
		this.Lock()
		defer this.Unlock()
	}

	ausCfg.runningCleanupEnd = end
}

func (this *ausSchedule) equals(sched *ausSchedule) bool {
	if len(this.days) != len(sched.days) {
		return false
	}

	for i, d := range this.days {
		if sched.days[i] != d {
			return false
		}
	}

	return this.startTime.Equal(sched.startTime) &&
		this.endTime.Equal(sched.endTime) &&
		this.timezone.String() == sched.timezone.String()
}

// Performs basic initialization of AUS configuration
func InitAus(server *server.Server) {
	ausCfg = ausConfig{
		server:      server,
		initialized: false,
	}

	bridge.SetAusEnabled(Enabled)
	// run as go routine to prevent hanging at the migration.Await() call
	go startupAus()
}

// Returns if AUS is enabled
func Enabled() bool {
	return ausCfg.initialized && ausCfg.settings.enable
}

// Returns if AUS is initialized
func Initialized() bool {
	return ausCfg.initialized
}

// Sets up AUS for the node. Must only be called post-migration of CBO stats.
func startupAus() {

	if ausCfg.initialized || tenant.IsServerless() {
		return
	}

	// If migration has completed - either the state of migration is MIGRATED or ABORTED - allow AUS to be initialized.
	// Otherwise wait for migration to complete
	if m, _ := migration.IsComplete("CBO_STATS"); !m {
		logging.Infof("AUS: Initialization will start when migration to a supported version is complete.")
		migration.Await("CBO_STATS")
	}

	logging.Infof("AUS: Initialization started.")

	// Initialize the system:aus_settings Fetch pool
	_STRING_ANNOTATED_POOL = value.NewStringAnnotatedPool(1)

	// Initialize the global settings in metakv
	err := initMetakv()
	if err != nil {
		logging.Errorf("AUS: Metakv initialization failed with error: %v", err)
	}

	go metakv.RunObserveChildren(_AUS_PATH, callback, make(chan struct{}))
	ausCfg.setInitialized(true)

	logging.Infof("AUS: Initialization completed.")
}

func callback(kve metakv.KVEntry) error {
	ausCfg.Lock()
	logging.Infof("AUS: New global settings received.")

	var v map[string]interface{}
	err := json.Unmarshal(kve.Value, &v)
	if err != nil {
		ausCfg.Unlock()
		logging.Errorf("AUS: Error unmarshalling new global settings: %v", err)
		return nil
	}

	if newSettings, err, _ := setAusHelper(v, false, false); err != nil {
		ausCfg.Unlock()
		logging.Errorf("AUS: Error during global settings retrieval: %v", err)
		return nil
	} else {

		prev := ausCfg.settings
		ausCfg.settings = newSettings

		var version string
		var reschedule bool  // whether a new AUS task should be scheduled
		var cancelOld bool   // whether any scheduled AUS tasks should be deleted
		var nowDisabled bool // whether AUS has been changed from enabled to disabled
		var nowEnabled bool  // whether AUS has been changed from disabled to enabled

		if prev.enable {
			if !newSettings.enable {
				// If AUS enablement changes from enabled to disabled
				cancelOld = true
				nowDisabled = true
			} else if !prev.schedule.equals(&newSettings.schedule) {
				// If AUS remains enabled but the schedule has changed
				reschedule = true
				cancelOld = true
			}
		} else if !prev.enable && newSettings.enable {
			// If AUS enablement changes from disabled to enabled
			reschedule = true
			nowEnabled = true
		}

		// Only if there is a change in the enablement/schedule change the version of the configuration
		if reschedule || cancelOld {
			ausCfg.configVersion = newSettings.version
		}

		version = ausCfg.configVersion
		ausCfg.Unlock()

		logging.Infof("AUS: Global settings are now: %v with internal version: %v.", v, version)

		if cancelOld || nowEnabled {
			// Delete all scheduled AUS tasks. But let running tasks complete execution.
			var taskId string
			scheduler.TasksForeach(func(name string, task *scheduler.TaskEntry) bool {
				taskId = ""
				if cancelOld && (task.Class == _AUS_TASK_CLASS && task.State == scheduler.SCHEDULED) {
					taskId = task.Id
				} else if nowEnabled && (task.Class == _AUS_CLEANUP_TASK_CLASS && task.State == scheduler.SCHEDULED) {
					taskId = task.Id
				}
				return true

			}, func() bool {
				if taskId != "" {
					execErr := scheduler.DeleteTask(taskId)
					if execErr != nil {
						logging.Errorf("AUS: Error deleting old scheduled task with id %v: %v", taskId, execErr)
					}
				}
				return true
			}, scheduler.SCHEDULED_TASKS_CACHE)
		}

		// Schedule next task with the new schedule
		if reschedule {
			ausCfg.scheduleAusTask(version, true)
		}

		if nowDisabled {
			ausCfg.scheduleCleanupTask(version)
		}
	}

	return nil
}

// Initialize the global settings in metakv
func initMetakv() error {
	_, _, err := fetchAusHelper(false)
	return err
}

// system:aus keyspace related functions

func CountAus() (int64, errors.Error) {
	if !ausCfg.initialized {
		return -1, errors.NewAusNotInitialized()
	}

	return 1, nil
}

func FetchAus() (map[string]interface{}, errors.Error) {
	if !ausCfg.initialized {
		return nil, errors.NewAusNotInitialized()
	}

	bytes, rev, err := fetchAusHelper(true)
	if err != nil {
		return nil, errors.NewAusStorageAccessError(err)
	}

	// Path does not exist in metakv
	if bytes == nil && rev == nil && err == nil {
		return nil, nil
	}

	var val map[string]interface{}
	err = json.Unmarshal(bytes, &val)
	if err != nil {
		return nil, errors.NewAusStorageAccessError(errors.NewAusDocEncodingError(false, err))
	}

	// Version field is only required for internal use.
	delete(val, "internal_version")

	return val, nil
}

// For the global settings path/ key that is to be in metakv:
// Checks if the path/ key is present in metakv
// If not present, attempts to add the path to metakv with the default global settings value
// get: if the value of the key are to be returned
func fetchAusHelper(get bool) (rv []byte, rev interface{}, err error) {
	rv, rev, err = metakv.Get(_AUS_GLOBAL_SETTINGS_PATH)
	if err != nil {
		logging.Errorf("AUS: Error retrieving global settings from metakv: %v", err)
		return nil, nil, err
	}

	// Only add the default global settings document in metakv if the path is non-existent
	if rv == nil && rev == nil && err == nil {

		// The default global settings document
		bytes, err := json.Marshal(map[string]interface{}{
			"enable":                    false,
			"all_buckets":               false,
			"change_percentage":         10,
			"internal_version":          "default",
			"create_missing_statistics": false,
		})

		if err != nil {
			logging.Errorf("AUS: Error encoding default global settings document: %v", err)
			return nil, nil, err
		}

		err = metakv.Add(_AUS_GLOBAL_SETTINGS_PATH, bytes)

		// If a non 200 OK status is returned from ns_server
		if err != nil && err != metakv.ErrRevMismatch {
			logging.Errorf("AUS: Error distributing default global settings via metakv: %v", err)
			return nil, nil, err
		}

		if get {
			// Attempt to get the latest value for the key
			rv, rev, err = metakv.Get(_AUS_GLOBAL_SETTINGS_PATH)
			if err != nil {
				logging.Errorf("AUS: Error retrieving global settings from metakv: %v", err)
			}
		}
	}

	if get {
		return rv, rev, err
	} else {
		return nil, nil, err
	}
}

func SetAus(settings interface{}, distribute bool, restore bool) (err errors.Error, warnings errors.Errors) {
	if !ausCfg.initialized {
		return errors.NewAusNotInitialized(), nil
	}

	_, err, warnings = setAusHelper(settings, true, restore)
	return err, warnings
}

// Function to validate schema of the input settings document. And optionally distribute said settings document in metakv
func setAusHelper(settings interface{}, distribute bool, restore bool) (obj ausGlobalSettings, err errors.Error,
	warnings errors.Errors) {

	if actual, ok := settings.(value.Value); ok {
		settings = actual.Actual()
	}

	if settings, ok := settings.(map[string]interface{}); !ok {
		return obj, errors.NewAusDocInvalidSettingsValue("settings", settings), nil
	} else {
		var isEnabled bool
		for k, v := range settings {
			if actual, ok := v.(value.Value); ok {
				v = actual.Actual()
			}

			// When JSON is unmarshalled into an interface, numbers are unmarshalled into float.
			if f, ok := v.(float64); ok && value.IsInt(f) {
				v = int64(f)
			}

			switch k {
			case "enable":
				if e, ok := v.(bool); !ok {
					return obj, errors.NewAusDocInvalidSettingsValue(k, v), nil
				} else if e {
					obj.enable = true
					isEnabled = true
				}
			case "all_buckets":
				if ab, ok := v.(bool); !ok {
					return obj, errors.NewAusDocInvalidSettingsValue(k, v), nil
				} else {
					obj.allBuckets = ab
				}
			case "change_percentage":
				if cp, ok := v.(int64); !ok {
					return obj, errors.NewAusDocInvalidSettingsValue(k, v), nil
				} else if cp < 0 || cp > 100 {
					return obj, errors.NewAusDocInvalidSettingsValue(k, cp), nil
				} else {
					obj.changePercentage = int(cp)
				}
			case "schedule":
				sched, err, warn := validateSchedule(v)
				if len(warn) > 0 {
					warnings = append(warnings, warn...)
				}
				if err != nil {
					return obj, err, warnings
				}
				obj.schedule = sched
			case "create_missing_statistics":
				if c, ok := v.(bool); !ok {
					return obj, errors.NewAusDocInvalidSettingsValue(k, v), nil
				} else {
					obj.createMissing = c
				}
			case "internal_version":
				if vv, ok := v.(string); ok {
					obj.version = vv
				} else {
					logging.Warnf("AUS: internal_version is not a string: %v", v)
				}
			default:
				return obj, errors.NewAusDocUnknownSetting(k), nil
			}
		}

		// For mandatory settings - set default values if not present in document.

		// If "enable" is not set, the default value is "false"
		if _, ok := settings["enable"]; !ok {
			settings["enable"] = false
			warnings = append(warnings, errors.NewAusDocMissingSetting("enable", false))
		}

		// Make a semantic check - if AUS is enabled then a valid schedule must be set.
		if isEnabled {
			if _, ok := settings["schedule"]; !ok {
				return obj, errors.NewAusDocInvalidSemantics("schedule"),
					warnings
			}
		}

		// If "all_buckets" is not set, the default value is "false"
		if _, ok := settings["all_buckets"]; !ok {
			settings["all_buckets"] = false
			obj.allBuckets = false
			warnings = append(warnings, errors.NewAusDocMissingSetting("all_buckets", false))
		}

		// If "change_percentage" is not set, the default value is 10%
		if _, ok := settings["change_percentage"]; !ok {
			settings["change_percentage"] = 10
			obj.changePercentage = 10
			warnings = append(warnings, errors.NewAusDocMissingSetting("change_percentage", 10))
		}

		// If "create_missing_statistics" is not set, the default value is "false"
		if _, ok := settings["create_missing_statistics"]; !ok {
			settings["create_missing_statistics"] = false
			obj.createMissing = false
			warnings = append(warnings, errors.NewAusDocMissingSetting("create_missing_statistics", false))
		}

		// Add the new settings document to metakv
		if distribute {

			// If restoring the settings, do not change the version.
			if !restore {
				// Add a random UUID as the "internal_version" field
				version, err := util.UUIDV4()
				if err != nil {
					return obj, errors.NewAusDocEncodingError(true, err), warnings
				}
				settings["internal_version"] = version
			}

			bytes, err := json.Marshal(settings)
			if err != nil {
				return obj, errors.NewAusDocEncodingError(true, err), warnings
			}

			err = metakv.Set(_AUS_GLOBAL_SETTINGS_PATH, bytes, nil)

			// This should be rare but if the global settings path is not present in metakv, attempt to create it.
			if metakvIsNotFoundError(err) {
				err = metakv.Add(_AUS_GLOBAL_SETTINGS_PATH, bytes)
			}

			// If the previous path creation attempt also fails - return error.
			if err != nil {
				return obj, errors.NewAusStorageAccessError(err), warnings
			}
		}

		return obj, nil, warnings

	}

}

func validateTime(val interface{}) (time.Time, bool) {
	if va, ok := val.(value.Value); ok {
		val = va.Actual()
	}

	if t, ok := val.(string); ok {
		tp, err := time.Parse("15:04", t)
		if err != nil {
			return time.Time{}, false
		}
		return tp, true
	}

	return time.Time{}, false
}

func validateSchedule(schedule interface{}) (ausSchedule ausSchedule, err errors.Error, warnings errors.Errors) {
	if va, ok := schedule.(value.Value); ok {
		schedule = va.Actual()
	}

	if sched, ok1 := schedule.(map[string]interface{}); ok1 {
		// Check if there are any unknown fields
		for k, _ := range sched {
			switch k {
			case "start_time", "end_time", "timezone", "days":
			default:
				return ausSchedule, errors.NewAusDocUnknownSetting("schedule." + k), warnings
			}
		}

		// Validate the "start_time"
		var start time.Time
		if s, ok := sched["start_time"]; ok {
			isValid := false
			start, isValid = validateTime(s)

			if !isValid {
				return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.start_time", s), warnings
			}
		} else {
			return ausSchedule, errors.NewAusDocMissingSetting("schedule.start_time", nil), warnings
		}

		// Validate the "end_time"
		var end time.Time
		if e, ok := sched["end_time"]; ok {
			isValid := false
			end, isValid = validateTime(e)

			if !isValid {
				return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.end_time", e), warnings
			}
		} else {
			return ausSchedule, errors.NewAusDocMissingSetting("schedule.end_time", nil), warnings
		}

		// Make a semantic check to check if "start_time" is before "end_time" at least by 30 minutes
		diff := end.Sub(start)
		if diff.Minutes() < 30 {
			return ausSchedule, errors.NewAusDocInvalidSemantics("schedule.end_time"), warnings
		}

		// Validate the "timezone"
		var timezone *time.Location
		if tz, ok := sched["timezone"]; ok {
			if tza, ok := tz.(value.Value); ok {
				tz = tza.Actual()
			}
			if t, ok := tz.(string); ok {
				var err error
				timezone, err = time.LoadLocation(t)
				if err != nil {
					return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.timezone", t),
						warnings
				}
			} else {
				return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.timezone", t), warnings
			}
		} else {
			var err1 error
			timezone, err1 = time.LoadLocation("UTC")
			if err1 != nil {
				return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.timezone", "UTC"),
					warnings
			}

			sched["timezone"] = "UTC"
			warnings = append(warnings, errors.NewAusDocMissingSetting("schedule.timezone", "UTC"))
		}

		// Validate the "days" list.
		if dv, ok := sched["days"]; ok {
			if dva, ok := dv.(value.Value); ok {
				dv = dva.Actual()
			}
			if days, ok := dv.([]interface{}); ok {
				if len(days) == 0 {
					return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.days", days), warnings
				}

				daysNum := make([]bool, 7)
				for _, d := range days {
					if da, ok := d.(value.Value); ok {
						d = da.Actual()
					}

					if ds, ok := d.(string); ok {
						switch strings.ToLower(strings.TrimSpace(ds)) {
						case "sunday":
							daysNum[time.Sunday] = true
						case "monday":
							daysNum[time.Monday] = true
						case "tuesday":
							daysNum[time.Tuesday] = true
						case "wednesday":
							daysNum[time.Wednesday] = true
						case "thursday":
							daysNum[time.Thursday] = true
						case "friday":
							daysNum[time.Friday] = true
						case "saturday":
							daysNum[time.Saturday] = true
						default:
							return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.days", ds), warnings
						}
					} else {
						return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.days", ds), warnings
					}
				}

				// Since all fields have been validated successfully - set the ausSchedule object
				ausSchedule.startTime = start
				ausSchedule.endTime = end
				ausSchedule.timezone = timezone
				ausSchedule.days = daysNum

			} else {
				return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.days", dv), warnings
			}
		} else {
			return ausSchedule, errors.NewAusDocMissingSetting("schedule.days", nil), warnings
		}
	} else {
		return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule", schedule), warnings
	}

	return ausSchedule, nil, warnings
}

// Metakv does not export the "not found" error returned when the path is not in metakv.
// Helper method for the same
func metakvIsNotFoundError(err error) bool {
	return err != nil && err.Error() == "Not found"
}

// system:aus_settings keyspace related functions

// The scan for system:aus_settings returns the fully qualified paths
// for the keyspaces that have a settings document stored in _system._query.
func ScanAusSettings(bucket string, f func(path string) error) errors.Error {
	if !ausCfg.initialized {
		return errors.NewAusNotInitialized()
	}

	return datastore.ScanSystemCollection(bucket, _AUS_SETTING_DOC_PREFIX, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {

			// Convert the KV document key into a path
			path, _ := key2path(key, systemCollection.NamespaceId(), bucket)
			if path == "" {
				return nil
			}

			err := f(path)
			if err != nil {
				return errors.NewAusStorageAccessError(err)
			}

			return nil
		}, nil)
}

// Given a fully qualified keyspace path, fetches the correesponding settings document stored in _system._query
func FetchAusSettings(path string) (value.Value, errors.Errors) {
	if !ausCfg.initialized {
		return nil, errors.Errors{errors.NewAusNotInitialized()}
	}

	// Validate the path and create a KV document key from it
	key, parts, err := path2key(path)
	if err != nil {
		return nil, errors.Errors{err}
	}

	if len(parts) < 2 || len(parts) > 4 {
		return nil, errors.Errors{errors.NewAusStorageInvalidKey(path, nil)}
	}

	return fetchSystemCollection(parts[1], key)
}

func fetchSystemCollection(bucketName string, key string) (value.Value, errors.Errors) {
	// Get system collection
	systemCollection, err := getSystemCollection(bucketName)
	if err != nil {
		return nil, errors.Errors{err}
	}

	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)

	errs := systemCollection.Fetch([]string{key}, fetchMap, datastore.NULL_QUERY_CONTEXT, nil, nil, false)

	if len(errs) > 0 {
		return nil, errs
	}

	rv := fetchMap[key]

	// return nil, nil if no document was found for the key
	if rv == nil {
		return nil, nil
	}

	return rv, nil

}

// Performs a mutation operation on system:aus_settings
// pair.Name will be a the fully qualified path of a keyspace. This path will be converted into its corresponding KV document key
// pair.Value will be the document to be mutated
// Schema validation will be done on the document if the operation is an UPDATE/ INSERT/ UPSERT
func MutateAusSettings(op MutateOp, pair value.Pair, queryContext datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	if !ausCfg.initialized {
		return 0, nil, errors.Errors{errors.NewAusNotInitialized()}
	}

	// Validate the path and create the corresponding KV document key from it
	key, parts, err := path2key(pair.Name)
	if err != nil {
		return 0, nil, errors.Errors{err}
	}

	if len(parts) < 2 || len(parts) > 4 {
		return 0, nil, errors.Errors{errors.NewAusStorageInvalidKey(pair.Name, nil)}
	}

	// Validate the schema of the document
	switch op {
	case MOP_INSERT, MOP_UPSERT, MOP_UPDATE:
		{
			_, err = validateAusSettingDoc(pair.Value)
			if err != nil {
				return 0, nil, errors.Errors{err}
			}
		}
	}

	// Get system collection for the bucket
	systemCollection, err := getSystemCollection(parts[1])
	if err != nil {
		return 0, nil, errors.Errors{err}
	}

	// Do not send OPTIONS to be mutated. Modifying options for system:aus_settings will not be allowed.
	dpairs := make(value.Pairs, 1)
	dpairs[0].Name = key
	dpairs[0].Value = pair.Value

	switch op {
	case MOP_INSERT:
		return systemCollection.Insert(dpairs, queryContext, preserveMutations)
	case MOP_UPSERT:
		return systemCollection.Upsert(dpairs, queryContext, preserveMutations)
	case MOP_UPDATE:
		return systemCollection.Update(dpairs, queryContext, preserveMutations)
	case MOP_DELETE:
		return systemCollection.Delete(dpairs, queryContext, preserveMutations)
	}

	return 0, nil, nil
}

// Returns the keyspace path and path parts from the KV document key.
// If it returns an empty string, the key is not from system:aus_settings
// Format is like so:
// 1. Bucket level document:
// aus_setting::bucket
// 2. Scope level document:
// aus_setting::scope_id::scope_name
// 3. Collection level document:
// aus_setting::scope_id::collection_id::scope_name.collection_name
func key2path(key, namespace, bucket string) (string, []string) {
	// strip prefix and scope (and collection) UIDs from the scope (and collection) names
	parts := strings.Split(key, "::")

	// Check if the key is in right bucket/ scope/ collection level document key format
	if len(parts) < 2 || len(parts) > 4 {
		return "", nil
	}

	// check if the key is prefixed by "aus_setting::"
	if parts[0] != "aus_setting" {
		return "", nil
	}

	// check if the key was for the bucket
	if parts[1] == "bucket" {
		return algebra.PathFromParts(namespace, bucket), []string{namespace, bucket}
	}

	// the last element in the parts array post-splitting will be the actual path with the scope & collection names
	ks := parts[len(parts)-1]
	dot := strings.IndexByte(ks, '.')

	// if '.' is not present then the key is for a scope level document
	if dot < 0 {
		pathParts := []string{namespace, bucket, ks}
		return algebra.PathFromParts(pathParts...), pathParts
	}

	// Otherwise the key is for a collection level document
	pathParts := []string{namespace, bucket, ks[:dot], ks[dot+1:]}
	return algebra.PathFromParts(pathParts...), pathParts
}

// Validates the input path and returns the KV document key and the path parts.
// The input path must be the fully qualified path
func path2key(path string) (string, []string, errors.Error) {

	parts := algebra.ParsePath(path)

	if len(parts) < 2 || len(parts) > 4 {
		return "", nil, errors.NewAusStorageInvalidKey(path, nil)
	} else if parts[0] != "default" {
		return "", nil, errors.NewAusStorageInvalidKey(path, nil)
	}

	store := datastore.GetDatastore()
	if store == nil {
		return "", nil, errors.NewNoDatastoreError()
	}

	// Check if namespace is valid
	namespace, err := store.NamespaceById(parts[0])
	if err != nil || namespace == nil || namespace.Name() != "default" {
		return "", nil, errors.NewAusStorageInvalidKey(path, err)
	}

	// Check if bucket exists
	bucket, err := namespace.BucketByName(parts[1])
	if err != nil {
		return "", nil, errors.NewAusStorageInvalidKey(path, err)
	}

	if len(parts) == 2 {
		// return default bucket key
		return _AUS_SETTINGS_BUCKET_DOC_KEY, parts, nil
	}

	// Check is scope exists
	scope, err := bucket.ScopeByName(parts[2])
	if err != nil {
		return "", nil, errors.NewAusStorageInvalidKey(path, err)
	}

	if len(parts) == 4 {
		// create a collection document
		collection, err := scope.KeyspaceByName(parts[3])
		if err != nil {
			return "", nil, errors.NewAusStorageInvalidKey(path, err)
		}
		return ausSettingsCollectionKey(scope.Uid(), parts[2], collection.Uid(), parts[3]), parts, nil
	} else {
		// create a scope document
		return ausSettingsScopeKey(scope.Uid(), parts[2]), parts, nil
	}

}

// Returns document id of a scope level document in system:aus_settings
func ausSettingsScopeKey(scopeUid string, scopeName string) string {
	sb := strings.Builder{}
	sb.WriteString(_AUS_SETTING_DOC_PREFIX)
	sb.WriteString(scopeUid)
	sb.WriteString("::")
	sb.WriteString(scopeName)
	return sb.String()
}

// Returns document id of a collection level document in system:aus_settings
func ausSettingsCollectionKey(scopeUid string, scopeName string, collectionUid string, collectionName string) string {
	sb := strings.Builder{}
	sb.WriteString(_AUS_SETTING_DOC_PREFIX)
	sb.WriteString(scopeUid)
	sb.WriteString("::")
	sb.WriteString(collectionUid)
	sb.WriteString("::")
	sb.WriteString(scopeName)
	sb.WriteString(".")
	sb.WriteString(collectionName)
	return sb.String()
}

// Get the _system._query collection for a bucket
func getSystemCollection(bucket string) (datastore.Keyspace, errors.Error) {
	store := datastore.GetDatastore()
	if store == nil {
		return nil, errors.NewNoDatastoreError()
	}

	return store.GetSystemCollection(bucket)
}

// Validate the schema of a bucket/scope/collection level document
func validateAusSettingDoc(doc value.Value) (*keyspaceSettings, errors.Error) {

	settings := &keyspaceSettings{enable: value.NONE, changePercentage: -1, update_stats_timeout: -1}

	// Check if there are any disallowed fields or invalid field values
	for k, v := range doc.Fields() {
		if actual, ok := v.(value.Value); ok {
			v = actual.Actual()
		}

		// When JSON is unmarshalled into an interface, numbers are unmarshalled into float
		if f, ok := v.(float64); ok && value.IsInt(f) {
			v = int64(f)
		}

		switch k {
		case "enable":
			if e, ok := v.(bool); !ok {
				return nil, errors.NewAusDocInvalidSettingsValue(k, v)
			} else {
				settings.enable = value.ToTristate(e)
			}
		case "change_percentage":
			if cp, ok := v.(int64); !ok {
				return nil, errors.NewAusDocInvalidSettingsValue(k, v)
			} else if cp < 0 || cp > 100 {
				return nil, errors.NewAusDocInvalidSettingsValue(k, v)
			} else {
				settings.changePercentage = int(cp)
			}
		case "update_statistics_timeout":
			if t, ok := v.(int64); !ok {
				return nil, errors.NewAusDocInvalidSettingsValue("update_statistics_timeout", v)
			} else if t < 0 {
				return nil, errors.NewAusDocInvalidSettingsValue("update_statistics_timeout", v)
			} else {
				settings.update_stats_timeout = int(t)
			}
		default:
			return nil, errors.NewAusDocUnknownSetting(k)
		}
	}

	return settings, nil
}

// Represents the keyspace i.e bucket/scope/collection level setting stored in system:aus_settings
type keyspaceSettings struct {
	enable           value.Tristate
	changePercentage int

	// Options for the UPDATE STATISTICS statements
	update_stats_timeout int
}

// Cleans up scope level AUS documents
func DropScope(namespace string, bucket string, scope string, scopeUid string) {
	if !ausCfg.initialized {
		return
	}

	var context datastore.QueryContext
	dPairs := make(value.Pairs, 0, _BATCH_SIZE)

	for _, prefix := range []string{_AUS_SETTING_DOC_PREFIX, _AUS_CHANGE_DOC_PREFIX} {
		datastore.ScanSystemCollection(bucket, prefix+scopeUid+"::",
			func(systemCollection datastore.Keyspace) errors.Error {
				context = datastore.GetDurableQueryContextFor(systemCollection)
				return nil
			},
			func(key string, systemCollection datastore.Keyspace) errors.Error {
				dPairs = append(dPairs, value.Pair{Name: key})

				if len(dPairs) >= _BATCH_SIZE {
					systemCollection.Delete(dPairs, context, false)
					dPairs = dPairs[:0]
				}

				return nil
			},
			func(systemCollection datastore.Keyspace) errors.Error {
				if len(dPairs) > 0 {
					systemCollection.Delete(dPairs, context, false)
				}
				return nil
			})

		dPairs = dPairs[:0]
	}

}

// Cleans up collection level AUS documents
func DropCollection(namespace string, bucket string, scope string, scopeUid string, collection string, collectionUid string) {
	if !ausCfg.initialized {
		return
	}

	var context datastore.QueryContext
	dPairs := make(value.Pairs, 0, _BATCH_SIZE)

	for _, prefix := range []string{_AUS_SETTING_DOC_PREFIX, _AUS_CHANGE_DOC_PREFIX} {
		datastore.ScanSystemCollection(bucket, prefix+scopeUid+"::"+collectionUid+"::",
			func(systemCollection datastore.Keyspace) errors.Error {
				context = datastore.GetDurableQueryContextFor(systemCollection)
				return nil
			},
			func(key string, systemCollection datastore.Keyspace) errors.Error {
				dPairs = append(dPairs, value.Pair{Name: key})

				if len(dPairs) >= _BATCH_SIZE {
					systemCollection.Delete(dPairs, context, false)
					dPairs = dPairs[:0]
				}

				return nil
			},
			func(systemCollection datastore.Keyspace) errors.Error {
				if len(dPairs) > 0 {
					systemCollection.Delete(dPairs, context, false)
				}
				return nil
			})

		dPairs = dPairs[:0]
	}

}

// Backup related functions

func BackupAusSettings(namespace string, bucket string, filter func([]string) bool) ([]interface{}, errors.Error) {
	rv := make([]interface{}, 0)
	keys := make([]string, 1)

	err := datastore.ScanSystemCollection(bucket, _AUS_SETTING_DOC_PREFIX, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {

			// Convert the KV document key into a path
			path, parts := key2path(key, namespace, bucket)

			if path == "" {
				return nil
			}

			if filter == nil || filter(parts) {

				// Fetch the document
				fetchMap := _STRING_ANNOTATED_POOL.Get()
				defer _STRING_ANNOTATED_POOL.Put(fetchMap)

				keys[0] = key
				errs := systemCollection.Fetch(keys, fetchMap, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
				if errs != nil && len(errs) > 0 {
					return errs[0]
				}
				av, ok := fetchMap[key]

				if ok && av != nil {
					setting := make(map[string]interface{})
					setting["identity"] = path
					v, ok1 := av.Field("enable")
					if ok1 {
						setting["enable"] = v.ToString()
					}

					v, ok1 = av.Field("change_percentage")
					if ok1 {
						setting["change_percentage"] = v.ToString()
					}

					v, ok1 = av.Field("update_statistics_timeout")
					if ok1 {
						setting["update_statistics_timeout"] = v.ToString()
					}

					rv = append(rv, setting)
				}
			}
			return nil
		}, nil)

	return rv, err
}

// Scheduling related functions

// Finds the next AUS window and schedules the task. Returns whether a config version change occurred and an error.
// Parameters:
// checkToday: If true, considers today's schedule when finding the next window and performs overlap check with any running task
func (this *ausConfig) scheduleAusTask(version string, checkToday bool) (bool, errors.Error) {

	this.RLock()
	// Skip scheduling if the task's version differs from current config version,
	// It indicates that newer settings have been received.
	if version != this.configVersion {
		this.RUnlock()
		logging.Infof("AUS: Detected global settings change. Task configured with internal version %v not scheduled.", version)
		return true, nil
	}

	// Check for overlap between the next scheduled window and the currently running task's window
	start, end, err := this.settings.schedule.findNextWindow(checkToday, this.runningAusEnd, this.runningCleanupEnd)
	this.RUnlock()
	if err != nil {
		logging.Errorf("AUS: Error finding window for next task: %v", err)
		return false, err
	}

	task := taskInfo{startTime: start, endTime: end, version: version, class: _AUS_TASK_CLASS}
	err = scheduleTask(task)
	if err != nil {
		return false, err
	}

	return false, nil
}

// Returns the start and end times of the next window for the next AUS task
// Parameters:
// checkToday: If true, considers today's schedule when finding the next window and
// performs overlap check with the provided start and end times specified by the parameters windowStart and windowEnd
func (this *ausSchedule) findNextWindow(checkToday bool, overlaps ...time.Time) (
	time.Time, time.Time, errors.Error) {

	// Get the current time in the required timezone
	now := time.Now().In(this.timezone)

	start := this.startTime
	end := this.endTime

	// check if the next run is Today
	if checkToday && this.days[now.Weekday()] {

		// if the current time is before the start time of the schedule, then this could be the next run.
		if now.Hour() < start.Hour() || (now.Hour() == start.Hour()) && (now.Minute() <= start.Minute()) {

			nextStart := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(),
				start.Minute(), 0, 0, this.timezone)

			today := true

			// Perform the overlap check. Check if the start time of today's potential schedule is within the provided window
			for _, runEnd := range overlaps {
				if !runEnd.IsZero() && !nextStart.After(runEnd) {
					today = false
					break
				}
			}

			if today {
				return nextStart,
					time.Date(now.Year(), now.Month(), now.Day(), end.Hour(),
						end.Minute(), 0, 0, this.timezone), nil
			}
		}
	}

	// Iterate through the days to see when the nearest next run is
	for i := 1; i <= len(this.days); i++ {
		if this.days[(int(now.Weekday())+i)%len(this.days)] {
			// This is the next run
			return time.Date(now.Year(), now.Month(), now.Day(),
					start.Hour(), start.Minute(), 0, 0, this.timezone).AddDate(0, 0, i),
				time.Date(now.Year(), now.Month(), now.Day(),
					end.Hour(), end.Minute(), 0, 0, this.timezone).AddDate(0, 0, i), nil
		}
	}

	// This should never happen since only allow valid schedules are allowed to be configured
	emptyTime := time.Time{}
	return emptyTime, emptyTime, errors.NewAusSchedulingError(emptyTime, emptyTime, nil)
}

func scheduleTask(task taskInfo) errors.Error {
	var err errors.Error
	for retry := 0; retry < _MAX_RETRY; retry++ {
		if err != nil {
			time.Sleep(_SCHEDULING_RETRY_INTERVAL)
		}

		// Create the session name
		session, errS := util.UUIDV4()
		if errS != nil {
			err = errors.NewAusSchedulingError(task.startTime, task.endTime, errS)
			continue
		}

		context, errC := newContext(newAusOutput())
		if errC != nil {
			err = errors.NewAusSchedulingError(task.startTime, task.endTime, errC)
			continue
		}

		task.sessionName = session
		parms := make(map[string]interface{}, 1)
		parms["task"] = task

		after := task.startTime.Sub(time.Now())
		var errT errors.Error

		if task.class == _AUS_TASK_CLASS {
			errT = scheduler.ScheduleStoppableTask(session, _AUS_TASK_CLASS, "", after, execAusTask, stopAusTask, parms, "",
				context)
		} else if task.class == _AUS_CLEANUP_TASK_CLASS {
			errT = scheduler.ScheduleStoppableTask(session, _AUS_CLEANUP_TASK_CLASS, "", after, execCleanupTask, stopCleanupTask,
				parms, "", context)
		}

		if errT != nil {
			err = errors.NewAusSchedulingError(task.startTime, task.endTime, errT)
			continue
		}

		logging.Infof("AUS: New %s task %s scheduled between %v and %v using internal version %s.",
			task.class, task.sessionName, task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT),
			task.version)

		return nil
	}

	logging.Errorf("AUS: Error scheduling %s task: %v", task.class, err)
	return err
}

// Task execution related functions
type taskContext struct {
	sync.Mutex
	task           taskInfo
	abort          bool
	coordKeyPrefix string
	err            errors.Error
	errCount       int
}

func newTaskContext(task taskInfo, coordKeyPrefix string) *taskContext {
	return &taskContext{
		task:           task,
		coordKeyPrefix: coordKeyPrefix,
	}
}

func (this *taskContext) Name() string {
	return this.task.sessionName
}

func (this *taskContext) AddError(err errors.Error) {
	this.Lock()
	this.errCount++
	if this.err == nil {
		this.err = err
	}
	this.Unlock()
}

func (this *taskContext) Stopped() bool {
	return this.abort
}

func (this *taskContext) stop() {
	this.Lock()
	this.abort = true
	this.Unlock()
}

func (this *taskContext) GetExecutionSetup() (query_ee.AusContext, *datastore.ValueConnection, errors.Error) {
	op := newAusOutput()
	ctx, err := newContext(op)
	if err != nil {
		return nil, nil, err
	}

	ausCtx := &ausContext{
		Context:     ctx,
		taskContext: this,
	}

	op.context = ausCtx
	return ausCtx, datastore.NewValueConnection(ausCtx), nil
}

func extractTaskInfo(parms interface{}) (taskInfo, bool) {
	params, ok := parms.(map[string]interface{})
	if !ok {
		return taskInfo{}, false
	}

	taskVal, ok := params["task"]
	if !ok {
		return taskInfo{}, false
	}

	task, ok := taskVal.(taskInfo)
	if !ok {
		return taskInfo{}, false
	}

	return task, true
}

func (this *taskContext) processCollections(skipDisabled bool, getKsSettings bool,
	keyspaceHandler func(datastore.Keyspace, *keyspaceSettings, *taskContext)) (
	bucketsDetected bool) {

	ausCfg.RLock()
	allBuckets := ausCfg.settings.allBuckets
	globalChangePercentage := ausCfg.settings.changePercentage
	ausCfg.RUnlock()

	buckets, err := getBuckets(allBuckets)
	if err != nil {
		this.AddError(err)
		return
	}

	if len(buckets) == 0 {
		bucketsDetected = false
		return
	}
	bucketsDetected = true

	rand := rand.New(rand.NewSource(time.Now().Unix()))
	bStart := rand.Intn(len(buckets))
	dpairs := value.Pairs{value.Pair{
		Name:    "",
		Value:   value.EMPTY_ANNOTATED_OBJECT,
		Options: value.NewValue(map[string]interface{}{"expiration": this.task.endTime.Unix()}),
	}}

	for i := 0; i < len(buckets); i++ {

		if this.abort {
			return
		}

		// Randomized start to prevent starvation of a keyspace
		bucket := buckets[(bStart+i)%len(buckets)]
		scopeIds, err := bucket.ScopeIds()
		if err != nil {
			this.AddError(err)
			continue
		}

		if len(scopeIds) == 0 {
			continue
		}

		systemCollection, err := getSystemCollection(bucket.Name())
		if err != nil {
			this.AddError(err)
			continue
		}

		var bucketSettingRetreived bool
		var bucketSetting *keyspaceSettings

		sStart := rand.Intn(len(scopeIds))
		for j := 0; j < len(scopeIds); j++ {
			if this.abort {
				return
			}

			scope, err := bucket.ScopeById(scopeIds[(sStart+j)%len(scopeIds)])
			if err != nil {
				this.AddError(err)
				continue
			}

			collectionIds, err := scope.KeyspaceIds()
			if err != nil {
				this.AddError(err)
				continue
			}

			if len(collectionIds) == 0 {
				continue
			}

			var scopeSettingRetreived bool
			var scopeSetting *keyspaceSettings

			cStart := rand.Intn(len(collectionIds))
			for k := 0; k < len(collectionIds); k++ {
				if this.abort {
					return
				}

				coll, err := scope.KeyspaceById(collectionIds[(cStart+k)%len(collectionIds)])
				if err != nil {
					this.AddError(err)
					continue
				}

				var sb strings.Builder
				sb.WriteString(this.coordKeyPrefix)
				sb.WriteString(coll.QualifiedName())

				dpairs[0].Name = sb.String()

				// Insert the coordination document to prevent more than one node performing the task on the same keyspace
				_, _, iErrs := systemCollection.Insert(dpairs, datastore.NULL_QUERY_CONTEXT, false)
				if len(iErrs) > 0 {
					if !iErrs[0].HasCause(errors.E_DUPLICATE_KEY) {
						this.AddError(errors.NewAusTaskError("Error during coordination.", iErrs[0]))
						logging.Errorf(fmt.Sprintf("AUS: Error inserting the coordination document for keyspace %s: %v",
							coll.QualifiedName(),
							iErrs[0]))
					}
					continue
				}

				settings := &defaultKeyspaceSettings
				if skipDisabled || getKsSettings {
					if !bucketSettingRetreived {
						val, errsF := fetchSystemCollection(bucket.Name(), _AUS_SETTINGS_BUCKET_DOC_KEY)
						if len(errsF) > 0 {
							logging.Errorf(fmt.Sprintf("AUS: Error fetching settings document for bucket %v: %v", bucket.Name(), errsF))
							this.AddError(errsF[0])
							continue
						} else if val != nil {
							setting, err := validateAusSettingDoc(val)
							if err != nil {
								logging.Errorf(fmt.Sprintf("AUS: Invalid settings document for bucket %v: %v", bucket.Name(), err))
								this.AddError(err)
								continue
							}
							bucketSetting = setting
						}
						bucketSettingRetreived = true
					}

					if skipDisabled && bucketSetting != nil && bucketSetting.enable == value.FALSE {
						continue
					}

					if !scopeSettingRetreived {
						val, errsS := fetchSystemCollection(bucket.Name(),
							ausSettingsScopeKey(scope.Uid(), scope.Name()))
						if len(errsS) > 0 {
							logging.Errorf(fmt.Sprintf("AUS: Error fetching settings document for scope %v: %v",
								algebra.PathFromParts("default", bucket.Name(), scope.Name()), errsS))
							this.AddError(errsS[0])
							continue
						} else if val != nil {
							setting, err := validateAusSettingDoc(val)
							if err != nil {
								logging.Errorf(fmt.Sprintf("AUS: Invalid settings document for scope %v: %v",
									algebra.PathFromParts("default", bucket.Name(), scope.Name()), err))
								this.AddError(err)
								continue
							}
							scopeSetting = setting
						}
						scopeSettingRetreived = true
					}

					if skipDisabled && scopeSetting != nil && scopeSetting.enable == value.FALSE {
						continue
					}

					var collSetting *keyspaceSettings
					val, errsC := fetchSystemCollection(bucket.Name(),
						ausSettingsCollectionKey(scope.Uid(), scope.Name(), coll.Uid(), coll.Name()))
					if len(errsC) > 0 {
						logging.Errorf(fmt.Sprintf("AUS: Error fetching settings document for collection %v: %v", coll.QualifiedName(),
							errsC))
						this.AddError(errsC[0])
						continue
					} else if val != nil {
						setting, err := validateAusSettingDoc(val)
						if err != nil {
							logging.Errorf(fmt.Sprintf("AUS: Invalid settings document for collection %v: %v", coll.QualifiedName(),
								err))
							this.AddError(err)
							continue
						}
						collSetting = setting
					}

					if skipDisabled && collSetting != nil && collSetting.enable == value.FALSE {
						continue
					}

					if getKsSettings {
						// Get the "change percentage" at which to evaluate this keyspace
						change := globalChangePercentage
						if collSetting != nil && collSetting.changePercentage >= 0 {
							change = collSetting.changePercentage
						} else if scopeSetting != nil && scopeSetting.changePercentage >= 0 {
							change = scopeSetting.changePercentage
						} else if bucketSetting != nil && bucketSetting.changePercentage >= 0 {
							change = bucketSetting.changePercentage
						}

						timeout := -1
						// Get the update_statistics_timeout from the settings
						if collSetting != nil && collSetting.update_stats_timeout >= 0 {
							timeout = collSetting.update_stats_timeout
						} else if scopeSetting != nil && scopeSetting.update_stats_timeout >= 0 {
							timeout = scopeSetting.update_stats_timeout
						} else if bucketSetting != nil && bucketSetting.update_stats_timeout >= 0 {
							timeout = bucketSetting.update_stats_timeout
						}

						settings.changePercentage = change
						settings.update_stats_timeout = timeout
					}
				}

				keyspaceHandler(coll, settings, this)
			}
		}

	}
	return
}

func getBuckets(allBuckets bool) ([]datastore.ExtendedBucket, errors.Error) {
	ds, ok := datastore.GetDatastore().(datastore.Datastore2)
	if !ok || ds == nil {
		// This should not ideally happen as migration should have completed.
		return nil, errors.NewAusNotInitialized()
	}

	var buckets []datastore.ExtendedBucket

	if allBuckets {
		ds.LoadAllBuckets(func(b datastore.ExtendedBucket) {
			buckets = append(buckets, b)
		})
	} else {
		ds.ForeachBucket(func(b datastore.ExtendedBucket) {
			buckets = append(buckets, b)
		})
	}

	return buckets, nil
}

// Returns whether the task should be started or not
func fuzzyStart() bool {
	for retry := 0; retry < _MAX_RETRY; retry++ {
		if ausCfg.server.LoadFactor() <= _MAX_LOAD_FACTOR {
			break
		} else if retry == _MAX_RETRY-1 {
			logging.Errorf("AUS: Task not started due to existing load on the node.")
			return false
		} else {
			time.Sleep(_RETRY_INTERVAL)
		}
	}

	return true
}

// AUS task related functions

func execAusTask(context scheduler.Context, parms interface{}, stopChannel <-chan bool) (interface{}, []errors.Error) {
	task, ok := extractTaskInfo(parms)
	if !ok {
		return nil, []errors.Error{errors.NewAusTaskInvalidInfoError(parms)}
	}

	taskCtx := newTaskContext(task, _AUS_COORDINATION_DOC_PREFIX+task.version+"::")

	var timedOut bool
	timer := time.NewTimer(task.endTime.Sub(task.startTime))

	go func() {
		var stop bool
		select {
		case stop = <-stopChannel:
			if stop {
				logging.Warnf("AUS: [%s] Task stopped due to user cancellation.", task.sessionName)
			}
		case <-timer.C:
			timedOut = true
			logging.Errorf("AUS: [%s] Task stopped as timeout exceeded.", task.sessionName)
		}

		if timedOut || stop {
			timer.Stop()
			taskCtx.stop()
		}
	}()

	defer func() {
		timer.Stop()
		timer = nil

		r := recover()
		if r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severef("AUS: [%s] Panic in task execution: %v\n%v", task.sessionName, r, s)
		}

		ausCfg.scheduleAusTask(task.version, false)
		ausCfg.setRunningAusEnd(true, time.Time{})
	}()

	logging.Infof(
		"AUS: Task %s scheduled between %v and %v started.", task.sessionName,
		task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT))

	ausCfg.Lock()
	if task.version != ausCfg.configVersion {
		ausCfg.Unlock()
		s := fmt.Sprintf("Not executed. Scheduled with outdated config version %s.", task.version)
		logging.Infof("AUS: [%s] %s", task.sessionName, s)
		return s, nil
	}
	ausCfg.setRunningAusEnd(false, task.endTime)
	allBuckets := ausCfg.settings.allBuckets
	globalChangePercentage := ausCfg.settings.changePercentage
	createMissing := ausCfg.settings.createMissing
	ausCfg.Unlock()

	logging.Infof(
		"AUS: [%s] Configurations: start_time: %v end_time: %v change_percentage: %v all_buckets: %v internal_version: %v",
		task.sessionName, task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT),
		globalChangePercentage, allBuckets, task.version)

	start := fuzzyStart()
	if !start {
		logging.Errorf("AUS: [%s] Task not started due to existing load on the node.", task.sessionName)
		return nil, errors.Errors{errors.NewAusTaskNotStartedError()}
	}

	var keyspacesUpdated []interface{}

	ausHandler := func(keyspace datastore.Keyspace, settings *keyspaceSettings, taskContext *taskContext) {

		if taskContext.abort {
			return
		}

		changePerc := globalChangePercentage
		if settings != nil && settings.changePercentage >= 0 {
			changePerc = settings.changePercentage
		}

		statupdater, err := ausCfg.server.Datastore().StatUpdater()
		if err != nil {
			taskContext.AddError(err)
			return
		} else if statupdater.Name() != datastore.UPDSTAT_DEFAULT {
			taskContext.AddError(errors.NewAusNotSupportedError())
			return
		}

		updated := aus.AutoUpdateStatistics(keyspace, changePerc, settings.update_stats_timeout, createMissing,
			statupdater, taskContext)
		if updated {
			keyspacesUpdated = append(keyspacesUpdated, keyspace.QualifiedName())
		}

		return
	}

	bucketsDetected := taskCtx.processCollections(true, true, ausHandler)

	logging.Infof("AUS: [%s] Keyspaces qualified for update: %v", task.sessionName, keyspacesUpdated)

	var errs errors.Errors
	if taskCtx.err != nil {
		errs = append(errs, taskCtx.err)
	}

	errCount := taskCtx.errCount
	if timedOut {
		errCount++
		errs = append(errs, errors.NewAusTaskTimeoutExceeded())
	}

	taskRv := make(map[string]interface{}, 2)
	if errCount > 0 {
		taskRv["error_count"] = errCount
	}

	if !bucketsDetected {
		var s string
		if allBuckets {
			s = "No buckets detected in cluster."
		} else {
			s = "No buckets loaded on the node."
		}
		logging.Infof("AUS: [%s] %s", task.sessionName, s)
		taskRv["info"] = s
	}

	taskRv["internal_version"] = task.version

	if len(keyspacesUpdated) > 0 {
		taskRv["keyspaces_updated"] = keyspacesUpdated
	}

	logging.Infof("AUS: [%s] Task scheduled between %v and %v has completed with %v errors.",
		task.sessionName, task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT), taskCtx.errCount)

	return taskRv, errs
}

func stopAusTask(context scheduler.Context, parms interface{}) (interface{}, []errors.Error) {

	task, ok := extractTaskInfo(parms)
	if !ok {
		return nil, []errors.Error{errors.NewAusTaskInvalidInfoError(parms)}
	}

	rv := fmt.Sprintf("Task scheduled between %v and %v has been cancelled.", task.startTime.Format(util.DEFAULT_FORMAT),
		task.endTime.Format(util.DEFAULT_FORMAT))

	versionChange, err := ausCfg.scheduleAusTask(task.version, false)
	if versionChange {
		rv += fmt.Sprintf(" Scheduled with outdated config version %s.", task.version)
	}

	logging.Infof("AUS: [%s] %s", task.sessionName, rv)

	if err != nil {
		return fmt.Sprintf("Not executed. Scheduled with outdated config version %s.", task.version), []errors.Error{err}
	}

	return rv, nil
}

// Cleanup task functions
// When AUS is changed from enabled to disabled, the cleanup tasks deletes all AUS related documents stored in KV

func (this *ausConfig) scheduleCleanupTask(version string) (bool, errors.Error) {
	this.RLock()
	cfgVersion := this.configVersion
	runningAusEnd := this.runningAusEnd
	runningCleanupEnd := this.runningCleanupEnd
	this.RUnlock()

	if version != cfgVersion {
		logging.Infof("AUS: Detected global settings change. Cleanup task configured with internal version %v not scheduled.",
			version)

		return true, nil
	}

	startTime := time.Now().Add(time.Second)

	if !runningAusEnd.IsZero() {
		// implies there is an AUS task running
		// Do not schedule a cleanup task right now. Schedule it after the task ends
		startTime = runningAusEnd.Add(time.Second * 30)
	}

	if !runningCleanupEnd.IsZero() {
		if startTime.Before(runningCleanupEnd) {
			startTime = runningCleanupEnd.Add(time.Second * 30)
		}
	}

	task := taskInfo{startTime: startTime, endTime: startTime.Add(time.Hour), version: version,
		class: _AUS_CLEANUP_TASK_CLASS}

	err := scheduleTask(task)
	if err != nil {
		return false, err
	}

	return true, nil
}

func execCleanupTask(context scheduler.Context, parms interface{}, stopChannel <-chan bool) (interface{}, []errors.Error) {
	task, ok := extractTaskInfo(parms)
	if !ok {
		return nil, []errors.Error{errors.NewAusTaskInvalidInfoError(parms)}
	}

	logging.Infof("AUS: [%s] Cleanup task scheduled between %v and %v started.",
		task.sessionName, task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT))
	logging.Infof("AUS: [%s] Configurations of cleanup task: start_time: %v end_time: %v internal version: %v",
		task.sessionName, task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT), task.version)

	taskCtx := newTaskContext(task, _AUS_CLEANUP_COORDINATION_DOC_PREFIX+task.version+"::")

	var timedOut bool
	timer := time.NewTimer(task.endTime.Sub(task.startTime))

	go func() {
		select {
		case stop := <-stopChannel:
			if stop {
				timer.Stop()
				taskCtx.stop()
				logging.Errorf("AUS: [%s] Cleanup task stopped due to user cancellation.", task.sessionName)
			}
		case <-timer.C:
			timer.Stop()
			taskCtx.stop()
			timedOut = true
			logging.Errorf("AUS: [%s] Cleanup task stopped as timeout exceeded.", task.sessionName)
		}
	}()

	defer func() {
		r := recover()
		if r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severef("AUS: [%s] Panic in cleanup execution: %v\n%v", task.sessionName, r, s)
		}

		timer.Stop()
		timer = nil
		ausCfg.setRunningCleanupEnd(true, time.Time{})
	}()

	ausCfg.Lock()
	if ausCfg.configVersion != task.version {
		ausCfg.Unlock()
		s := fmt.Sprintf(
			"Not executed. Scheduled with outdated config version %s.", task.version)
		logging.Infof("AUS: [%s] Cleanup task %s", task.sessionName, s)
		return s, nil
	}
	ausCfg.setRunningCleanupEnd(false, task.endTime)
	ausCfg.Unlock()

	var keyspacesCleaned []interface{}

	cleanupHandler := func(keyspace datastore.Keyspace, settings *keyspaceSettings, taskContext *taskContext) {
		if taskContext.abort {
			return
		}

		err := dictionary.DeleteAllAusHistoryDocs(keyspace)
		if err != nil {
			logging.Errorf("AUS: [%s] Error performing cleanup for keyspace %s: %v", task.sessionName, keyspace.QualifiedName(), err)
			taskContext.AddError(err)
			return
		}

		keyspacesCleaned = append(keyspacesCleaned, keyspace.QualifiedName())
		return
	}

	bucketsDetected := taskCtx.processCollections(false, false, cleanupHandler)

	logging.Infof("AUS: [%s] Cleaned up done on keyspaces: %v", task.sessionName, keyspacesCleaned)

	var errs errors.Errors
	if taskCtx.err != nil {
		errs = append(errs, taskCtx.err)
	}

	errCount := taskCtx.errCount
	if timedOut {
		errCount++
		errs = append(errs, errors.NewAusTaskTimeoutExceeded())
	}

	taskRv := make(map[string]interface{}, 2)
	if errCount > 0 {
		taskRv["error_count"] = errCount
	}

	if !bucketsDetected {
		s := "No buckets detected in cluster."
		logging.Infof("AUS: [%s] Cleanup task completed with %s", task.sessionName, s)
		taskRv["info"] = s
	}

	taskRv["internal_version"] = task.version

	if len(keyspacesCleaned) > 0 {
		taskRv["keyspaces_cleaned"] = keyspacesCleaned
	}

	logging.Infof("AUS: [%s] Cleanup task scheduled between %v and %v has completed with %v errors.",
		task.sessionName, task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT), taskCtx.errCount)

	return taskRv, errs
}

func stopCleanupTask(context scheduler.Context, parms interface{}) (interface{}, []errors.Error) {
	task, ok := extractTaskInfo(parms)
	if !ok {
		return nil, []errors.Error{errors.NewAusTaskInvalidInfoError(parms)}
	}

	rv := fmt.Sprintf("Cleanup task scheduled between %v and %v has been cancelled.", task.startTime.Format(util.DEFAULT_FORMAT),
		task.endTime.Format(util.DEFAULT_FORMAT))

	ausCfg.RLock()
	version := ausCfg.settings.version
	ausCfg.RUnlock()

	if version != task.version {
		rv += fmt.Sprintf(" Scheduled with outdated config version %s.", task.version)
	}

	logging.Infof("AUS: [%s] %s", task.sessionName, rv)

	return fmt.Sprintf("Not executed. Scheduled with outdated config version %s.", task.version), nil
}

// Execution related functions

func newContext(output *ausOutput) (*execution.Context, errors.Error) {

	// Get admin credentials for this node
	creds, err := getNodeAdminCreds()
	if err != nil {
		return nil, err
	}

	qServer := ausCfg.server
	ctx := execution.NewContext("aus_request", qServer.Datastore(), qServer.Systemstore(), "default", false,
		qServer.MaxParallelism(), qServer.ScanCap(), qServer.PipelineCap(), qServer.PipelineBatch(), nil, nil,
		creds, datastore.NOT_SET, &ZeroScanVectorSource{}, output, nil, qServer.MaxIndexAPI(),
		util.GetN1qlFeatureControl(), "", util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_FLEXINDEX),
		util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_CBO), server.GetNewOptimizer(), datastore.DEF_KVTIMEOUT,
		0, logging.NONE)

	memQuota := qServer.MemoryQuota()
	if memQuota > 0 {
		ctx.SetMemoryQuota(memQuota)
	}

	if memQuota > 0 || memory.Quota() > 0 {
		ctx.SetMemorySession(memory.Register())
	}

	ctx.SetAUS()

	return ctx, nil
}

func getNodeAdminCreds() (*auth.Credentials, errors.Error) {

	if distributed.RemoteAccess().StandAlone() {
		return nil, nil
	}

	node, err := getNodeName()
	if err != nil {
		return nil, err
	}

	creds, err1 := datastore.AdminCreds(node)
	if err1 != nil {
		return nil, errors.NewNoAdminPrivilegeError(err)
	}

	return creds, err
}

func getNodeName() (string, errors.Error) {
	node := distributed.RemoteAccess().WhoAmI()
	if node == "" {
		// This can only happen for nodes running outside of the cluster
		return "", errors.NewNoAdminPrivilegeError(fmt.Errorf("cannot establish node name"))
	}

	return node, nil
}

// scanVectorEntries implements timestamp.Vector
type scanVectorEntries struct {
	entries []timestamp.Entry
}

func (this *scanVectorEntries) Entries() []timestamp.Entry {
	return this.entries
}

// implements timestamp.ScanVectorSource interface
type ZeroScanVectorSource struct {
	empty scanVectorEntries
}

func (this *ZeroScanVectorSource) ScanVector(namespace_id string, keyspace_name string) timestamp.Vector {
	// Always return a vector of 0 entries.
	return &this.empty
}

func (this *ZeroScanVectorSource) Type() int32 {
	return timestamp.NO_VECTORS
}

type ausContext struct {
	*execution.Context
	taskContext *taskContext
	output      *ausOutput
	stopped     bool
}

func (this *ausContext) IsActive() bool {
	return !this.stopped && !this.taskContext.Stopped()
}

func (this *ausContext) stop() {
	this.stopped = true
}

// context.Output implementation for executions in AUS related operations
type ausOutput struct {
	err     errors.Error
	context *ausContext
}

func newAusOutput() *ausOutput {
	return &ausOutput{}
}

func (this *ausOutput) SetUp() {
}

func (this *ausOutput) Result(item value.AnnotatedValue) bool {
	return (this.err == nil)
}

func (this *ausOutput) CloseResults() {
}

func (this *ausOutput) Abort(err errors.Error) {
	if this.err == nil {
		this.err = err

		if this.context != nil {
			this.context.stop()
		}
	}
}

func (this *ausOutput) Fatal(err errors.Error) {
	if this.err == nil {
		this.err = err

		if this.context != nil {
			this.context.stop()
		}
	}
}

func (this *ausOutput) Error(err errors.Error) {
	if this.err == nil {
		this.err = err

		if this.context != nil {
			this.context.stop()
		}
	}
}

func (this *ausOutput) SetErrors(errs errors.Errors) {
	for _, err := range errs {
		this.Error(err)
	}
}

func (this *ausOutput) Warning(wrn errors.Error) {
}

func (this *ausOutput) Errors() []errors.Error {
	if this.err == nil {
		return nil
	}
	return []errors.Error{this.err}
}

func (this *ausOutput) AddMutationCount(i uint64) {
	// do nothing
}

func (this *ausOutput) MutationCount() uint64 {
	return 0
}

func (this *ausOutput) SetSortCount(i uint64) {
	// do nothing
}

func (this *ausOutput) SortCount() uint64 {
	return 0
}

func (this *ausOutput) AddPhaseCount(p execution.Phases, c uint64) {
	// do nothing
}

func (this *ausOutput) AddPhaseOperator(p execution.Phases) {
	// do nothing
}

func (this *ausOutput) PhaseOperator(p execution.Phases) uint64 {
	return uint64(0)
}

func (this *ausOutput) FmtPhaseCounts() map[string]interface{} {
	// do nothing
	return nil
}

func (this *ausOutput) FmtPhaseOperators() map[string]interface{} {
	// do nothing
	return nil
}

func (this *ausOutput) AddPhaseTime(phase execution.Phases, duration time.Duration) {
	// do nothing
}

func (this *ausOutput) FmtPhaseTimes(s util.DurationStyle) map[string]interface{} {
	// do nothing
	return nil
}

func (this *ausOutput) RawPhaseTimes() map[string]interface{} {
	return nil
}

func (this *ausOutput) FmtOptimizerEstimates(op execution.Operator) map[string]interface{} {
	// do nothing
	return nil
}

func (this *ausOutput) TrackMemory(size uint64) {
	// do nothing
}

func (this *ausOutput) SetTransactionStartTime(t time.Time) {
	// do nothing
}

func (this *ausOutput) AddTenantUnits(s tenant.Service, ru tenant.Unit) {
	// do nothing
}

func (this *ausOutput) AddCpuTime(d time.Duration) {
	// do nothing
}

func (this *ausOutput) AddIoTime(d time.Duration) {
	// do nothing
}

func (this *ausOutput) AddWaitTime(d time.Duration) {
	// do nothing
}

func (this *ausOutput) Loga(l logging.Level, f func() string) {
	// do nothing
}

func (this *ausOutput) LogLevel() logging.Level {
	return logging.INFO
}

func (this *ausOutput) GetErrorLimit() int {
	return 1
}

func (this *ausOutput) GetErrorCount() int {
	if this.err == nil {
		return 0
	}

	return 1
}
