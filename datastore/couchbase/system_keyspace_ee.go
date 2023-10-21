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
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

const (
	_N1QL_SYSTEM_BUCKET       = dictionary.N1QL_SYSTEM_BUCKET
	_N1QL_SYSTEM_SCOPE        = dictionary.N1QL_SYSTEM_SCOPE
	_N1QL_CBO_STATS           = dictionary.N1QL_CBO_STATS
	_CBO_STATS_PRIMARY_INDEX  = dictionary.CBO_STATS_PRIMARY_INDEX
	_BUCKET_SYSTEM_SCOPE      = dictionary.BUCKET_SYSTEM_SCOPE
	_BUCKET_SYSTEM_COLLECTION = dictionary.BUCKET_SYSTEM_COLLECTION
	_BUCKET_SYSTEM_PRIM_INDEX = dictionary.BUCKET_SYSTEM_PRIM_INDEX
)

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	dPool, er := s.NamespaceByName("default") // so we're using the cached namespace always
	if er != nil {
		return er
	}
	defaultPool := dPool.(*namespace)

	// create/get system bucket/scope/collection
	sysBucket, er := defaultPool.keyspaceByName(_N1QL_SYSTEM_BUCKET)
	if er != nil {
		// only ignore bucket/keyspace not found error
		if er.Code() != errors.E_CB_KEYSPACE_NOT_FOUND && er.Code() != errors.E_CB_BUCKET_NOT_FOUND {
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
		if er.Code() != errors.E_CB_SCOPE_NOT_FOUND {
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
			} else if er != nil && er.Code() != errors.E_CB_SCOPE_NOT_FOUND {
				return er
			}
		}
		if sysScope == nil {
			return errors.NewCbBucketCreateScopeError(_N1QL_SYSTEM_BUCKET+"."+_N1QL_SYSTEM_SCOPE, nil)
		}
	}

	cboStats, er := sysScope.KeyspaceByName(_N1QL_CBO_STATS)
	if er != nil {
		if er.Code() != errors.E_CB_KEYSPACE_NOT_FOUND {
			// only ignore keyspace not found error
			return er
		}

		// allow "already exists" error in case of duplicated Create call
		er = sysScope.CreateCollection(_N1QL_CBO_STATS, nil)
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
			} else if er != nil && er.Code() != errors.E_CB_KEYSPACE_NOT_FOUND {
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
		if !errors.IsIndexNotFoundError(er) {
			// only ignore index not found error
			return er
		}

		er = s.CreateSysPrimaryIndex(_CBO_STATS_PRIMARY_INDEX, requestId, indexer3)
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

func (s *store) CreateSysPrimaryIndex(idxName, requestId string, indexer3 datastore.Indexer3) errors.Error {
	// if not serverless, get number of index nodes in the cluster, and create the primary index
	// with replicas in the following fashion:
	//    numIndexNode >= 4    ==> num_replica = 2
	//    numIndexNode >  1    ==> num_replica = 1
	//    numIndexNode == 1    ==> no replica
	// for serverless, the number of replica is determined automatically
	var with value.Value
	var replica map[string]interface{}
	num_replica := 0
	if !tenant.IsServerless() {
		numIndexNode, errs := s.getNumIndexNode()
		if len(errs) > 0 {
			return errs[0]
		}

		if numIndexNode >= 4 {
			num_replica = 2
		} else if numIndexNode > 1 {
			num_replica = 1
		}
		if num_replica > 0 {
			replica = make(map[string]interface{}, 1)
			replica["num_replica"] = num_replica
			with = value.NewValue(replica)
		}
	}

	_, er := indexer3.CreatePrimaryIndex3(requestId, idxName, nil, with)
	if er != nil && !errors.IsIndexExistsError(er) {
		// if the create failed due to not enough indexer nodes, retry with
		// less number of replica
		for num_replica > 0 {
			// defined as ErrNotEnoughIndexers in indexing/secondary/common/const.go
			if !er.ContainsText("not enough indexer nodes to create index with replica") {
				return er
			}

			num_replica--
			if num_replica == 0 {
				with = nil
			} else {
				replica["num_replica"] = num_replica
				with = value.NewValue(replica)
			}

			// retry with less number of replica
			_, er = indexer3.CreatePrimaryIndex3(requestId, idxName, nil, with)
			if er == nil || errors.IsIndexExistsError(er) {
				break
			}
		}
		if er != nil && !errors.IsIndexExistsError(er) {
			return er
		}
	}

	var sysIndex datastore.Index
	maxRetry := 8
	if idxName == _BUCKET_SYSTEM_PRIM_INDEX {
		maxRetry = 10
	}
	interval := 250 * time.Millisecond
	for i := 0; i < maxRetry; i++ {
		time.Sleep(interval)
		interval *= 2

		er = indexer3.Refresh()
		if er != nil {
			return er
		}
		sysIndex, er = indexer3.IndexByName(idxName)
		if sysIndex != nil {
			state, _, err1 := sysIndex.State()
			if err1 != nil {
				return err1
			}
			if state == datastore.ONLINE {
				break
			}
		} else if er != nil && !errors.IsIndexNotFoundError(er) {
			return er
		}
	}

	return er
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

func (s *store) GetSystemCollection(bucketName string) (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", bucketName, _BUCKET_SYSTEM_SCOPE, _BUCKET_SYSTEM_COLLECTION)
}

func (s *store) getNumIndexNode() (int, errors.Errors) {
	info := s.Info()
	nodes, errs := info.Topology()
	if len(errs) > 0 {
		return 0, errs
	}

	numIndexNode := 0
	for _, node := range nodes {
		nodeServices, errs := info.Services(node)
		if len(errs) > 0 {
			return 0, errs
		}
		// the nodeServices should have an element named "services" which is
		// an array of service names on that node, e.g. ["n1ql", "kv", "index"]
		if services, ok := nodeServices["services"]; ok {
			if serviceArr, ok := services.([]interface{}); ok {
				for _, serv := range serviceArr {
					if name, ok := serv.(string); ok {
						if name == "index" {
							numIndexNode++
						}
					}
				}
			}
		}
	}

	return numIndexNode, nil
}

// check for existance of system collection, and create primary index if necessary
func (s *store) CheckSystemCollection(bucketName, requestId string) errors.Error {
	sysColl, err := s.GetSystemCollection(bucketName)
	if err != nil {
		// make sure the bucket exists before we wait (e.g. index advisor)
		switch err.Code() {
		case errors.E_CB_KEYSPACE_NOT_FOUND, errors.E_CB_BUCKET_NOT_FOUND:
			defaultPool, er := s.NamespaceByName("default")
			if er != nil {
				return er
			}

			_, er = defaultPool.BucketByName(bucketName)
			if er != nil {
				return er
			}
		case errors.E_CB_SCOPE_NOT_FOUND:
			// no-op, ignore
		default:
			return err
		}
		// wait for system collection to show up
		maxRetry := 8
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			switch err.Code() {
			case errors.E_CB_KEYSPACE_NOT_FOUND, errors.E_CB_BUCKET_NOT_FOUND, errors.E_CB_SCOPE_NOT_FOUND:
				// no-op, ignore these errors
			default:
				return err
			}

			time.Sleep(interval)
			interval *= 2

			sysColl, err = s.GetSystemCollection(bucketName)
			if sysColl != nil || err == nil {
				break
			}
		}
		if err != nil {
			return err
		} else if sysColl == nil {
			return errors.NewSystemCollectionError("System collection not available for bucket "+bucketName, nil)
		}
	}

	if requestId == "" {
		return nil
	}

	indexer, er := sysColl.Indexer(datastore.GSI)
	if er != nil {
		return er
	}

	indexer3, ok := indexer.(datastore.Indexer3)
	if !ok {
		return errors.NewInvalidGSIIndexerError("Cannot get primary index on system collection")
	}

	sysIndex, er := indexer3.IndexByName(_BUCKET_SYSTEM_PRIM_INDEX)
	if er != nil {
		if !errors.IsIndexNotFoundError(er) {
			// only ignore index not found error
			return er
		}

		// create primary index on system collection if not already exists
		// the create function waits for ONLINE state before it returns
		er = s.CreateSysPrimaryIndex(_BUCKET_SYSTEM_PRIM_INDEX, requestId, indexer3)
		if er != nil && !errors.IsIndexExistsError(er) {
			// ignore index already exist error
			return er
		}
	} else {
		// make sure the primary index is ONLINE
		maxRetry := 10
		interval := 250 * time.Millisecond
		for i := 0; i < maxRetry; i++ {
			state, _, er1 := sysIndex.State()
			if er1 != nil {
				return er1
			}
			if state == datastore.ONLINE {
				break
			} else if state == datastore.DEFERRED {
				// build system index if it is deferred (e.g. just restored)
				er = indexer3.BuildIndexes(requestId, sysIndex.Name())
				if er != nil {
					return er
				}
			}

			time.Sleep(interval)
			interval *= 2

			er = indexer3.Refresh()
			if er != nil {
				return er
			}

			sysIndex, er = indexer3.IndexByName(_BUCKET_SYSTEM_PRIM_INDEX)
			if er != nil {
				return er
			}
		}
	}

	return er
}
