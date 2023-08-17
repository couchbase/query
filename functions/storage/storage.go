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
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
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
	_AWAITING_BUCKETS
	_WAITING
	_MIGRATING
	_MIGRATED
)

const _UDF_MIGRATION = "UDF"
const _GRACE = 30 * time.Second

var migrating int = _NOT_MIGRATING
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

	countDownStarted = time.Now()
	logging.Infof("UDF migration: Evaluating migration state")

	// check for outstanding migration
	if migration.Resume(_UDF_MIGRATION) {
		logging.Infof("UDF migration: Resuming migration")
		go migrate()
		return
	}

	// no migration needed if the datastore doesn't support scanning buckets directly
	ds, ok := datastore.GetDatastore().(datastore.Datastore2)
	if !ok || ds == nil {
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

	// or there are no buckets
	if bucketCount == 0 {

		// this is unlikely, but if we can't complete migration and
		// other nodes haven't, in the immediate there's no issue
		// as there's no scope UDFs to use anyway
		// when a bucket is added, must restart the node to pick up
		// migration at the next step
		// TODO find better policy
		if migration.TryComplete(_UDF_MIGRATION) {
			logging.Infof("UDF migration: Not needed on a new cluster")
			migrating = _MIGRATED
		} else {
			logging.Warnf("UDF migration: Could not enable _system scope usage, please restart node")
		}
		return
	}

	// if all the buckets appear migrated attempt migration now
	if bucketCount == newBucketCount {
		if migration.Register(_UDF_MIGRATION) {
			logging.Infof("UDF migration: Starting migration")
			go migrate()
		} else if migration.IsComplete(_UDF_MIGRATION) {
			// another process picked it up in the interim
			migrating = _MIGRATED
		} else {
			logging.Infof("UDF migration: Waiting for migration to complete")
			migrating = _WAITING
		}
		return
	}

	datastore.RegisterMigrator(func(b string) {
		// At least one bucket has the new _query collection
		// decide if we are going to do the migration
		switch migrating {
		case _NOT_MIGRATING:
			// is it us?
			if migration.Register(_UDF_MIGRATION) {
				migrating = _AWAITING_BUCKETS
				checkMigrate()
			} else {
				logging.Infof("UDF migration: Migration started on a different node")
				migrating = _WAITING
			}
		case _AWAITING_BUCKETS:
			// it's us, one more bucket
			checkMigrate()

		case _WAITING:
			return
		}
	}, datastore.HAS_SYSTEM_COLLECTION)
}

func checkMigrate() {

	// if we are here, we know we have an extended datastore
	ds := datastore.GetDatastore().(datastore.Datastore2)
	if ds != nil {
		bucketCount := 0
		canMigrate := true

		// this is expensive, but buckets may be dropped and readded
		// in between we initialize and when we check, so we cannot
		// rely on maintaining bucket counts, and have to check all the buckets
		ds.LoadAllBuckets(func(b datastore.ExtendedBucket) {
			if !b.HasCapability(datastore.HAS_SYSTEM_COLLECTION) {
				canMigrate = false
			} else {
				bucketCount++
			}
		})
		if canMigrate {
			migrate()
		} else {
			logging.Infof("UDF migration: Detected %v buckets with system scope", bucketCount)
		}
	}
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

func migrate() {

	// TODO KV doesn't like being hammered straight away so wait for KV to prime before migrating
	countDown := time.Since(countDownStarted)
	if countDown < _GRACE {
		time.Sleep(_GRACE - countDown)
	}

	logging.Infof("UDF migration: Starting scope UDFs migration")

	// TODO it would be useful here to load the cache so that other requests don't hit the storage
	// except that to be useful, this would have to be done on all query nodes, which requires
	// synchronization

	migrating = _MIGRATING
	migration.Start(_UDF_MIGRATION)
	metaStorage.ForeachBody(func(parts []string, body functions.FunctionBody) {
		if len(parts) != 4 {
			return
		}
		name, err := systemStorage.NewScopeFunction(parts[0], parts[1], parts[2], parts[3])
		if err == nil {
			err1 := name.Save(body, true)
			if err1 != nil {
				logging.Warnf("UDF migration: Migrating %v error %v writing body", parts, err1)
			} else {
				logging.Infof("UDF migration: Migrated %v", parts)
			}
			name, err = metaStorage.NewScopeFunction(parts[0], parts[1], parts[2], parts[3])
			err1 = name.Delete()
			if err1 != nil {
				logging.Warnf("UDF migration: Migrating %v error %v deleting old entry", parts, err1)
			}
		} else {
			logging.Warnf("UDF migration: Migrating %v error %v parsing name", parts, err)
		}
	})
	migrating = _MIGRATED
	migration.Complete(_UDF_MIGRATION)
	logging.Infof("UDF migration: Completed scope UDFs migration")
}

func UseSystemStorage() bool {
	if migrating == _MIGRATED || tenant.IsServerless() {
		return true
	}

	if migrating == _NOT_MIGRATING {
		return false
	}

	migration.Await(_UDF_MIGRATION)

	// mark migrated for those migrations executed by another node
	migrating = _MIGRATED
	return true
}
