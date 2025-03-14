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
	"sync/atomic"
	"time"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query-ee/aus"
	"github.com/couchbase/query-ee/aus/bridge"
	"github.com/couchbase/query-ee/optimizer"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/expression"
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
	_AUS_SETTINGS_BUCKET_DOC_KEY = AUS_DOC_PREFIX + "bucket"

	_MAX_RETRY                 = 10
	_RETRY_INTERVAL            = 30 * time.Second
	_SCHEDULING_RETRY_INTERVAL = 5 * time.Second
	_MAX_LOAD_FACTOR           = 65
)

var ausCfg ausConfig

// since we fetch documents from _system._query 1 at a time
var _STRING_ANNOTATED_POOL *value.StringAnnotatedPool

var defaultKeyspaceSettings = keyspaceSettings{enable: value.NONE, changePercentage: -1}

type ausConfig struct {
	sync.RWMutex
	server      *server.Server
	settings    ausGlobalSettings
	initialized bool

	// Indicates the version of the global settings that was used to schedule/ enable the current task.
	// This field is only changes when the new global settings received, change the enablement or schedule.
	configVersion string

	// Stores information about the running task's window
	// Is set when the AUS task begins and re-set to zero when the task completes.
	// When there is a new global setting received that changes the schedule, the scheduling mechanism uses this field to ensure
	// that the new task is not scheduled to run in the same window as the current running task.
	// This is to prevent multiple tasks from running at once on the node.
	currentWindow taskInfo
}

type ausGlobalSettings struct {
	enable           bool
	allBuckets       bool
	changePercentage int
	schedule         ausSchedule
	version          string
}

type ausSchedule struct {
	startTime time.Time
	endTime   time.Time
	timezone  *time.Location
	days      []bool
}

type taskInfo struct {
	startTime time.Time
	endTime   time.Time

	// The configVersion that was used to schedule this task.
	// This field should be used to perform an equality check against the current global configVersion before the task is
	// executed or [re]scheduled. If the check fails, it means that there were new settings received that changed the config and
	// the task operation must not proceed.
	version string
}

func (this *ausConfig) setInitialized(init bool) {
	this.Lock()
	this.initialized = init
	this.Unlock()
}

