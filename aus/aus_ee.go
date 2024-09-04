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
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/migration"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

const (
	_AUS_PATH                    = "/query/auto_update_statistics/"
	_AUS_GLOBAL_SETTINGS_PATH    = _AUS_PATH + "global_settings"
	_AUS_SETTINGS_BUCKET_DOC_KEY = AUS_DOC_PREFIX + "bucket"
)

var ausCfg ausConfig
var daysOfWeek map[string]time.Weekday

// since we fetch documents from _system._query 1 at a time
var _STRING_ANNOTATED_POOL *value.StringAnnotatedPool

type ausConfig struct {
	sync.Mutex
	server      *server.Server
	settings    ausGlobalSettings
	initialized bool
}

type ausGlobalSettings struct {
	enable           bool
	allBuckets       bool
	changePercentage int
	schedule         ausSchedule
	version          int64
}

type ausSchedule struct {
	startTime time.Time
	endTime   time.Time
	timezone  *time.Location
	days      []bool
}

func (this *ausConfig) setInitialized(init bool) {
	this.Lock()
	this.initialized = init
	this.Unlock()
}

// Performs basic initialization of AUS configuration
func InitAus(server *server.Server) {
	ausCfg = ausConfig{
		server:      server,
		initialized: false,
	}

	// run as go routine to prevent hanging at the migration.Await() call
	go startupAus()
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

	// Initialize the days of the week map
	daysOfWeek = initDaysOfWeek()

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
	defer ausCfg.Unlock()

	logging.Infof(AUS_LOG_PREFIX + "New global settings received.")

	var v map[string]interface{}
	err := json.Unmarshal(kve.Value, &v)
	if err != nil {
		logging.Errorf(AUS_LOG_PREFIX+"Error unmarshalling new global settings: %v", err)
		return nil
	}

	if newSettings, err, _ := setAusHelper(v, false); err != nil {
		logging.Errorf(AUS_LOG_PREFIX+"Error during global settings retrieval: %v", err)
		return nil
	} else {
		ausCfg.settings = newSettings
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

	return val, nil
}

// For the global settings path/ key that is to be in metakv:
// Checks if the path/ key is present in metakv
// If not present, attempts to add the path to metakv with the default global settings value
// If the param 'get' is set to true, returns the contents in metakv for the key
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

func SetAus(settings interface{}, distribute bool) (err errors.Error, warnings errors.Errors) {
	if !ausCfg.initialized {
		return errors.NewAusNotInitialized(), nil
	}

	_, err, warnings = setAusHelper(settings, true)
	return err, warnings
}

// Function to validate schema of the input settings document. And optionally distribute said settings document in metakv
func setAusHelper(settings interface{}, distribute bool) (obj ausGlobalSettings, err errors.Error,
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
				} else if cp <= 0 || cp > 100 {
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
		tp, err := time.Parse(time.TimeOnly, t)
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
						if w, ok1 := daysOfWeek[strings.ToLower(strings.TrimSpace(ds))]; ok1 {
							daysNum[w] = true
							continue
						}
					}

					return ausSchedule, errors.NewAusDocInvalidSettingsValue("schedule.days", days), warnings
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
			path := key2path(key, systemCollection.NamespaceId(), bucket)
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

	// Get system collection
	systemCollection, err := getSystemCollection(parts[1])
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
			err = validateAusSettingDoc(pair.Value)
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

// Returns the keyspace path from the KV document key.
// If it returns an empty string, the key is not from system:aus_settings
// Format is like so:
// 1. Bucket level document:
// aus::bucket
// 2. Scope level document:
// aus::scope_id::scope_name
// 3. Collection level document:
// aus::scope_id::collection_id::scope_name.collection_name
func key2path(key, namespace, bucket string) string {
	// strip prefix and scope (and collection) UIDs from the scope (and collection) names
	parts := strings.Split(key, "::")

	// Check if the key is in right bucket/ scope/ collection level document key format
	if len(parts) < 2 || len(parts) > 4 {
		return ""
	}

	// check if the key is prefixed by "aus::"
	if parts[0] != "aus" {
		return ""
	}

	// check if the key was for the bucket
	if parts[1] == "bucket" {
		return algebra.PathFromParts(namespace, bucket)
	}

	// the last element in the parts array post-splitting will be the actual path with the scope & collection names
	ks := parts[len(parts)-1]
	dot := strings.IndexByte(ks, '.')

	// if '.' is not present then the key is for a scope level document
	if dot < 0 {
		return algebra.PathFromParts(namespace, bucket, ks)
	}

	// Otherwise the key is for a collection level document
	return algebra.PathFromParts(namespace, bucket, ks[:dot], ks[dot+1:])
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
		return AUS_DOC_PREFIX + scope.Uid() + "::" + collection.Uid() + "::" + parts[2] + "." + parts[3], parts, nil
	} else {
		// create a scope document
		return AUS_DOC_PREFIX + scope.Uid() + "::" + parts[2], parts, nil
	}

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
func validateAusSettingDoc(doc value.Value) errors.Error {

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
			if _, ok := v.(bool); !ok {
				return errors.NewAusDocInvalidSettingsValue(k, v)
			}
		case "change_percentage":
			if cp, ok := v.(int64); !ok {
				return errors.NewAusDocInvalidSettingsValue(k, v)
			} else if cp <= 0 || cp > 100 {
				return errors.NewAusDocInvalidSettingsValue(k, v)
			}
		default:
			return errors.NewAusDocUnknownSetting(k)
		}
	}

	return nil
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
