// Copyright 2020-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software will
// be governed by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

//go:build enterprise

package couchbase

import (
	"time"

	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

const (
	_N1QL_SYSTEM_BUCKET      = dictionary.N1QL_SYSTEM_BUCKET
	_N1QL_SYSTEM_SCOPE       = dictionary.N1QL_SYSTEM_SCOPE
	_N1QL_CBO_STATS          = dictionary.N1QL_CBO_STATS
	_CBO_STATS_PRIMARY_INDEX = dictionary.CBO_STATS_PRIMARY_INDEX
)

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	defaultPool, er := loadNamespace(s, "default")
	if er != nil {
		return er
	}

	// create/get system bucket/scope/collection
	sysBucket, er := defaultPool.keyspaceByName(_N1QL_SYSTEM_BUCKET)
	if er != nil {
		// only ignore bucket/keyspace not found error
		if er.Code() != 12003 && er.Code() != 12020 {
			return er
		}

		// create bucket
		_, err := cb.GetSystemBucket(&s.client, defaultPool.cbNamespace, _N1QL_SYSTEM_BUCKET)
		if err != nil {
			return errors.NewCbCreateSystemBucketError(_N1QL_SYSTEM_BUCKET, err)
		}

		// no need for a retry loop, cb.GetSystemBucket() call above should
		// have made sure that BucketMap is updated already
		defaultPool.refresh()
		sysBucket, er = defaultPool.keyspaceByName(_N1QL_SYSTEM_BUCKET)
		if er != nil {
			return er
		}
	}

	sysScope, er := sysBucket.ScopeByName(_N1QL_SYSTEM_SCOPE)
	if er != nil {
		if er.Code() != 12021 {
			// only ignore scope not found error
			return er
		}

		// allow "already exists" error in case of duplicated Create call
		er = sysBucket.CreateScope(_N1QL_SYSTEM_SCOPE)
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
			sysBucket, er = defaultPool.keyspaceByName(_N1QL_SYSTEM_BUCKET)
			if er != nil {
				return er
			}

			sysScope, er = sysBucket.ScopeByName(_N1QL_SYSTEM_SCOPE)
			if sysScope != nil {
				break
			} else if er != nil && er.Code() != 12021 {
				return er
			}
		}
		if sysScope == nil {
			return errors.NewCbBucketCreateScopeError(_N1QL_SYSTEM_BUCKET+"."+_N1QL_SYSTEM_SCOPE, nil)
		}
	}

	cboStats, er := sysScope.KeyspaceByName(_N1QL_CBO_STATS)
	if er != nil {
		if er.Code() != 12003 {
			// only ignore keyspace not found error
			return er
		}

		// allow "already exists" error in case of duplicated Create call
		er = sysScope.CreateCollection(_N1QL_CBO_STATS)
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
			sysBucket, er = defaultPool.keyspaceByName(_N1QL_SYSTEM_BUCKET)
			if er != nil {
				return er
			}

			sysScope, er = sysBucket.ScopeByName(_N1QL_SYSTEM_SCOPE)
			if er != nil {
				return er
			}

			cboStats, er = sysScope.KeyspaceByName(_N1QL_CBO_STATS)
			if cboStats != nil {
				break
			} else if er != nil && er.Code() != 12003 {
				return er
			}
		}
		if cboStats == nil {
			return errors.NewCbBucketCreateCollectionError(_N1QL_SYSTEM_BUCKET+"."+_N1QL_SYSTEM_SCOPE+"."+_N1QL_CBO_STATS, nil)
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
		cb.DropSystemBucket(&s.client, _N1QL_SYSTEM_BUCKET)
		return errors.NewInvalidGSIIndexerError("Cannot create system bucket/scope/collection")
	}

	_, er = indexer3.IndexByName(_CBO_STATS_PRIMARY_INDEX)
	if er != nil {
		if er.Code() != 12016 {
			// only ignore index not found error
			return er
		}

		_, er = indexer3.CreatePrimaryIndex3(requestId, _CBO_STATS_PRIMARY_INDEX, nil, nil)
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

	sysBucket, er := defaultPool.BucketByName(_N1QL_SYSTEM_BUCKET)
	if er != nil {
		return false, er
	}

	sysScope, er := sysBucket.ScopeByName(_N1QL_SYSTEM_SCOPE)
	if er != nil {
		return false, er
	}

	cboStats, er := sysScope.KeyspaceByName(_N1QL_CBO_STATS)
	if er != nil {
		return false, er
	}

	return (cboStats != nil), nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", _N1QL_SYSTEM_BUCKET, _N1QL_SYSTEM_SCOPE, _N1QL_CBO_STATS)
}
