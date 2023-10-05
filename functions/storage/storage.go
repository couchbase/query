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
	"github.com/couchbase/query/value"
)

const (
	_NOT_MIGRATING = iota
	_MIGRATING
	_MIGRATED
	_PART_MIGRATED
)

const _UDF_MIGRATION = "UDF"
const _GRACE = 30 * time.Second

var migrating int = _NOT_MIGRATING
var migratingLock sync.Mutex
var countDownStarted time.Time

func Migrate() {

	// no migration needed if serverless
	if tenant.IsServerless() {
		return
	}

	// or functions already migrated
	if migration.IsComplete(_UDF_MIGRATION) {
		logging.Infof("UDF migration: Already done")
		migrating = _MIGRATED
		return
	}

	datastore.RegisterMigrationAbort(abortMigration, datastore.HAS_SYSTEM_COLLECTION)

	countDownStarted = time.Now()
	logging.Infof("UDF migration: Evaluating migration state")

	// no migration needed if the datastore doesn't support scanning buckets directly
	ds, ok := datastore.GetDatastore().(datastore.Datastore2)
	if !ok || ds == nil {
		logging.Warnf("UDF migration: Migration not done - datastore does not support scanning buckets directly")
		migrating = _MIGRATED
		return
	}

	bucketCount := 0
	newBucketCount := 0
	ds.LoadAllBuckets(func(b datastore.ExtendedBucket) {
		bucketCount++
		if b.HasCapability(datastore.HAS_SYSTEM_COLLECTION) {
			newBucketCount++
		}
	})

	// if all the buckets appear migrated attempt migration now (including when bucketCount == 0)
	if bucketCount == newBucketCount {
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
	}, datastore.HAS_SYSTEM_COLLECTION)
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

type migrateBucket struct {
	sync.Mutex
	name  string
	state int
}

var migrations map[string]*migrateBucket
var migrationsLock sync.Mutex

func migrateAll() {

	// TODO KV doesn't like being hammered straight away so wait for KV to prime before migrating
	countDown := time.Since(countDownStarted)
	if countDown < _GRACE {
		time.Sleep(_GRACE - countDown)
	}

	logging.Infof("UDF migration: Start migration of all buckets")

	migrating = _MIGRATING

	// if we are here, we know we have an extended datastore
	ds := datastore.GetDatastore().(datastore.Datastore2)
	if ds != nil {
		ds.LoadAllBuckets(func(b datastore.ExtendedBucket) {
			if !b.HasCapability(datastore.HAS_SYSTEM_COLLECTION) {
				logging.Infof("UDF migration: Bucket %s missing system collection capability", b.Name())
			} else {
				checkSystemCollection(b.Name())

				doMigrateBucket(b.Name())
			}
		})

		if checkMigrationComplete() {
			logging.Infof("UDF migration: End migration of all buckets")
		} else {
			logging.Errorf("UDF migration: Migration incomplete, please restart node")
		}
	} else {
		logging.Errorf("UDF migration: Unexpected error - datastore not available")
	}
}

func checkMigrateBucket(name string) {

	// is migration complete?
	migratingLock.Lock()
	if migrating == _MIGRATED {
		migratingLock.Unlock()
		return
	} else if migration.IsComplete(_UDF_MIGRATION) {
		migrating = _MIGRATED
		migratingLock.Unlock()
		return
	}
	migratingLock.Unlock()

	// avoid parallel migration of each bucket on this query node
	migrationsLock.Lock()
	bucket, ok := migrations[name]
	if !ok {
		bucket = &migrateBucket{
			name:  name,
			state: _NOT_MIGRATING,
		}
		migrations[name] = bucket
	}
	migrationsLock.Unlock()

	logging.Infof("UDF migration: Evaluating bucket (%s) migration state (%v)", name, bucket.state)

	doSysColl := false
	doMigrate := true
	bucket.Lock()
	if bucket.state == _NOT_MIGRATING {
		bucket.state = _MIGRATING
		doSysColl = true
	} else if bucket.state == _PART_MIGRATED {
		bucket.state = _MIGRATING
	} else if bucket.state == _MIGRATING || bucket.state == _MIGRATED {
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
	}

	if doSysColl {
		checkSystemCollection(name)
	}

	// TODO KV doesn't like being hammered straight away so wait for KV to prime before migrating
	countDown := time.Since(countDownStarted)
	if countDown < _GRACE {
		time.Sleep(_GRACE - countDown)
	}

	// if migration completed while we are waiting ...
	migratingLock.Lock()
	if migrating == _MIGRATED {
		migratingLock.Unlock()
		return
	} else if migration.IsComplete(_UDF_MIGRATION) {
		migrating = _MIGRATED
		migratingLock.Unlock()
		return
	}
	migratingLock.Unlock()

	b := doMigrateBucket(name)

	bucket.Lock()
	if b {
		bucket.state = _MIGRATED
	} else {
		bucket.state = _PART_MIGRATED
	}
	bucket.Unlock()

	checkMigrationComplete()
}

func doMigrateBucket(name string) bool {

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
			return errors.NewMigrationError(_UDF_MIGRATION, "Error deleting old entry", parts, err)
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

func checkSystemCollection(name string) {
	ds := datastore.GetDatastore()
	if ds != nil {
		// check existence of system collection, no need to create primary index
		err := ds.CheckSystemCollection(name, "")
		if err != nil {
			logging.Errorf("UDF migration: Error during UDF migration for bucket %s, system collection does not exist - %v", name, err)
		} else {
			logging.Infof("UDF migration: System collection available (bucket %s)", name)
		}
	} else {
		logging.Errorf("UDF migration: Unexpected error - datastore not available")
	}
}

func checkMigrationComplete() bool {
	migratingLock.Lock()
	defer migratingLock.Unlock()

	if migrating == _MIGRATED {
		return true
	}

	complete := true
	err1 := metaStorage.ForeachBody(func(parts []string, body functions.FunctionBody) errors.Error {
		if len(parts) == 4 {
			// still entries to be migrated
			// return an error such that we don't need to continue the scan
			complete = false
			return errors.NewMigrationInternalError(_UDF_MIGRATION, "Entry found in metakv", nil, nil)
		}
		return nil
	})
	if err1 != nil {
		switch err1.Code() {
		case errors.E_MIGRATION_INTERNAL:
			// no-op, ignore this error which indicates migration not complete
		default:
			logging.Errorf("UDF migration: Error during scan of metakv - %v", err1)
			return false
		}
	}

	if complete {
		migration.Complete(_UDF_MIGRATION)
		migrating = _MIGRATED
	}

	return complete
}

func UseSystemStorage() bool {
	if migrating == _MIGRATED || tenant.IsServerless() {
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

	migration.Await(_UDF_MIGRATION)

	// mark migrated for those migrations executed by another node
	migrating = _MIGRATED
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

	if migrating == _MIGRATED {
		return "UDF migration: Already completed.\n", nil
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

	migration.Complete(_UDF_MIGRATION)
	migrating = _MIGRATED

	res.WriteString("UDF migration: Aborted.\n")
	return res.String(), nil
}