func (this *ausConfig) setCurrentWindow(lock bool, start time.Time, end time.Time, version string) {
	if lock {
		this.Lock()
		defer this.Unlock()
	}

	this.currentWindow.startTime = start
	this.currentWindow.endTime = end
	this.currentWindow.version = version
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

// Sets up AUS for the node. Must only be called post-migration of CBO stats.
func startupAus() {

	if ausCfg.initialized || tenant.IsServerless() {
		return
	}

	// If migration has completed - either the state of migration is MIGRATED or ABORTED - allow AUS to be initialized.
	// Otherwise wait for migration to complete
	if m, _ := migration.IsComplete("CBO_STATS"); !m {
		logging.Infof(AUS_LOG_PREFIX + "Initialization will start when migration to a supported version is complete.")
		migration.Await("CBO_STATS")
	}

	logging.Infof(AUS_LOG_PREFIX + "Initialization started.")

	// Initialize the system:aus_settings Fetch pool
	_STRING_ANNOTATED_POOL = value.NewStringAnnotatedPool(1)

	// Initialize the global settings in metakv
	err := initMetakv()
	if err != nil {
		logging.Errorf(AUS_LOG_PREFIX+"Metakv initialization failed with error: %v", err)
	}

	go metakv.RunObserveChildren(_AUS_PATH, callback, make(chan struct{}))
	ausCfg.setInitialized(true)

	logging.Infof(AUS_LOG_PREFIX + "Initialization completed.")
}

func callback(kve metakv.KVEntry) error {
	ausCfg.Lock()
	logging.Infof(AUS_LOG_PREFIX + "New global settings received.")

	var v map[string]interface{}
	err := json.Unmarshal(kve.Value, &v)
	if err != nil {
		ausCfg.Unlock()
		logging.Errorf(AUS_LOG_PREFIX+"Error unmarshalling new global settings: %v", err)
		return nil
	}

	if newSettings, err, _ := setAusHelper(v, false, false); err != nil {
		ausCfg.Unlock()
		logging.Errorf(AUS_LOG_PREFIX+"Error during global settings retrieval: %v", err)
		return nil
	} else {

		prev := ausCfg.settings
		ausCfg.settings = newSettings

		var version string
		var reschedule bool // whether a new AUS task should be scheduled
		var cancelOld bool  // whether any scheduled AUS tasks should be deleted

		if prev.enable {
			if !newSettings.enable {
				// If AUS enablement changes from enabled to disabled
				cancelOld = true
			} else if !prev.schedule.equals(&newSettings.schedule) {
				// If AUS remains enabled but the schedule has changed
				reschedule = true
				cancelOld = true
			}
		} else if !prev.enable && newSettings.enable {
			// If AUS enablement changes from disabled to enabled
			reschedule = true
		}

		// Only if there is a change in the enablement/schedule change the version of the configuration
		if reschedule || cancelOld {
			ausCfg.configVersion = newSettings.version
		}

		version = ausCfg.configVersion
		ausCfg.Unlock()

		logging.Infof(AUS_LOG_PREFIX+"Global settings are now: %v with internal version: %v.", v, version)

		if cancelOld {
			// Delete all scheduled AUS tasks. But let running tasks complete execution.
			if err == nil {

				var taskId string
				scheduler.TasksForeach(func(name string, task *scheduler.TaskEntry) bool {
					taskId = ""
					if task.Class == "auto_update_statistics" && task.State == scheduler.SCHEDULED {
						taskId = task.Id
					}
					return true

				}, func() bool {
					if taskId != "" {
						execErr := scheduler.DeleteTask(taskId)
						if execErr != nil {
							logging.Errorf(AUS_LOG_PREFIX+"Error deleting old scheduled task with id %v: %v", taskId, execErr)
						}
					}
					return true
				}, scheduler.SCHEDULED_TASKS_CACHE)
			}
			if err != nil {
				logging.Errorf(AUS_LOG_PREFIX+"Error during global settings retrieval: %v", err)
			}
		}

		// Schedule next task with the new schedule
		if reschedule {
			ausCfg.schedule(version, true)
		}
	}

	return nil
}

// Initialize the global settings in metakv
func initMetakv() error {
	_, _, err := fetchAusHelper(false)
	return err
}

func initDaysOfWeek() map[string]time.Weekday {
	days := make(map[string]time.Weekday, 7)
	days["sunday"] = time.Sunday
	days["monday"] = time.Monday
	days["tuesday"] = time.Tuesday
	days["wednesday"] = time.Wednesday
	days["thursday"] = time.Thursday
	days["friday"] = time.Friday
	days["saturday"] = time.Saturday
	return days
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
		logging.Errorf(AUS_LOG_PREFIX+"Error retrieving global settings from metakv: %v", err)
		return nil, nil, err
	}

	// Only add the default global settings document in metakv if the path is non-existent
	if rv == nil && rev == nil && err == nil {

		// The default global settings document
		bytes, err := json.Marshal(map[string]interface{}{
			"enable":            false,
			"all_buckets":       false,
			"change_percentage": 10,
			"internal_version":  "default",
		})

		if err != nil {
			logging.Errorf(AUS_LOG_PREFIX+"Error encoding default global settings document: %v", err)
			return nil, nil, err
		}

		err = metakv.Add(_AUS_GLOBAL_SETTINGS_PATH, bytes)

		// If a non 200 OK status is returned from ns_server
		if err != nil && err != metakv.ErrRevMismatch {
			logging.Errorf(AUS_LOG_PREFIX+"Error distributing default global settings via metakv: %v", err)
			return nil, nil, err
		}

		if get {
			// Attempt to get the latest value for the key
			rv, rev, err = metakv.Get(_AUS_GLOBAL_SETTINGS_PATH)
			if err != nil {
				logging.Errorf(AUS_LOG_PREFIX+"Error retrieving global settings from metakv: %v", err)
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
				if warnings != nil {
					warnings = append(warnings, warn...)
				}
				if err != nil {
					return obj, err, nil
				}
				obj.schedule = sched
			case "internal_version":
				if vv, ok := v.(string); ok {
					obj.version = vv
				} else {
					logging.Warnf(AUS_LOG_PREFIX+"internal_version is not a string: %v", v)
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

	return datastore.ScanSystemCollection(bucket, AUS_DOC_PREFIX, nil,
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
	dpairs := make([]value.Pair, 1)
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
// aus::bucket
// 2. Scope level document:
// aus::scope_id::scope_name
// 3. Collection level document:
// aus::scope_id::collection_id::scope_name.collection_name
func key2path(key, namespace, bucket string) (string, []string) {
	// strip prefix and scope (and collection) UIDs from the scope (and collection) names
	parts := strings.Split(key, "::")

	// Check if the key is in right bucket/ scope/ collection level document key format
	if len(parts) < 2 || len(parts) > 4 {
		return "", nil
	}

	// check if the key is prefixed by "aus::"
	if parts[0] != "aus" {
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
	sb.WriteString(AUS_DOC_PREFIX)
	sb.WriteString(scopeUid)
	sb.WriteString("::")
	sb.WriteString(scopeName)
	return sb.String()
}

// Returns document id of a collection level document in system:aus_settings
func ausSettingsCollectionKey(scopeUid string, scopeName string, collectionUid string, collectionName string) string {
	sb := strings.Builder{}
	sb.WriteString(AUS_DOC_PREFIX)
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
func validateAusSettingDoc(doc value.Value) (keyspaceSettings, errors.Error) {

	settings := defaultKeyspaceSettings

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
				return defaultKeyspaceSettings, errors.NewAusDocInvalidSettingsValue(k, v)
			} else {
				settings.enable = value.ToTristate(e)
			}
		case "change_percentage":
			if cp, ok := v.(int64); !ok {
				return defaultKeyspaceSettings, errors.NewAusDocInvalidSettingsValue(k, v)
			} else if cp < 0 || cp > 100 {
				return defaultKeyspaceSettings, errors.NewAusDocInvalidSettingsValue(k, v)
			} else {
				settings.changePercentage = int(cp)
			}
		default:
			return defaultKeyspaceSettings, errors.NewAusDocUnknownSetting(k)
		}
	}

	return settings, nil
}

// Represents the keyspace i.e bucket/scope/collection level setting stored in system:aus_settings
type keyspaceSettings struct {
	enable           value.Tristate
	changePercentage int
}

// Cleans up scope level AUS documents
func DropScope(namespace string, bucket string, scope string, scopeUid string) {
	if !ausCfg.initialized {
		return
	}

	datastore.ScanSystemCollection(bucket, AUS_DOC_PREFIX+scopeUid+"::", nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			dpairs := make([]value.Pair, 1)
			dpairs[0].Name = key
			queryContext := datastore.GetDurableQueryContextFor(systemCollection)

			_, _, mErrs := systemCollection.Delete(dpairs, queryContext, false)
			if len(mErrs) > 0 {
				logging.Errorf(AUS_LOG_PREFIX+
					"Errors during cleanup of settings document for the scope %s Uid %s. Error: %v",
					algebra.PathFromParts(namespace, bucket, scope), scopeUid, mErrs[0])
			}
			return nil
		}, nil)
}

// Cleans up collection level AUS documents
func DropCollection(namespace string, bucket string, scope string, scopeUid string, collection string, collectionUid string) {
	if !ausCfg.initialized {
		return
	}

	datastore.ScanSystemCollection(bucket, AUS_DOC_PREFIX+scopeUid+"::"+collectionUid+"::", nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			dpairs := make([]value.Pair, 1)
			dpairs[0].Name = key
			queryContext := datastore.GetDurableQueryContextFor(systemCollection)

			_, _, mErrs := systemCollection.Delete(dpairs, queryContext, false)
			if len(mErrs) > 0 {
				logging.Errorf(AUS_LOG_PREFIX+
					"Errors during cleanup of settings document for the collection '%s' with Uid '%s'. Error: %v",
					algebra.PathFromParts(namespace, bucket, scope, collection), collectionUid, mErrs[0])
			}
			return nil
		}, nil)
}

// Backup related functions

func BackupAusSettings(namespace string, bucket string, filter func([]string) bool) ([]interface{}, errors.Error) {
	rv := make([]interface{}, 0)
	keys := make([]string, 1)

	err := datastore.ScanSystemCollection(bucket, AUS_DOC_PREFIX, nil,
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
func (this *ausConfig) schedule(version string, checkToday bool) (bool, errors.Error) {

	this.RLock()
	defer this.RUnlock()

	// Skip scheduling if the task's version differs from current config version,
	// It indicates that newer settings have been received.
	if version != this.configVersion {
		logging.Infof(AUS_LOG_PREFIX+
			"Detected global settings change. Task configured with internal version %v not scheduled.", version)
		return true, nil
	}

	// Check for overlap between the next scheduled window and the currently running task's window
	var windowStart time.Time
	var windowEnd time.Time
	if checkToday {
		windowStart = this.currentWindow.startTime
		windowEnd = this.currentWindow.endTime
	}

	start, end, err := this.settings.schedule.findNextWindow(checkToday, windowStart, windowEnd)
	if err != nil {
		return false, err
	}

	task := taskInfo{startTime: start, endTime: end, version: version}
	err = scheduleTask(task)
	if err != nil {
		logging.Errorf(AUS_LOG_PREFIX+"Error scheduling task: %v", err)
		return false, err
	}

	logging.Infof(AUS_LOG_PREFIX+"New task scheduled between %v and %v using configuration version %s.",
		start.Format(util.DEFAULT_FORMAT), end.Format(util.DEFAULT_FORMAT), this.configVersion)

	return false, nil
}

// Returns the start and end times of the next window for the next AUS task
// Parameters:
// checkToday: If true, considers today's schedule when finding the next window and
// performs overlap check with the provided start and end times specified by the parameters windowStart and windowEnd
func (this *ausSchedule) findNextWindow(checkToday bool, windowStart time.Time, windowEnd time.Time) (
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
			if !windowStart.IsZero() && !windowEnd.IsZero() {
				// If the next run is during the provided window
				if !nextStart.Before(windowStart) && !nextStart.After(windowEnd) {
					today = false
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
	d := (int(now.Weekday()) + 1) % len(this.days)
	for i := 0; i < len(this.days); i++ {
		if this.days[d] {
			// This is the next run
			numDaysBetween := i + 1
			return time.Date(now.Year(), now.Month(), now.Day(),
					start.Hour(), start.Minute(), 0, 0, this.timezone).AddDate(0, 0, numDaysBetween),
				time.Date(now.Year(), now.Month(), now.Day(),
					end.Hour(), end.Minute(), 0, 0, this.timezone).AddDate(0, 0, numDaysBetween), nil
		}

		d++
		if d == len(this.days) {
			d = 0
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

		parms := make(map[string]interface{}, 1)
		parms["task"] = task

		after := task.startTime.Sub(time.Now())
		errT := scheduler.ScheduleStoppableTask(session, "auto_update_statistics", "", after, execTask, stopTask, parms, "",
			context)
		if errT != nil {
			err = errors.NewAusSchedulingError(task.startTime, task.endTime, errT)
			continue
		}

		// Task scheduling successful
		return nil
	}

	// Task scheduling unsuccessful
	return err
}

func execTask(context scheduler.Context, parms interface{}, stopChannel <-chan bool) (interface{}, []errors.Error) {

	if params, okP := parms.(map[string]interface{}); okP {
		var task taskInfo
		if v, ok := params["task"]; ok {
			if task, ok = v.(taskInfo); !ok {
				return nil, []errors.Error{errors.NewAusTaskInvalidInfoError("execution", "task", v)}
			}
		} else {
			return nil, []errors.Error{errors.NewAusTaskInvalidInfoError("execution", "task", nil)}
		}

		var activeUpdStatLock sync.Mutex
		// The currently executing update statistics.
		var activeUpdStat *ausOutput
		// Indicates that a stop signal has been received.
		// Further code must check this variable to determine if further processing can proceed.
		abort := false

		timedOut := false
		timer := time.AfterFunc(task.endTime.Sub(task.startTime), func() {
			activeUpdStatLock.Lock()
			updStat := activeUpdStat
			abort = true
			activeUpdStatLock.Unlock()

			// Stop any currently running update statistics
			if updStat != nil {
				updStat.stopActiveUpdStat()
			}

			timedOut = true
			logging.Errorf(AUS_LOG_PREFIX + "Task stopped as timeout exceeded.")
		})

		go func() {
			_, ok := <-stopChannel
			// If a stop signal was received
			if ok {
				// Stops the timer if not already fired/ stopped.
				// Because once the stop signal is received there is no need for the timer to fire.
				timer.Stop()

				activeUpdStatLock.Lock()
				updStat := activeUpdStat
				abort = true
				activeUpdStatLock.Unlock()

				// Stop any currently running update statistics
				if updStat != nil {
					updStat.stopActiveUpdStat()
				}

				logging.Warnf(AUS_LOG_PREFIX + "Task stopped due to user cancellation.")
			}
			return
		}()

		defer func() {
			// Stops the timer if not already fired/ stopped.
			timer.Stop()
			timer = nil

			r := recover()
			if r != nil {
				buf := make([]byte, 1<<16)
				n := runtime.Stack(buf, false)
				s := string(buf[0:n])
				logging.Severef(AUS_LOG_PREFIX+"Panic in task execution: %v\n%v", r, s)
			}

			// End the AUS Evaluation Task
			// Schedule the next task run. Do not consider "today" in the determination of the window of the next run.
			ausCfg.schedule(task.version, false)
			emptyTime := time.Time{}
			ausCfg.setCurrentWindow(true, emptyTime, emptyTime, "")

		}()

		logging.Infof(AUS_LOG_PREFIX+"Execution of the task scheduled between %v and %v started.",
			task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT))

		// If the configuration version is different from the version with which the task was scheduled
		// that means a new configuration was received. Do not execute the task.
		ausCfg.Lock()
		if task.version != ausCfg.configVersion {
			ausCfg.Unlock()
			return "Newer version of settings detected. This task was scheduled from older settings. Task not executed.", nil
		}

		// Set the current task window information
		ausCfg.setCurrentWindow(false, task.startTime, task.endTime, task.version)

		// Get the global information before starting execution. These values will be used for the course of this task run.
		// This is so that consistent global settings are used during a task's entirety in the case changes to the global settings
		// occur during the task execution.
		globalChangePercentage := ausCfg.settings.changePercentage
		allBuckets := ausCfg.settings.allBuckets
		ausCfg.Unlock()

		ds, ok := datastore.GetDatastore().(datastore.Datastore2)
		if !ok || ds == nil {
			// This should not ideally happen as migration should have completed.
			logging.Errorf(AUS_LOG_PREFIX + "Task not started as cluster is not migrated to a supported version")
			return nil, []errors.Error{errors.NewAusNotInitialized()}
		}

		logging.Infof(AUS_LOG_PREFIX + fmt.Sprintf(
			"Configurations of task: start_time: %v end_time: %v change_percentage: %v all_buckets: %v internal version: %v",
			task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT), globalChangePercentage,
			allBuckets, task.version))

		var errs errors.Errors

		// Fuzzy Start
		// Do not start task if the Load Factor of the query node is too high.
		for retry := 0; retry < _MAX_RETRY; retry++ {
			if ausCfg.server.LoadFactor() <= _MAX_LOAD_FACTOR {
				break
			} else if retry == _MAX_RETRY-1 {
				// Exit on the last retry if the load factor is still beyond the threshold. Do not wastefully perform the sleep.
				logging.Errorf(AUS_LOG_PREFIX + "Task not started due to existing load on the node.")
				return nil, append(errs, errors.NewAusTaskNotStartedError())
			} else {
				time.Sleep(_RETRY_INTERVAL)
			}
		}

		var keyspacesEvaluated []interface{}
		var keyspacesUpdated []interface{}

		// Start the AUS Evaluation Task
		// Keep a list of buckets to do randomized selection against
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

		if len(buckets) != 0 {
			// Attempt to prevent starvation.
			// By iterating through the list of buckets/ scopes/ collections starting at a random index, we attempt to prevent
			// always performing AUS on the first keyspaces in the list. As keyspaces at the end of the list will routinely be
			// starved of evaluation/update in the case of timeout/cancellations.

			// Iterate through the buckets list. Generate the random start index.
			gen := rand.New(rand.NewSource(time.Now().Unix()))
			b := gen.Int() % len(buckets)
			coordKeyPrefix := AUS_COORDINATION_DOC_PREFIX + task.version

		bucketLoop:
			for i := 0; i < len(buckets); i++ {

				if abort {
					break
				}

				bucket := buckets[b]
				b++
				if b == len(buckets) {
					b = 0
				}

				scopeIds, err := bucket.ScopeIds()
				if err != nil || len(scopeIds) == 0 {
					continue
				}

				systemCollection, err := getSystemCollection(bucket.Name())
				if err != nil {
					errs = append(errs, err)
					continue
				}

				var bucketSettingRetreived bool
				bucketSetting := defaultKeyspaceSettings

				// Iterate through the scopes list. Generate the random start index.
				s := gen.Int() % len(scopeIds)
				for j := 0; j < len(scopeIds); j++ {

					if abort {
						break bucketLoop
					}

					scope, err := bucket.ScopeById(scopeIds[s])
					if err != nil {
						continue
					}

					s++
					if s == len(scopeIds) {
						s = 0
					}

					collectionIds, err := scope.KeyspaceIds()
					if err != nil || len(collectionIds) == 0 {
						continue
					}

					var scopeSettingRetreived bool
					scopeSetting := defaultKeyspaceSettings

					// Iterate through the collections list. Generate the random start index.
					c := gen.Int() % len(collectionIds)
					for k := 0; k < len(collectionIds); k++ {

						if abort {
							break bucketLoop
						}

						coll, err := scope.KeyspaceById(collectionIds[c])
						if err != nil {
							continue
						}

						c++
						if c == len(collectionIds) {
							c = 0
						}

						// Check if any other node is performing AUS on the collection by attempting to INSERT a unique
						// coordination document for the keyspace.
						// if INSERT fails with a "duplicate key" error, then another node has performed AUS on it.
						dpairs := make([]value.Pair, 1)

						// Include task version in coordination doc to prevent conflicts
						// when config changes occur before previous doc's TTL expires
						dpairs[0].Name = coordKeyPrefix + coll.QualifiedName()
						dpairs[0].Value = value.EMPTY_ANNOTATED_OBJECT

						// The expiration of the coordination document should be the end time of this task window.
						dpairs[0].Options = value.NewValue(map[string]interface{}{
							"expiration": task.endTime.Unix()})

						_, _, iErrs := systemCollection.Insert(dpairs, datastore.NULL_QUERY_CONTEXT, false)

						if len(iErrs) > 0 {
							if !iErrs[0].HasCause(errors.E_DUPLICATE_KEY) {
								errs = append(errs, errors.NewAusTaskError("coordination", errs[0]))
								logging.Errorf(AUS_LOG_PREFIX + fmt.Sprintf(
									"Error inserting the coordination document for keyspace %s: %v", coll.QualifiedName(),
									iErrs[0]))
							}
							continue
						}

						// Get the bucket level setting
						if !bucketSettingRetreived {
							val, errsF := fetchSystemCollection(bucket.Name(), _AUS_SETTINGS_BUCKET_DOC_KEY)
							if len(errsF) > 0 {
								logging.Errorf(AUS_LOG_PREFIX + fmt.Sprintf(
									"Error fetching settings document for bucket %v: %v", bucket.Name(), errsF))
								errs = append(errs, errsF[0])
								continue
							} else if val != nil {
								setting, err := validateAusSettingDoc(val)
								if err != nil {
									logging.Errorf(AUS_LOG_PREFIX + fmt.Sprintf(
										"Invalid settings document for bucket %v: %v", bucket.Name(), err))
									errs = append(errs, err)
									continue
								}
								bucketSetting = setting
							}
							bucketSettingRetreived = true
						}

						if bucketSetting.enable == value.FALSE {
							continue
						}

						// Get scope level setting
						if !scopeSettingRetreived {
							val, errsS := fetchSystemCollection(bucket.Name(),
								ausSettingsScopeKey(scope.Uid(), scope.Name()))
							if len(errsS) > 0 {
								logging.Errorf(AUS_LOG_PREFIX + fmt.Sprintf(
									"Error fetching settings document for scope %v: %v", algebra.PathFromParts("default",
										bucket.Name(), scope.Name()), errsS))
								errs = append(errs, errsS[0])
								continue
							} else if val != nil {
								setting, err := validateAusSettingDoc(val)
								if err != nil {
									logging.Errorf(AUS_LOG_PREFIX + fmt.Sprintf(
										"Invalid settings document for scope %v: %v", algebra.PathFromParts("default",
											bucket.Name(), scope.Name()), err))
									errs = append(errs, err)
									continue
								}
								scopeSetting = setting
							}
							scopeSettingRetreived = true
						}

						if scopeSetting.enable == value.FALSE {
							continue
						}

						// Get collection level setting
						collSetting := defaultKeyspaceSettings
						val, errsC := fetchSystemCollection(bucket.Name(),
							ausSettingsCollectionKey(scope.Uid(), scope.Name(), coll.Uid(), coll.Name()))
						if len(errsC) > 0 {
							logging.Errorf(AUS_LOG_PREFIX + fmt.Sprintf(
								"Error fetching settings document for collection %v: %v", coll.QualifiedName(), errsC))
							errs = append(errs, errsC[0])
							continue
						} else if val != nil {
							setting, err := validateAusSettingDoc(val)
							if err != nil {
								logging.Errorf(AUS_LOG_PREFIX + fmt.Sprintf(
									"Invalid settings document for collection %v: %v", coll.QualifiedName(), err))
								errs = append(errs, err)
								continue
							}
							collSetting = setting
						}

						if collSetting.enable == value.FALSE {
							continue
						}

						// Get the "change percentage" at which to evaluate this keyspace
						change := globalChangePercentage

						if collSetting.changePercentage >= 0 {
							change = collSetting.changePercentage
						} else if scopeSetting.changePercentage >= 0 {
							change = scopeSetting.changePercentage
						} else if bucketSetting.changePercentage >= 0 {
							change = bucketSetting.changePercentage
						}

						keyspacesEvaluated = append(keyspacesEvaluated, coll.QualifiedName())

						// Evaluation Phase
						if abort {
							break bucketLoop
						}
						exprs, customUpdStats, err := aus.Evaluation(coll, change)
						if err != nil {
							errs = append(errs, errors.NewAusEvaluationStageError(coll.QualifiedName(), err))
							logging.Errorf(AUS_LOG_PREFIX+
								"Auto Update Statistics task's Evaluation phase for keyspace %s encountered an error: %v",
								coll.QualifiedName(), err)
							continue
						}

						if len(exprs) > 0 || len(customUpdStats) > 0 {
							keyspacesUpdated = append(keyspacesUpdated, coll.QualifiedName())
						}

						// Update Phase
						// Perform the statistics update for all index expressions that require default UPDATE STATISTICS options
						if len(exprs) > 0 {

							if abort {
								break bucketLoop
							}

							output := newAusOutput()
							activeUpdStatLock.Lock()
							activeUpdStat = output
							activeUpdStatLock.Unlock()

							err := output.executeUpdateStatistics(coll, exprs, nil)

							activeUpdStatLock.Lock()
							activeUpdStat = nil
							activeUpdStatLock.Unlock()

							if err != nil {
								logging.Errorf(AUS_LOG_PREFIX+
									"Auto Update Statistics task's Update phase for keyspace %s encountered an error: %v",
									coll.QualifiedName(), err)

								errs = append(errs, errors.NewAusUpdateStageError(coll.QualifiedName(), err))
							}

							if output.err != nil {
								errs = append(errs, errors.NewAusUpdateStageError(coll.QualifiedName(), output.err))
								logging.Errorf(AUS_LOG_PREFIX+
									"Auto Update Statistics task's Update phase for keyspace %s encountered an error: %v",
									coll.QualifiedName(), output.err)
							}
						}

						// Perform the statistics update for all index expressions that require customized UPDATE STATISTICS options
						if len(customUpdStats) > 0 {
							if abort {
								break bucketLoop
							}

							for i, _ := range customUpdStats {
								cExprs := customUpdStats[i].Expressions()

								with := value.NewValue(map[string]interface{}{
									"sample_size": customUpdStats[i].SampleSize(),
									"resolution":  customUpdStats[i].Resolution(),
								})

								output := newAusOutput()
								activeUpdStatLock.Lock()
								activeUpdStat = output
								activeUpdStatLock.Unlock()

								err := output.executeUpdateStatistics(coll, cExprs, with)

								activeUpdStatLock.Lock()
								activeUpdStat = nil
								activeUpdStatLock.Unlock()

								if err != nil {
									logging.Errorf(AUS_LOG_PREFIX+
										"Auto Update Statistics task's Update phase for keyspace %s encountered an error: %v",
										coll.QualifiedName(), err)

									errs = append(errs, errors.NewAusUpdateStageError(coll.QualifiedName(), err))
								}

								if output.err != nil {
									logging.Errorf(AUS_LOG_PREFIX+
										"Auto Update Statistics task's Update phase for keyspace %s encountered an error: %v",
										coll.QualifiedName(), output.err)
									errs = append(errs, errors.NewAusUpdateStageError(coll.QualifiedName(), output.err))
								}
							}
						}
					}

				}

			}
		}

		if timedOut {
			errs = append(errs, errors.NewAusTaskTimeoutExceeded())
		}

		// Cache task history in system:tasks_cache
		taskRv := make(map[string]interface{}, 4)
		if len(buckets) == 0 {
			if allBuckets {
				s := "No buckets detected in cluster."
				logging.Infof(AUS_LOG_PREFIX + s)
				taskRv["info"] = s
			} else {
				s := "No buckets loaded on the node."
				logging.Infof(AUS_LOG_PREFIX + s)
				taskRv["info"] = s
			}
		}

		// Persist task history in the log
		logging.Infof(AUS_LOG_PREFIX + fmt.Sprintf("Keyspaces Evaluated: %v", keyspacesEvaluated))
		logging.Infof(AUS_LOG_PREFIX + fmt.Sprintf("Keyspaces qualified for Update Phase: %v", keyspacesUpdated))

		taskRv["Configuration"] = map[string]interface{}{
			"start_time":        task.startTime.Format(util.DEFAULT_FORMAT),
			"end_time":          task.endTime.Format(util.DEFAULT_FORMAT),
			"internal_version":  task.version,
			"change_percentage": globalChangePercentage,
			"all_buckets":       allBuckets,
		}

		if len(keyspacesEvaluated) > 0 {
			taskRv["keyspaces_evaluated"] = keyspacesEvaluated
		}

		if len(keyspacesUpdated) > 0 {
			taskRv["keyspaces_updated"] = keyspacesUpdated
		}

		logging.Infof(AUS_LOG_PREFIX+"Execution of the task scheduled between %v and %v has completed.",
			task.startTime.Format(util.DEFAULT_FORMAT), task.endTime.Format(util.DEFAULT_FORMAT))

		return taskRv, errs

	} else {
		return nil, []errors.Error{errors.NewAusTaskInvalidInfoError("execution", "param", parms)}
	}

}

func stopTask(context scheduler.Context, parms interface{}) (interface{}, []errors.Error) {

	if params, okP := parms.(map[string]interface{}); okP {
		var task taskInfo
		if v, ok := params["task"]; ok {
			if task, ok = v.(taskInfo); !ok {
				return nil, []errors.Error{errors.NewAusTaskInvalidInfoError("execution", "task", v)}
			}
		} else {
			return nil, []errors.Error{errors.NewAusTaskInvalidInfoError("execution", "task", nil)}
		}

		rv := fmt.Sprintf("Task scheduled between %v and %v has been stopped.", task.startTime, task.endTime)

		// Schedule the next task run. Do not consider "today" in the determination of the window of the next run
		versionChange, err := ausCfg.schedule(task.version, false)
		if versionChange {
			rv += " Detected global settings change."
			return rv, nil
		}

		if err != nil {
			return rv, []errors.Error{err}
		}

		return rv, nil

	} else {
		return nil, []errors.Error{errors.NewAusTaskInvalidInfoError("execution", "param", parms)}
	}

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
		util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_CBO), optimizer.NewOptimizer(), datastore.DEF_KVTIMEOUT,
		qServer.Timeout())

	if qServer.MemoryQuota() > 0 {
		ctx.SetMemoryQuota(qServer.MemoryQuota())
		ctx.SetMemorySession(memory.Register())
	}

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

// context.Output implementation for executions in AUS related operations
type ausOutput struct {
	err           errors.Error
	mutationCount uint64
	activeUpdStat *datastore.ValueConnection
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
		this.stopActiveUpdStat()
	}
}

func (this *ausOutput) Fatal(err errors.Error) {
	if this.err == nil {
		this.err = err
		this.stopActiveUpdStat()
	}
}

func (this *ausOutput) Error(err errors.Error) {
	if this.err == nil {
		this.err = err
		this.stopActiveUpdStat()
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
	atomic.AddUint64(&this.mutationCount, i)
}

func (this *ausOutput) MutationCount() uint64 {
	return atomic.LoadUint64(&this.mutationCount)
}

func (this *ausOutput) SetSortCount(i uint64) {
	// do nothing
}

func (this *ausOutput) SortCount() uint64 {
	return uint64(0)
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

func (this *ausOutput) stopActiveUpdStat() {
	if this.activeUpdStat != nil {
		select {
		case this.activeUpdStat.StopChannel() <- false:
		default:
		}
	}
}

func (this *ausOutput) executeUpdateStatistics(collection datastore.Keyspace, terms expression.Expressions,
	options value.Value) errors.Error {
	context, err := newContext(this)
	if err != nil {
		return err
	}

	opContext := execution.NewOpContext(context)
	updStat, err := ausCfg.server.Datastore().StatUpdater()
	if err != nil {
		return err
	}

	this.activeUpdStat = datastore.NewValueConnection(opContext)
	updStat.UpdateStatistics(collection, nil, terms, options, this.activeUpdStat, opContext, false)
	this.stopActiveUpdStat()
	this.activeUpdStat = nil
	return nil
}
