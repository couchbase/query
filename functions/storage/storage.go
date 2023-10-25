//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package storage

import (
	"encoding/json"
	go_errors "errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/metakv"
	"github.com/couchbase/query/functions/system"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/migration"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// node migration state
type nodeState int

const (
	_NOT_MIGRATING = nodeState(iota)
	_MIGRATING
	_MIGRATED
	_ABORTING
	_ABORTED
)

// bucket migration state
type bucketState int

const (
	_BUCKET_NOT_MIGRATING = bucketState(iota)
	_BUCKET_MIGRATING
	_BUCKET_MIGRATED
	_BUCKET_PART_MIGRATED
)

const _UDF_MIGRATION = "UDF"
const _GRACE_PERIOD = 30 * time.Second
const _RETRY_TIME = 10 * time.Second
const _MAX_RETRY = 5

const _N1QL_SYSTEM_BUCKET = "N1QL_SYSTEM_BUCKET"

var migrating nodeState = _NOT_MIGRATING
var migratingLock sync.Mutex
var countDownStarted time.Time
var lastActivity time.Time

type migrateBucket struct {
	sync.Mutex
	name  string
	state bucketState
	index bool
}

var migrations map[string]*migrateBucket
var migrationsLock sync.Mutex

var migrationStartLock sync.Mutex
var migrationStartCond *sync.Cond
var migrationStartWait bool

func Migrate() {

	// no migration needed if serverless
	if tenant.IsServerless() {
		return
	}

	// or functions already migrated
	if checkSetComplete() {
		logging.Infof("UDF migration: Already done")
		return
	}

	datastore.RegisterMigrationAbort(abortMigration, _UDF_MIGRATION, datastore.HAS_SYSTEM_COLLECTION)

	countDownStarted = time.Now()
	lastActivity = time.Now()

	logging.Infof("UDF migration: Evaluating migration state")

	// no migration needed if the datastore doesn't support scanning buckets directly
	ds, ok := datastore.GetDatastore().(datastore.Datastore2)
	if !ok || ds == nil {
		logging.Warnf("UDF migration: Migration not done - datastore does not support scanning buckets directly")
		migrating = _MIGRATED
		datastore.MarkMigrationComplete(true, _UDF_MIGRATION, datastore.HAS_SYSTEM_COLLECTION)
		return
	}

	bucketCount := 0
	newBucketCount := 0
	ds.LoadAllBuckets(func(b datastore.ExtendedBucket) {
		if b.Name() != _N1QL_SYSTEM_BUCKET {
			bucketCount++
			if b.HasCapability(datastore.HAS_SYSTEM_COLLECTION) {
				newBucketCount++
			}
		}
	})

	// if all the buckets appear migrated attempt migration now (including when bucketCount == 0)
	if bucketCount == newBucketCount {
		migrating = _MIGRATING

		go migrateAll()

		return
	}

	migrationsLock.Lock()
	migrations = make(map[string]*migrateBucket, bucketCount)
	migrationsLock.Unlock()

	datastore.RegisterMigrator(func(b string) {
		// At least one bucket has the new _query collection
		migratingLock.Lock()
		switch migrating {
		case _NOT_MIGRATING:
			migrating = _MIGRATING
			fallthrough
		case _MIGRATING:
			migratingLock.Unlock()
			checkMigrateBucket(b)
		default:
			migratingLock.Unlock()
			return
		}
	}, _UDF_MIGRATION, datastore.HAS_SYSTEM_COLLECTION)

	go checkRetryMigration()
}

func checkRetryMigration() {

	// if we are here, we know we have an extended datastore
	ds := datastore.GetDatastore().(datastore.Datastore2)

	// wait until migration has started
	// the node can be in mixed-clustered mode for undetermined time, do not proceed
	// with retry of migration until migration has started
	logging.Infof("UDF migration: Waiting on migration to start")

	started := false
	ds.LoadAllBuckets(func(b datastore.ExtendedBucket) {
		migrationsLock.Lock()
		if bucket, ok := migrations[b.Name()]; ok {
			bucket.Lock()
			if bucket.state != _BUCKET_NOT_MIGRATING {
				started = true
			}
			bucket.Unlock()
		}
		migrationsLock.Unlock()
	})
	if !started {
		migrationStartLock.Lock()
		migrationStartCond = sync.NewCond(&migrationStartLock)
		migrationStartWait = true
		migrationStartCond.Wait()
		migrationStartLock.Unlock()
	}

	// since migration waits for _GRACE_PERIOD before it starts, wait for _GRACE_PERIOD plus
	// some before migration retry
	countDown := time.Since(countDownStarted)
	if countDown < _GRACE_PERIOD {
		time.Sleep(_GRACE_PERIOD - countDown)
	}

	// extra sleep to allow migration to proceed before initiating retry
	time.Sleep(_RETRY_TIME)

	// migration complete?
	migratingLock.Lock()
	if migrating == _MIGRATED || migrating == _ABORTED || migrating == _ABORTING {
		migratingLock.Unlock()
		return
	} else if checkSetComplete() {
		migratingLock.Unlock()
		return
	}
	migratingLock.Unlock()

	retryMigration()
}

func retryMigration() {

	logging.Infof("UDF migration: Gathering migration information on all buckets")

	// make sure the migrations map has information on all relevant buckets
	// note this has to be done after migration has started (i.e. no further changes)
	checkMigrations()

	lastActivity = time.Now()

	for i := 1; i <= _MAX_RETRY; i++ {
		duration := time.Duration(i) * _RETRY_TIME
		for {
			// only initiate retry if there has been no activity for the duration
			inactive := time.Since(lastActivity)
			if inactive < duration {
				time.Sleep(duration - inactive)
			} else {
				break
			}
		}

		// migration complete?
		migratingLock.Lock()
		if migrating == _MIGRATED || migrating == _ABORTED || migrating == _ABORTING {
			migratingLock.Unlock()
			return
		} else if checkSetComplete() {
			migratingLock.Unlock()
			return
		}
		migratingLock.Unlock()

		logging.Infof("UDF migration: Retry migration (%d of %d)", i, _MAX_RETRY)

		// migrationsLock ensures that checkMigrateBucket() cannot proceed while we
		// try migration here
		migrationsLock.Lock()
		for _, bucket := range migrations {
			// quick check for abort without lock
			if migrating == _ABORTING || migrating == _ABORTED {
				migrationsLock.Unlock()
				logging.Infof("UDF migration: Migration is aborting, skip further migration operations")
				return
			}

			bucket.Lock()
			// of all bucket states, only _BUCKET_MIGRATED should skip further migration
			if bucket.state != _BUCKET_MIGRATED {
				if bucket.state == _BUCKET_NOT_MIGRATING {
					// make sure _system scope is available
					err := checkSystemCollection(bucket.name)
					if err != nil {
						bucket.Unlock()
						continue
					}
					bucket.state = _BUCKET_MIGRATING
				}
				// this is probably not the most efficient way to retry migration,
				// since for each bucket we are going to scan metakv entries,
				// however this does allow us to mark migration of individual bucket
				// as _BUCKET_MIGRATED; also reuse the same code as regular migration.
				if doMigrateBucket(bucket.name) {
					bucket.state = _BUCKET_MIGRATED
					if !bucket.index {
						// create primary index in the background
						go createPrimaryIndex(bucket.name)
						bucket.index = true
					}
				} else {
					bucket.state = _BUCKET_PART_MIGRATED
				}
			}
			bucket.Unlock()
		}
		migrationsLock.Unlock()

		if checkMigrationComplete() {
			return
		}

		lastActivity = time.Now()
	}

	// if we get here, migration is not complete after maximum retry, log severe error
	// this requires manual intervention
	logging.Severef("UDF migration: Migration is not complete, please restart node")
}

// migratingLock, if needed, should be obtained from the caller
func checkSetComplete() bool {
	complete, success := migration.IsComplete(_UDF_MIGRATION)
	if complete {
		if success {
			migrating = _MIGRATED
		} else {
			migrating = _ABORTED
		}
		datastore.MarkMigrationComplete(success, _UDF_MIGRATION, datastore.HAS_SYSTEM_COLLECTION)
	}
	return complete
}

func MakeName(bytes []byte) (functions.FunctionName, error) {
	var name_type struct {
		Type string `json:"type"`
	}

	err := json.Unmarshal(bytes, &name_type)
	if err != nil {
		return nil, err
	}

	switch name_type.Type {
	case "global":
		var _unmarshalled struct {
			_         string `json:"type"`
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		}

		err = json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, err
		}
		return metaStorage.NewGlobalFunction(_unmarshalled.Namespace, _unmarshalled.Name)

	case "scope":
		var _unmarshalled struct {
			_         string `json:"type"`
			Namespace string `json:"namespace"`
			Bucket    string `json:"bucket"`
			Scope     string `json:"scope"`
			Name      string `json:"name"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, err
		}
		if _unmarshalled.Namespace == "" || _unmarshalled.Bucket == "" || _unmarshalled.Scope == "" || _unmarshalled.Name == "" {
			return nil, go_errors.New("incomplete function name")
		}
		if UseSystemStorage() {
			return systemStorage.NewScopeFunction(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope,
				_unmarshalled.Name)
		} else {
			return metaStorage.NewScopeFunction(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope,
				_unmarshalled.Name)
		}
	default:
		return nil, fmt.Errorf("unknown name type %v", name_type.Type)
	}
}

func IsInternal(val interface{}) (bool, error) {
	var t string

	switch val := val.(type) {
	case []byte:
		var outer struct {
			Definition json.RawMessage `json:"definition"`
		}
		var language_type struct {
			Language string `json:"#language"`
		}

		err := json.Unmarshal(val, &outer)
		if err != nil {
			return false, err
		}

		err = json.Unmarshal(outer.Definition, &language_type)
		if err != nil {
			return false, err
		}
		t = language_type.Language
	default:
		d, _ := val.(value.Value).Field("definition")
		if d != nil {
			v, _ := d.(value.Value).Field("#language")
			t, _ = v.Actual().(string)
		}
	}

	switch t {
	case "inline":
		return true, nil
	case "golang":
		return false, nil
	case "javascript":
		return false, nil
	default:
		return false, fmt.Errorf("unknown function type %v", t)
	}
}

func DropScope(namespace, bucket, scope string) {
	if UseSystemStorage() {
		systemStorage.DropScope(namespace, bucket, scope)
	} else {
		metaStorage.DropScope(namespace, bucket, scope)
	}
}

func Count(bucket string) (int64, error) {
	if bucket != "" && UseSystemStorage() {
		return systemStorage.Count(bucket)
	} else {
		return metaStorage.Count(bucket)
	}
}

func Get(key string) (value.Value, error) {
	if UseSystemStorage() && algebra.PartsFromPath(key) == 4 {
		return systemStorage.Get(key)
	} else {
		return metaStorage.Get(key)
	}
}

func Foreach(bucket string, f func(path string, v value.Value) error) error {
	if bucket != "" && UseSystemStorage() {
		return systemStorage.Foreach(bucket, f)
	} else {
		return metaStorage.Foreach(bucket, f)
	}
}

func Scan(bucket string, f func(path string) error) error {
	if bucket != "" && UseSystemStorage() {
		return systemStorage.Scan(bucket, f)
	} else {
		return metaStorage.Scan(bucket, f)
	}
}

func ExternalBucketArchive() bool {
	return !UseSystemStorage()
}

func migrateAll() {

	// TODO KV doesn't like being hammered straight away so wait for KV to prime before migrating
	countDown := time.Since(countDownStarted)
	if countDown < _GRACE_PERIOD {
		time.Sleep(_GRACE_PERIOD - countDown)
	}

	// is migration complete?
	migratingLock.Lock()
	if migrating == _MIGRATED || migrating == _ABORTED || migrating == _ABORTING {
		migratingLock.Unlock()
		return
	} else if checkSetComplete() {
		migratingLock.Unlock()
		return
	}
	migratingLock.Unlock()

	logging.Infof("UDF migration: Start migration of all buckets")

	// if we are here, we know we have an extended datastore
	ds := datastore.GetDatastore().(datastore.Datastore2)
	if ds != nil {
		ds.LoadAllBuckets(func(b datastore.ExtendedBucket) {
			if !b.HasCapability(datastore.HAS_SYSTEM_COLLECTION) {
				logging.Infof("UDF migration: Bucket %s missing system collection capability", b.Name())
			} else {
				bucketName := b.Name()
				if bucketName == _N1QL_SYSTEM_BUCKET {
					return
				}

				// quick check for abort without lock
				if migrating == _ABORTING || migrating == _ABORTED {
					logging.Infof("UDF migration: Migration is aborting, skip further migration operations")
					return
				}

				err := checkSystemCollection(bucketName)
				if err == nil {
					if doMigrateBucket(bucketName) {
						go createPrimaryIndex(bucketName)
					}
				}
			}
		})

		if checkMigrationComplete() {
			logging.Infof("UDF migration: End migration of all buckets")
			return
		}
	} else {
		logging.Errorf("UDF migration: Unexpected error - datastore not available")
		return
	}

	retryMigration()
}

func checkMigrateBucket(name string) {

	// is migration complete?
	migratingLock.Lock()
	if migrating == _MIGRATED || migrating == _ABORTED || migrating == _ABORTING {
		migratingLock.Unlock()
		return
	} else if checkSetComplete() {
		migratingLock.Unlock()
		return
	}
	migratingLock.Unlock()

	if name == _N1QL_SYSTEM_BUCKET {
		return
	}

	// avoid parallel migration of each bucket on this query node
	migrationsLock.Lock()
	bucket, ok := migrations[name]
	if !ok {
		bucket = &migrateBucket{
			name:  name,
			state: _BUCKET_NOT_MIGRATING,
		}
		migrations[name] = bucket
	}
	migrationsLock.Unlock()

	logging.Infof("UDF migration: Evaluating bucket (%s) migration state (%v)", name, bucket.state)

	doSysColl := false
	doMigrate := true
	bucket.Lock()
	if bucket.state == _BUCKET_NOT_MIGRATING {
		bucket.state = _BUCKET_MIGRATING
		doSysColl = true
	} else if bucket.state == _BUCKET_PART_MIGRATED {
		bucket.state = _BUCKET_MIGRATING
	} else if bucket.state == _BUCKET_MIGRATING || bucket.state == _BUCKET_MIGRATED {
		doMigrate = false
	} else {
		logging.Errorf("UDF migration: Unexpected bucket migration state %v", bucket.state)
		bucket.Unlock()
		return
	}
	bucket.Unlock()

	if !doMigrate {
		logging.Infof("UDF migration: Migration of bucket %s being performed by another thread", name)
		return
	} else if migrationStartWait {
		needSignal := false
		migrationStartLock.Lock()
		if migrationStartWait && migrationStartCond != nil {
			needSignal = true
			migrationStartWait = false
		}
		migrationStartLock.Unlock()
		if needSignal {
			// there should be only a single waiter
			migrationStartCond.Signal()
		}
	}

	if doSysColl {
		err := checkSystemCollection(name)
		if err != nil {
			bucket.Lock()
			bucket.state = _BUCKET_NOT_MIGRATING
			bucket.Unlock()
			lastActivity = time.Now()
			return
		}
	}

	// TODO KV doesn't like being hammered straight away so wait for KV to prime before migrating
	countDown := time.Since(countDownStarted)
	if countDown < _GRACE_PERIOD {
		time.Sleep(_GRACE_PERIOD - countDown)
	}

	// if migration completed while we are waiting ...
	migratingLock.Lock()
	if migrating == _MIGRATED || migrating == _ABORTED || migrating == _ABORTING {
		migratingLock.Unlock()
		return
	} else if checkSetComplete() {
		migratingLock.Unlock()
		return
	}
	migratingLock.Unlock()

	b := doMigrateBucket(name)

	bucket.Lock()
	if b {
		bucket.state = _BUCKET_MIGRATED
		if !bucket.index {
			// create primary index in the background
			go createPrimaryIndex(name)
			bucket.index = true
		}
	} else {
		bucket.state = _BUCKET_PART_MIGRATED
	}
	bucket.Unlock()

	checkMigrationComplete()

	lastActivity = time.Now()
}

func doMigrateBucket(name string) bool {

	if name == _N1QL_SYSTEM_BUCKET {
		return false
	}

	logging.Infof("UDF migration: Start UDF migration for bucket %s", name)

	// TODO it would be useful here to load the cache so that other requests don't hit the storage
	// except that to be useful, this would have to be done on all query nodes, which requires
	// synchronization

	complete := true
	err1 := metaStorage.ForeachBody(func(parts []string, body functions.FunctionBody) errors.Error {
		if len(parts) != 4 {
			return nil
		}
		if parts[1] != name {
			// not this bucket
			return nil
		}
		logging.Infof("UDF migration: Handling %v", parts)

		name, err := systemStorage.NewScopeFunction(parts[0], parts[1], parts[2], parts[3])
		if err != nil {
			logging.Errorf("UDF migration: Migrating %v error %v parsing name", parts, err)
			return errors.NewMigrationError(_UDF_MIGRATION, "Error parsing name", parts, err)
		}

		err = name.Save(body, false)
		if err != nil {
			logging.Errorf("UDF migration: Migrating %v error %v writing body", parts, err)
			// ignore duplicated function error but return all other errors
			if err.Code() != errors.E_DUPLICATE_FUNCTION {
				return errors.NewMigrationError(_UDF_MIGRATION, "Error writing body to system storage", parts, err)
			}
		} else {
			logging.Infof("UDF migration: Added %v", parts)
		}

		name, err = metaStorage.NewScopeFunction(parts[0], parts[1], parts[2], parts[3])
		if err != nil {
			logging.Errorf("UDF migration: Migrating %v error %v generating metakv function name for deleting old entry", parts, err)
			return errors.NewMigrationError(_UDF_MIGRATION, "Error generating metakv function name for deleting old entry", parts, err)
		}

		err = name.Delete()
		if err != nil {
			logging.Errorf("UDF migration: Migrating %v error %v deleting old entry", parts, err)
			// ignore missing function error but return ll other errors
			if err.Code() != errors.E_MISSING_FUNCTION {
				return errors.NewMigrationError(_UDF_MIGRATION, "Error deleting old entry", parts, err)
			}
		} else {
			logging.Infof("UDF migration: Deleted old entry for %v", parts)
		}

		logging.Infof("UDF migration: Migrated %v", parts)

		return nil
	})
	if err1 != nil {
		logging.Errorf("UDF migration: Error during scan of old UDF definitions - %V", err1)
		complete = false
	}

	logging.Infof("UDF migration: End UDF migration for bucket %s complete %t", name, complete)
	return complete
}

func checkSystemCollection(name string) errors.Error {
	if name == _N1QL_SYSTEM_BUCKET {
		return nil
	}

	ds := datastore.GetDatastore()
	if ds != nil {
		// make sure the system collection exist, but no need for primary index for migration
		// to proceed; since index creation may take additonal time, we'll do that at the end
		// of migration
		err := ds.CheckSystemCollection(name, "")
		if err != nil {
			logging.Errorf("UDF migration: Error during UDF migration for bucket %s, system collection unavailable - %v", name, err)
			return errors.NewMigrationError(_UDF_MIGRATION, fmt.Sprintf("Error during UDF migration for bucket %s - system collection unavailable", name), nil, err)
		} else {
			logging.Infof("UDF migration: System collection available (bucket %s)", name)
		}
	} else {
		logging.Errorf("UDF migration: Unexpected error - datastore not available")
		return errors.NewMigrationInternalError(_UDF_MIGRATION, "Unexpected error - datastore not available", nil, nil)
	}
	return nil
}

func createPrimaryIndex(bucketName string) {
	ds := datastore.GetDatastore()
	if ds != nil {
		requestId, _ := util.UUIDV4()
		err := ds.CheckSystemCollection(bucketName, requestId)
		if err == nil {
			logging.Infof("UDF migration: Primary index on system collection available for bucket %s", bucketName)
		} else if !errors.IsIndexExistsError(err) {
			logging.Errorf("UDF migration: Error creating primary index on system collection for bucket %s - %v", bucketName, err)
		}
	}
}

func checkMigrationComplete() bool {
	migratingLock.Lock()
	defer migratingLock.Unlock()

	if migrating == _MIGRATED || migrating == _ABORTED {
		return true
	} else if migrating == _ABORTING {
		return false
	}

	complete := true
	err1 := metaStorage.ForeachBody(func(parts []string, body functions.FunctionBody) errors.Error {
		if len(parts) == 4 {
			if parts[1] == _N1QL_SYSTEM_BUCKET {
				// ignore entries on N1QL_SYSTEM_BUCKET since that'll be dropped
				return nil
			}

			// still entries to be migrated
			// return an error such that we don't need to continue the scan
			complete = false
			return errors.NewMigrationError(_UDF_MIGRATION, "Entry found in metakv", nil, nil)
		}
		return nil
	})
	if err1 != nil {
		switch err1.Code() {
		case errors.E_MIGRATION:
			// no-op, ignore this error which indicates migration not complete
		default:
			logging.Errorf("UDF migration: Error during scan of metakv - %v", err1)
			return false
		}
	}

	if complete {
		migration.Complete(_UDF_MIGRATION, true)
		migrating = _MIGRATED
		datastore.MarkMigrationComplete(true, _UDF_MIGRATION, datastore.HAS_SYSTEM_COLLECTION)
	}

	return complete
}

// if retrying migration, since we go from the migrations map, make sure all buckets are in the map
func checkMigrations() {
	metaStorage.ForeachBody(func(parts []string, body functions.FunctionBody) errors.Error {
		if len(parts) == 4 {
			bname := parts[1]
			if bname == _N1QL_SYSTEM_BUCKET {
				return nil
			}
			migrationsLock.Lock()
			if _, ok := migrations[bname]; !ok {
				migrations[bname] = &migrateBucket{
					name:  bname,
					state: _BUCKET_NOT_MIGRATING,
				}
			}
			migrationsLock.Unlock()
		}
		return nil
	})
}

func UseSystemStorage() bool {
	if migrating == _MIGRATED || migrating == _ABORTED || tenant.IsServerless() {
		return true
	}

	if migrating == _NOT_MIGRATING {
		migratingLock.Lock()
		notMigrating := migrating == _NOT_MIGRATING
		migratingLock.Unlock()
		if notMigrating {
			return false
		}
	}

	success := migration.Await(_UDF_MIGRATION)

	// mark migrated for those migrations executed by another node
	if migrating != _MIGRATED && migrating != _ABORTED {
		changed := false
		migratingLock.Lock()
		if migrating != _MIGRATED && migrating != _ABORTED {
			if success {
				migrating = _MIGRATED
			} else {
				migrating = _ABORTED
			}
			changed = true
		}
		migratingLock.Unlock()
		if changed {
			datastore.MarkMigrationComplete(success, _UDF_MIGRATION, datastore.HAS_SYSTEM_COLLECTION)
		}
	}
	return true
}

func GetDDLFromDefinition(name string, defn value.Value) string {
	var b strings.Builder
	d, ok := defn.Field("definition")
	if !ok {
		return ""
	}
	l, ok := d.Field("#language")
	if !ok {
		return ""
	}
	b.WriteString("CREATE OR REPLACE FUNCTION ")
	b.WriteString(name)
	b.WriteRune('(')
	if p, ok := d.Field("parameters"); ok {
		for i := 0; ; i++ {
			v, ok := p.Index(i)
			if !ok {
				break
			}
			if i > 0 {
				b.WriteRune(',')
			}
			b.WriteString(v.ToString())
		}
	}
	b.WriteString(") LANGUAGE ")
	b.WriteString(l.ToString())
	b.WriteString(" AS ")
	switch l.ToString() {
	case "inline":
		t, ok := d.Field("text")
		if !ok {
			return ""
		}
		b.WriteString(t.ToString())
	case "javascript":
		t, ok := d.Field("text")
		if !ok {
			o, ok := d.Field("object")
			if !ok {
				return ""
			}
			lib, ok := d.Field("library")
			if !ok {
				return ""
			}
			b.WriteString(o.String())
			b.WriteString(" AT ")
			b.WriteString(lib.String())
		} else {
			b.WriteString(t.String())
		}
	case "golang":
		o, ok := d.Field("object")
		if !ok {
			return ""
		}
		lib, ok := d.Field("library")
		if !ok {
			return ""
		}
		b.WriteString(o.String())
		b.WriteString(" AT ")
		b.WriteString(lib.String())
	}
	return b.String()
}

func abortMigration() (string, errors.Error) {

	var returnMsg string
	migratingLock.Lock()
	if migrating == _MIGRATED {
		returnMsg = "UDF migration: Already completed.\n"
	} else if migrating == _ABORTED {
		returnMsg = "UDF migration: Already aborted.\n"
	} else {
		migrating = _ABORTING
	}
	migratingLock.Unlock()

	if returnMsg != "" {
		return returnMsg, nil
	}

	logging.Severef("UDF migration: Aborting migration")

	var res strings.Builder
	first := true

	metaStorage.Foreach("*", func(name string, v value.Value) error {
		s := GetDDLFromDefinition(name, v)
		if s != "" {
			if first {
				res.WriteString("UDF migration: The following were not migrated and should be recreated:\n\n")
				first = false
			}
			res.WriteString(s)
			res.WriteString(";\n")

			parts := algebra.ParsePath(name)
			name, err := metaStorage.NewScopeFunction(parts[0], parts[1], parts[2], parts[3])
			if err == nil {
				err = name.Delete()
			}
			if err != nil {
				logging.Severef("UDF migration: Abort encountered error: %v cleaning up entry for %v", err, name)
			}
		}
		return nil
	})

	if first {
		res.WriteString("UDF migration: No functions needing migration found.\n")
	} else {
		res.WriteRune('\n')
	}

	migration.Complete(_UDF_MIGRATION, false)
	migrating = _ABORTED
	datastore.MarkMigrationComplete(false, _UDF_MIGRATION, datastore.HAS_SYSTEM_COLLECTION)

	res.WriteString("UDF migration: Aborted.\n")
	return res.String(), nil
}
