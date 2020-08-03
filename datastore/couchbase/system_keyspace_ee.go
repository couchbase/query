// Copyright (c) 2020 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you
// may not use this file except in compliance with the License. You
// may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

// +build enterprise

package couchbase

import (
	"time"

	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

const (
	N1QL_SYSTEM_BUCKET     = "N1QL_SYSTEM_BUCKET"
	N1QL_SYSTEM_SCOPE      = "N1QL_SYSTEM_SCOPE"
	N1QL_SYSTEM_COLLECTION = "N1QL_SYSTEM_COLLECTION"
)

func (s *store) CreateSystemCollection() errors.Error {
	defaultPool, er := loadNamespace(s, "default")
	if er != nil {
		return er
	}

	// create/get system bucket/scope/collection in a separate go thread
	go GetSystemStore(&s.client, defaultPool)

	return nil
}

func GetSystemStore(client *cb.Client, defaultPool *namespace) {
	sysBucket, er := defaultPool.keyspaceByName(N1QL_SYSTEM_BUCKET)
	if er != nil {
		// only ignore bucket/keyspace not found error
		if er.Code() != 12003 && er.Code() != 12020 {
			logging.Errorp("Cannot get System Bucket",
				logging.Pair{"error", er},
			)
			return
		}

		// create bucket
		_, err := cb.GetSystemBucket(client, defaultPool.cbNamespace, N1QL_SYSTEM_BUCKET)
		if err != nil {
			logging.Errorp("Cannot create N1QL System Bucket",
				logging.Pair{"error", err},
			)
			return
		}

		// no need for a retry loop, cb.GetSystemBucket() call above should
		// have made sure that BucketMap is updated already
		defaultPool.refresh()
		sysBucket, er = defaultPool.keyspaceByName(N1QL_SYSTEM_BUCKET)
		if er != nil {
			logging.Errorp("Cannot get System Bucket",
				logging.Pair{"error", er},
			)
			return
		}
	}

	sysScope, er := sysBucket.ScopeByName(N1QL_SYSTEM_SCOPE)
	if er != nil {
		if er.Code() != 12021 {
			// only ignore scope not found error
			logging.Errorp("Cannot get System Scope",
				logging.Pair{"error", er},
			)
			return
		}

		// allow "already exists" error in case of duplicated Create call
		er = sysBucket.CreateScope(N1QL_SYSTEM_SCOPE)
		if er != nil && !cb.AlreadyExistsError(er) {
			logging.Errorp("Cannot create System Scope",
				logging.Pair{"error", er},
			)
			return
		}

		// retry till we have the newly created scope available
		maxRetry := 8
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			time.Sleep(interval)
			interval *= 2

			// reload sysBucket
			sysBucket.setNeedsManifest()
			sysBucket, er = defaultPool.keyspaceByName(N1QL_SYSTEM_BUCKET)
			if er != nil {
				logging.Errorp("Cannot get System Bucket",
					logging.Pair{"error", er},
				)
				return
			}

			sysScope, er = sysBucket.ScopeByName(N1QL_SYSTEM_SCOPE)
			if sysScope != nil {
				break
			} else if er != nil && er.Code() != 12021 {
				logging.Errorp("Cannot get System Scope",
					logging.Pair{"error", er},
				)
				return
			}
		}
	}

	_, er = sysScope.KeyspaceByName(N1QL_SYSTEM_COLLECTION)
	if er != nil {
		if er.Code() != 12003 {
			// only ignore keyspace not found error
			logging.Errorp("Cannot get System Collection",
				logging.Pair{"error", er},
			)
			return
		}

		// allow "already exists" error in case of duplicated Create call
		er = sysScope.CreateCollection(N1QL_SYSTEM_COLLECTION)
		if er != nil && !cb.AlreadyExistsError(er) {
			logging.Errorp("Cannot create System Collection",
				logging.Pair{"error", er},
			)
			return
		}

		// retry till we have the newly created collection available
		maxRetry := 8
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			time.Sleep(interval)
			interval *= 2

			// reload sysBucket
			sysBucket.setNeedsManifest()
			sysBucket, er = defaultPool.keyspaceByName(N1QL_SYSTEM_BUCKET)
			if er != nil {
				logging.Errorp("Cannot get System Bucket",
					logging.Pair{"error", er},
				)
				return
			}

			sysScope, er = sysBucket.ScopeByName(N1QL_SYSTEM_SCOPE)
			if er != nil {
				logging.Errorp("Cannot get System Scope",
					logging.Pair{"error", er},
				)
				return
			}

			sysCollection, er := sysScope.KeyspaceByName(N1QL_SYSTEM_COLLECTION)
			if sysCollection != nil {
				break
			} else if er != nil && er.Code() != 12003 {
				logging.Errorp("Cannot get System Collection",
					logging.Pair{"error", er},
				)
				return
			}
		}
	}

	return
}

func (s *store) HasSystemKeyspace() (bool, errors.Error) {
	defaultPool, er := loadNamespace(s, "default")
	if er != nil {
		return false, er
	}

	sysBucket, er := defaultPool.BucketByName(N1QL_SYSTEM_BUCKET)
	if er != nil {
		return false, er
	}

	sysScope, er := sysBucket.ScopeByName(N1QL_SYSTEM_SCOPE)
	if er != nil {
		return false, er
	}

	sysCollection, er := sysScope.KeyspaceByName(N1QL_SYSTEM_COLLECTION)
	if er != nil {
		return false, er
	}

	return (sysCollection != nil), nil
}

func (s *store) GetSystemKeyspace() (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", N1QL_SYSTEM_BUCKET, N1QL_SYSTEM_SCOPE, N1QL_SYSTEM_COLLECTION)
}
