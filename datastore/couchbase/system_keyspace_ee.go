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
	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

const (
	N1QL_SYSTEM_BUCKET = "N1QL_SYSTEM_BUCKET"
	N1QL_SYSTEM_SCOPE  = "N1QL_SYSTEM_SCOPE"
	N1QL_CBO_STATS     = "N1QL_CBO_STATS"
)

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	defaultPool, er := loadNamespace(s, "default")
	if er != nil {
		return er
	}

	// create/get system bucket/scope/collection
	sysBucket, er := defaultPool.keyspaceByName(N1QL_SYSTEM_BUCKET)
	if er != nil {
		// only ignore bucket/keyspace not found error
		if er.Code() != 12003 && er.Code() != 12020 {
			return er
		}

		// create bucket
		_, err := cb.GetSystemBucket(&s.client, defaultPool.cbNamespace, N1QL_SYSTEM_BUCKET)
		if err != nil {
			return errors.NewCbCreateSystemBucketError(N1QL_SYSTEM_BUCKET, err)
		}

		// no need for a retry loop, cb.GetSystemBucket() call above should
		// have made sure that BucketMap is updated already
		defaultPool.refresh()
		sysBucket, er = defaultPool.keyspaceByName(N1QL_SYSTEM_BUCKET)
		if er != nil {
			return er
		}
	}

	sysScope, er := sysBucket.ScopeByName(N1QL_SYSTEM_SCOPE)
	if er != nil {
		if er.Code() != 12021 {
			// only ignore scope not found error
			return er
		}

		// allow "already exists" error in case of duplicated Create call
		er = sysBucket.CreateScope(N1QL_SYSTEM_SCOPE)
		if er != nil && !cb.AlreadyExistsError(er) {
			return er
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
				return er
			}

			sysScope, er = sysBucket.ScopeByName(N1QL_SYSTEM_SCOPE)
			if sysScope != nil {
				break
			} else if er != nil && er.Code() != 12021 {
				return er
			}
		}
		if sysScope == nil {
			return errors.NewCbBucketCreateScopeError(N1QL_SYSTEM_BUCKET+"."+N1QL_SYSTEM_SCOPE, nil)
		}
	}

	cboStats, er := sysScope.KeyspaceByName(N1QL_CBO_STATS)
	if er != nil {
		if er.Code() != 12003 {
			// only ignore keyspace not found error
			return er
		}

		// allow "already exists" error in case of duplicated Create call
		er = sysScope.CreateCollection(N1QL_CBO_STATS)
		if er != nil && !cb.AlreadyExistsError(er) {
			return er
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
				return er
			}

			sysScope, er = sysBucket.ScopeByName(N1QL_SYSTEM_SCOPE)
			if er != nil {
				return er
			}

			cboStats, er = sysScope.KeyspaceByName(N1QL_CBO_STATS)
			if cboStats != nil {
				break
			} else if er != nil && er.Code() != 12003 {
				return er
			}
		}
		if cboStats == nil {
			return errors.NewCbBucketCreateCollectionError(N1QL_SYSTEM_BUCKET+"."+N1QL_SYSTEM_SCOPE+"."+N1QL_CBO_STATS, nil)
		}
	}

	// create primary index
	// make sure we have indexer3 first
	indexer, er := cboStats.Indexer(datastore.GSI)
	if er != nil {
		return er
	}

	indexer3, ok := indexer.(datastore.Indexer3)
	if !ok {
		cb.DropSystemBucket(&s.client, N1QL_SYSTEM_BUCKET)
		return errors.NewInvalidGSIIndexerError("Cannot create system bucket/scope/collection")
	}

	_, er = indexer3.IndexByName(dictionary.CBO_STATS_PRIMARY_INDEX)
	if er != nil {
		// IndexByName currently returns a generic error code (5000), and only returns
		// an error in case of "index not found"
		_, er = indexer3.CreatePrimaryIndex3(requestId, dictionary.CBO_STATS_PRIMARY_INDEX, nil, nil)
		if er != nil {
			return er
		}
	}

	return nil
}

func (s *store) HasSystemCBOStats() (bool, errors.Error) {
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

	cboStats, er := sysScope.KeyspaceByName(N1QL_CBO_STATS)
	if er != nil {
		return false, er
	}

	return (cboStats != nil), nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", N1QL_SYSTEM_BUCKET, N1QL_SYSTEM_SCOPE, N1QL_CBO_STATS)
}
