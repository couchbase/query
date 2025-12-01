// Copyright 2020-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

//go:build enterprise

package couchbase

import (
	"time"

	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	cb "github.com/couchbase/query/primitives/couchbase"
)

const (
	_N1QL_SYSTEM_BUCKET      = dictionary.N1QL_SYSTEM_BUCKET
	_N1QL_SYSTEM_SCOPE       = dictionary.N1QL_SYSTEM_SCOPE
	_N1QL_CBO_STATS          = dictionary.N1QL_CBO_STATS
	_CBO_STATS_PRIMARY_INDEX = dictionary.CBO_STATS_PRIMARY_INDEX
	_QUERY_METADATA_BUCKET   = dictionary.QUERY_METADATA_BUCKET
)

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	return s.createSysCollection(_N1QL_SYSTEM_BUCKET, _N1QL_SYSTEM_SCOPE, _N1QL_CBO_STATS, _CBO_STATS_PRIMARY_INDEX, requestId)
}

func (s *store) CreateQueryMetadata(requestId string) errors.Error {
	return s.createSysCollection(_QUERY_METADATA_BUCKET, _BUCKET_SYSTEM_SCOPE, _BUCKET_SYSTEM_COLLECTION, "", requestId)
}

func (s *store) createSysCollection(bucketName, scopeName, collectionName, indexName, requestId string) errors.Error {
	dPool, er := s.NamespaceByName("default") // so we're using the cached namespace always
	if er != nil {
		return er
	}
	defaultPool := dPool.(*namespace)

	// create/get system bucket/scope/collection
	sysBucket, er := defaultPool.keyspaceByName(bucketName)
	if er != nil {
		// only ignore bucket/keyspace not found error
		if er.Code() != errors.E_CB_KEYSPACE_NOT_FOUND && er.Code() != errors.E_CB_BUCKET_NOT_FOUND {
			return er
		}

		// create bucket
		_, err := cb.GetSystemBucket(&s.client, defaultPool.cbNamespace, bucketName)
		if err != nil {
			return errors.NewCbCreateSystemBucketError(bucketName, err)
		}

		// no need for a retry loop, cb.GetSystemBucket() call above should
		// have made sure that BucketMap is updated already
		defaultPool.refresh()
		sysBucket, er = defaultPool.keyspaceByName(bucketName)
		if er != nil {
			return er
		}
	}

	sysScope, er := sysBucket.ScopeByName(scopeName)
	if er != nil {
		if er.Code() != errors.E_CB_SCOPE_NOT_FOUND {
			// only ignore scope not found error
			return er
		}

		// _system scope automatically created
		if scopeName != _BUCKET_SYSTEM_SCOPE {
			// allow "already exists" error in case of duplicated Create call
			er = sysBucket.CreateScope(scopeName)
			if er != nil && !cb.AlreadyExistsError(er) {
				return er
			}
		}

		// retry till we have the newly created scope available
		maxRetry := 8
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			time.Sleep(interval)
			interval *= 2

			// reload sysBucket
			sysBucket.setNeedsManifest()
			sysBucket, er = defaultPool.keyspaceByName(bucketName)
			if er != nil {
				return er
			}

			sysScope, er = sysBucket.ScopeByName(scopeName)
			if sysScope != nil {
				break
			} else if er != nil && er.Code() != errors.E_CB_SCOPE_NOT_FOUND {
				return er
			}
		}
		if sysScope == nil {
			return errors.NewCbBucketCreateScopeError(bucketName+"."+scopeName, nil)
		}
	}

	sysCollection, er := sysScope.KeyspaceByName(collectionName)
	if er != nil {
		if er.Code() != errors.E_CB_KEYSPACE_NOT_FOUND {
			// only ignore keyspace not found error
			return er
		}

		// _query collection automatically created
		if collectionName != _BUCKET_SYSTEM_COLLECTION {
			// allow "already exists" error in case of duplicated Create call
			er = sysScope.CreateCollection(collectionName, nil)
			if er != nil && !cb.AlreadyExistsError(er) {
				return er
			}
		}

		// retry till we have the newly created collection available
		maxRetry := 8
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			time.Sleep(interval)
			interval *= 2

			// reload sysBucket
			sysBucket.setNeedsManifest()
			sysBucket, er = defaultPool.keyspaceByName(bucketName)
			if er != nil {
				return er
			}

			sysScope, er = sysBucket.ScopeByName(scopeName)
			if er != nil {
				return er
			}

			sysCollection, er = sysScope.KeyspaceByName(collectionName)
			if sysCollection != nil {
				break
			} else if er != nil && er.Code() != errors.E_CB_KEYSPACE_NOT_FOUND {
				return er
			}
		}
		if sysCollection == nil {
			return errors.NewCbBucketCreateCollectionError(bucketName+"."+scopeName+"."+collectionName, nil)
		}
	}

	// if no index requested
	if indexName == "" {
		return nil
	}

	// create primary index
	// make sure we have indexer3 first
	indexer, er := sysCollection.Indexer(datastore.GSI)
	if er != nil {
		return er
	}

	indexer3, ok := indexer.(datastore.Indexer3)
	if !ok {
		cb.DropSystemBucket(&s.client, bucketName)
		return errors.NewInvalidGSIIndexerError("Cannot create system bucket/scope/collection")
	}

	_, er = indexer3.IndexByName(indexName)
	if er != nil {
		if !errors.IsIndexNotFoundError(er) {
			// only ignore index not found error
			return er
		}

		er = s.CreateSysPrimaryIndex(indexName, requestId, indexer3)
		if er != nil {
			return er
		}
	}

	return nil
}

func (s *store) HasSystemCBOStats() (bool, errors.Error) {
	defaultPool, er := s.NamespaceByName("default") // so we're using the cached namespace always
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

func (s *store) DropSystemCBOStats() errors.Error {
	err := cb.DropSystemBucket(&s.client, _N1QL_SYSTEM_BUCKET)
	if err != nil {
		return errors.NewCbDropSystemBucketError(_N1QL_SYSTEM_BUCKET, err)
	}
	return nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", _N1QL_SYSTEM_BUCKET, _N1QL_SYSTEM_SCOPE, _N1QL_CBO_STATS)
}

func (s *store) HasQueryMetadata() (bool, errors.Error) {
	defaultPool, er := s.NamespaceByName("default") // so we're using the cached namespace always
	if er != nil {
		return false, er
	}

	sysBucket, er := defaultPool.BucketByName(_QUERY_METADATA_BUCKET)
	if er != nil {
		return false, er
	}

	sysScope, er := sysBucket.ScopeByName(_BUCKET_SYSTEM_SCOPE)
	if er != nil {
		return false, er
	}

	queryMetadata, er := sysScope.KeyspaceByName(_BUCKET_SYSTEM_COLLECTION)
	if er != nil {
		return false, er
	}

	return (queryMetadata != nil), nil
}

func (s *store) GetQueryMetadata() (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", _QUERY_METADATA_BUCKET, _BUCKET_SYSTEM_SCOPE, _BUCKET_SYSTEM_COLLECTION)
}
